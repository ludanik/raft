package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

const (
	DEFAULT_TIMEOUT_MS = 2000
)

type Node struct {
	mu sync.Mutex

	state  *State
	peers  map[int32]*Peer
	nodeId int32

	stepDownCh     chan bool
	resetTimeoutCh chan bool

	UnimplementedRaftServiceServer
}

func NewNode(cluster map[int]string, id int) (*Node, error) {
	state, err := InitState()
	if err != nil {
		return nil, err
	}

	peers := make(map[int32]*Peer)
	for key, value := range cluster {
		peers[int32(key)] = NewPeer(int32(key), "localhost:"+value)
	}

	return &Node{
		state:          state,
		peers:          peers,
		nodeId:         int32(id),
		stepDownCh:     make(chan bool),
		resetTimeoutCh: make(chan bool, 2),
	}, nil
}

func (n *Node) RunFollower() {
	n.mu.Lock()
	defer n.mu.Unlock()

	r := rand.New(rand.NewSource(int64(time.Now().Second())))
	// max is 2T
	timeout := (r.Int()%2)*(DEFAULT_TIMEOUT_MS/4) + (r.Int()%2)*(DEFAULT_TIMEOUT_MS/4) + (r.Int()%2)*(DEFAULT_TIMEOUT_MS/2) + DEFAULT_TIMEOUT_MS
	fmt.Println("follower timeout for ", time.Millisecond*time.Duration(timeout))

	for {
		// timeout should be random value between T and 2T, I choose T to be whatever I defined above
		select {
		case <-n.resetTimeoutCh:
			continue
		case <-time.After(time.Millisecond * time.Duration(timeout)):
			slog.Info("follower timed out waiting for dear leader")
			n.state.role = CANDIDATE
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

	r := rand.New(rand.NewSource(int64(time.Now().Second())))
	// max is 2T
	timeout := (r.Int()%2)*(DEFAULT_TIMEOUT_MS/4) + (r.Int()%2)*(DEFAULT_TIMEOUT_MS/4) + (r.Int()%2)*(DEFAULT_TIMEOUT_MS/2) + DEFAULT_TIMEOUT_MS
	fmt.Println("timeout for ", time.Millisecond*time.Duration(timeout))

	electedCh := make(chan bool, 1)
	timeoutCh := make(chan bool, 1)

	go func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		// increment term each election
		n.state.persistentState.currentTerm += 1

		// vote for urself
		votes := 1
		// we need a majority, >50% of nodes need to give their vote
		votesNeeded := int(len(n.peers)/2) + 1

		lastLogTerm := n.state.persistentState.currentTerm
		var lastLogIdx int32

		// this isn't right
		// but i will leave it here for now bcoz i wanna test election
		// TODO: do this more intelligently
		if len(n.state.persistentState.log) == 0 {
			lastLogIdx = 0
		} else {
			lastLogIdx = int32(len(n.state.persistentState.log) - 1)
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
						Term:         n.state.persistentState.currentTerm,
						LastLogIndex: lastLogIdx,
						LastLogTerm:  lastLogTerm,
					}

					reply, err := p.RequestVoteFromPeer(&msg)
					if err != nil {
						slog.Error(err.Error())
						continue
					}

					if reply.VoteGranted == true {
						peerSet[int(p.id)] = true
						votes += 1
						slog.Info("Candidate got vote from", "addr", p.addr)
					} else if reply.VoteGranted == false && reply.Term > n.state.persistentState.currentTerm {
						// if response contains term higher than ours, we step down
						n.state.persistentState.currentTerm = reply.Term
						n.state.role = FOLLOWER
						return
					} else if reply.VoteGranted == false {
						// if votegranted is false but term is not higher
						// that means this node already voted for someone else
						peerSet[int(p.id)] = true
					}
				}
				// check how many votes we got
				if votes == votesNeeded && n.state.role == CANDIDATE {
					electedCh <- true
					return
				}
			}
		}
	}()

	select {
	case <-electedCh:
		// we successfully became leader
		n.state.role = LEADER
		slog.Info("candidate became dear leader")
		break
	case <-n.stepDownCh:
		slog.Info("candidate received signal to step down")
		n.state.role = FOLLOWER
		timeoutCh <- true
		break
	case <-time.After(time.Millisecond * time.Duration(timeout)):
		slog.Info("candidate timed out waiting for election to complete")
		timeoutCh <- true
		break
	}
}

