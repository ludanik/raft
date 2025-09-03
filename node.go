package main

import (
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"
)

const (
	DEFAULT_TIMEOUT_MIN_MS = 1000
)

type Role int

const (
	FOLLOWER Role = iota
	CANDIDATE
	LEADER
)

type LogEntry struct {
	term    int32
	command string
}

type Node struct {
	/* PERSISTENT STATE */
	log         []LogEntry
	currentTerm int32
	votedFor    int32

	/* VOLATILE STATE */
	commitIndex int32
	lastApplied int32
	role        Role

	/* LEADER STATE */
	nextIndex  int32
	matchIndex int32

	/* IMPLEMENTATION DETAILS */
	mu sync.Mutex

	peers        map[int32]*Peer
	nodeId       int32
	leaderNodeId int32

	stepDownCh     chan bool
	resetTimeoutCh chan bool
	newEntryCh     chan LogEntry

	UnimplementedRaftServiceServer
}

func NewNode(cluster map[int]string, id int) (*Node, error) {
	peers := make(map[int32]*Peer)
	for key, value := range cluster {
		peers[int32(key)] = NewPeer(int32(key), "localhost:"+value)
	}

	n := Node{
		peers:          peers,
		nodeId:         int32(id),
		stepDownCh:     make(chan bool),
		resetTimeoutCh: make(chan bool, 2),
		newEntryCh:     make(chan LogEntry),
	}

	err := n.LoadPersistentState()
	if err != nil {
		log.Fatal("Could not load persistent state")
	}

	return &n, nil
}

func (n *Node) Start() {
	for {
		switch n.role {
		case FOLLOWER:
			n.RunFollower()
		case CANDIDATE:
			n.RunCandidate()
		case LEADER:
			n.RunLeader()
		}
	}
}

func (n *Node) RunFollower() {
	n.mu.Lock()
	defer n.mu.Unlock()

	timeout := GetTimeoutMs()

	fmt.Println("follower timeout for ", timeout)

	for {
		// timeout should be random value between T and 2T, I choose T to be whatever I defined above
		select {
		case <-n.resetTimeoutCh:
			continue
		case <-time.After(timeout):
			slog.Info("follower timed out waiting for dear leader")
			n.role = CANDIDATE
			return
		}
	}
}

func (n *Node) RunCandidate() {
	// stand for election
	// try connect to every client first
	// then try rpc to that client
	// if you get no rpc response keep trying for that node until u get majority or timeout
	// if you get AppendEntries rpc, stand down
	// check if your role was changed before appointing yourself leader
	n.mu.Lock()
	defer n.mu.Unlock()

	timeout := GetTimeoutMs()

	fmt.Println("timeout for ", timeout)

	electedCh := make(chan bool, 1)
	timeoutCh := make(chan bool, 1)

	go func() {
		n.currentTerm += 1
		// vote for self
		votes := 1
		// we need a majority, >50% of nodes need to give their vote
		votesNeeded := int(len(n.peers)/2) + 1

		lastLogTerm := n.currentTerm
		var lastLogIdx int32

		// this isn't right
		// but i will leave it here for now bcoz i wanna test election
		// TODO: do this more intelligently
		if len(n.log) == 0 {
			lastLogIdx = 0
		} else {
			lastLogIdx = int32(len(n.log) - 1)
		}

		slog.Info("Election started, trying to get votes", "timeout", timeout, "votesNeeded", votesNeeded, "term", lastLogTerm)

		for key := range n.peers {
			p := n.peers[key]
			err := p.Connect()
			if err != nil {
				slog.Error("Couldn't connect to peer", "addr", "err", n.peers[key].addr, err)
			}
		}

		// we need to make sure we can't get duplicate votes
		peerSet := make(map[int]bool)

		for {
			select {
			case <-timeoutCh:
				return
			default:
				for key := range n.peers {
					select {
					case <-timeoutCh:
						return
					default:
						p := n.peers[key]

						// skip if we already got a vote from this node
						if peerSet[int(p.id)] == true {
							continue
						}

						// send rpc to all
						msg := RequestVoteMessage{
							CandidateId:  n.nodeId,
							Term:         n.currentTerm,
							LastLogIndex: lastLogIdx,
							LastLogTerm:  lastLogTerm,
						}

						reply, err := p.RequestVoteFromPeer(&msg, timeout)
						if err != nil {
							slog.Error(err.Error())
							continue
						}

						slog.Info("response: ", "voteGranted", reply.VoteGranted)

						if reply.VoteGranted == true {
							peerSet[int(p.id)] = true
							votes += 1
							slog.Info("Candidate got vote from", "addr", p.addr)
						} else if reply.VoteGranted == false && reply.Term > n.currentTerm {
							// if response contains term higher than ours, we step down
							n.currentTerm = reply.Term
							n.role = FOLLOWER
							return
						} else if reply.VoteGranted == false {
							// if votegranted is false but term is not higher
							// that means this node already voted for someone else
							peerSet[int(p.id)] = true
						}
					}
					// check how many votes we got
					if votes == votesNeeded && n.role == CANDIDATE {
						electedCh <- true
						return
					}
				}
			}
		}
	}()

	for {
		select {
		case <-electedCh:
			// we successfully became leader
			n.role = LEADER
			n.votedFor = -1
			slog.Info("candidate became dear leader")
			return
		case <-n.stepDownCh:
			slog.Info("candidate received signal to step down")
			n.role = FOLLOWER
			n.votedFor = -1
			timeoutCh <- true
			return
		case <-time.After(timeout):
			slog.Info("candidate timed out waiting for election to complete")
			n.votedFor = -1
			timeoutCh <- true
			return
		}
	}
}

// TODO: implement simple REST api to give commands to the leader
func (n *Node) RunLeader() {
	timeout := time.Duration((DEFAULT_TIMEOUT_MIN_MS / 2) * time.Millisecond)

	heartBeatCh := make(chan bool)
	// this looks dumb but it helps
	stepDownCh := make(chan bool)
	go func() {
		for {
			select {
			case <-stepDownCh:
				return
			case <-heartBeatCh:
				for key := range n.peers {
					p := n.peers[key]

					// heartbeat is appendentries with empty log
					msg := AppendEntriesMessage{
						Term:         n.currentTerm,
						LeaderId:     n.nodeId,
						PrevLogIndex: int32(len(n.log) - 1),
						PrevLogTerm:  n.log[len(n.log)-1].term,
						CommitIndex:  n.commitIndex,
						Entries:      nil,
					}

					deadlineTimeout := timeout / 2

					reply, err := p.AppendEntriesToPeer(&msg, deadlineTimeout)
					if err != nil {
						slog.Error(err.Error())
						continue
					}

					slog.Info("response: ", "success", reply.Success, "term", reply.Term)

					if reply.Success == false && reply.Term > n.currentTerm {
						// step down
						n.role = FOLLOWER
						return
					}
				}
			case <-n.newEntryCh:

			}
		}
	}()

	for {
		select {
		case <-n.stepDownCh:
			stepDownCh <- true
			n.role = FOLLOWER
			return
		case <-time.After(timeout):
			heartBeatCh <- true
		}
	}
}