func (n *Node) RunLeader() {
	for {
		// listen for messages from client
		// use appendentries to send heartbeat
		// use appendentries to replicate logs across client
	}
}

func (n *Node) Start() {
	for {
		switch n.state.role {
		case FOLLOWER:
			n.RunFollower()
		case CANDIDATE:
			n.RunCandidate()
		case LEADER:
			n.RunLeader()
		}
	}
}

// this is for receiving RequestVote from another candidate node
func (n *Node) RequestVote(ctx context.Context, msg *RequestVoteMessage) (*RequestVoteReply, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if msg.Term > n.state.persistentState.currentTerm {
		n.state.persistentState.currentTerm = msg.Term
		n.stepDownCh <- true
	} else if msg.Term < n.state.persistentState.currentTerm {
		// tell client to step down
		return &RequestVoteReply{
			Term:        n.state.persistentState.currentTerm,
			VoteGranted: false,
		}, nil
	}

	// if we already voted just return
	votedFor := n.state.persistentState.votedFor
	if votedFor != -1 {
		return &RequestVoteReply{
			Term:        n.state.persistentState.currentTerm,
			VoteGranted: false,
		}, nil
	}

	sameTerm := msg.Term == n.state.persistentState.currentTerm
	validVote := (votedFor == msg.CandidateId) || (votedFor == -1)

	var lastLogTerm int32
	var lastLogIdx int32
	// this isn't right
	// but i will leave it here for now bcoz i wanna test election
	// TODO: do this more intelligently
	if len(n.state.persistentState.log) == 0 {
		lastLogTerm = 0
		lastLogIdx = 0
	} else {
		lastLogTerm = n.state.persistentState.log[len(n.state.persistentState.log)-1].term
		lastLogIdx = int32(len(n.state.persistentState.log) - 1)
	}

	// "candidate's log is at least as complete as local log"
	// what does this mean?
	// candidateLogTerm >= serverLogTerm
	// candidateLogIdx >= serverLogIdx
	// this is probably wrong though
	validLogTerm := msg.LastLogTerm >= lastLogTerm
	validLogIdx := msg.LastLogIndex >= lastLogIdx

	if sameTerm && validVote && validLogIdx && validLogTerm {
		n.resetTimeoutCh <- true

		return &RequestVoteReply{
			Term:        n.state.persistentState.currentTerm,
			VoteGranted: true,
		}, nil
	} else {
		return &RequestVoteReply{
			Term:        n.state.persistentState.currentTerm,
			VoteGranted: false,
		}, nil
	}
}

// this is for receiving AppendEntries from a leader node
func (n *Node) AppendEntries(ctx context.Context, msg *AppendEntriesMessage) (*AppendEntriesReply, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	return &AppendEntriesReply{}, nil
}

func main() {
	// parse arguments
	// take list of machines from args
	// max 5 machines in raft cluster
	clusterPtr := flag.String("cluster", "", "Define Raft cluster. For example, in --cluster='1:3001,2:3002,3:3003', 1:3001 means node 1 corresponds to port 3001")
	nodePtr := flag.Int("node", -1, "Node number of this process. Must exist in the defined Raft cluster. Ex. for --cluster='1:3001,2:3002,3:3003', valid nodes are 1, 2 and 3.")
	flag.Parse()

	cluster := make(map[int]string)
	nodes := strings.Split(*clusterPtr, ",")
	nodePort := ""
	for _, node := range nodes {
		splitNode := strings.Split(node, ":")
		nodeId, err := strconv.Atoi(splitNode[0])
		if err != nil {
			panic("Unable to read cluster passed in args")
		}

		if nodeId == *nodePtr {
			nodePort = splitNode[1]
			continue
		}

		cluster[nodeId] = splitNode[1]
	}
	fmt.Println(cluster)

	if nodePort == "" {
		panic("this node doesnt exist in cluster")
	}

	node, err := NewNode(cluster, *nodePtr)
	lis, err := net.Listen("tcp", ":"+nodePort)

	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	RegisterRaftServiceServer(grpcServer, node)

	go node.Start()

	fmt.Println("Listening on port:" + nodePort)
	grpcServer.Serve(lis)
}
