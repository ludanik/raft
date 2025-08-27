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
	"time"

	"google.golang.org/grpc"
)

const (
	DEFAULT_TIMEOUT_MS = 500
)

type Peer struct {
	id   int32
	addr string
	conn *grpc.ClientConn
	stub RaftServiceClient
}

func NewPeer(id int32, addr string) *Peer {
	// only try to connect to peer when u are candidate
	return &Peer{
		id:   id,
		addr: addr,
		conn: nil,
		stub: nil,
	}
}

func (p *Peer) Connect() error {
	conn, err := grpc.NewClient(p.addr)
	if err != nil {
		slog.Error("Could not connect to peer", err)
		return err
	}

	client := NewRaftServiceClient(conn)
	p.conn = conn
	p.stub = client
	return nil
}

type Node struct {
	state  *State
	peers  map[int32]*Peer
	nodeId int32

	electionTriggerCh chan bool
	stepDownCh        chan bool
	resetTimeoutCh    chan bool

	UnimplementedRaftServiceServer
}

func NewNode(cluster map[int]string, id int) (*Node, error) {
	state, err := InitState()
	if err != nil {
		return nil, err
	}

	peers := make(map[int32]*Peer)
	for key, value := range cluster {
		peers[int32(key)] = NewPeer(int32(key), value)
	}

	return &Node{
		state:          state,
		peers:          peers,
		nodeId:         int32(id),
		stepDownCh:     make(chan bool),
		resetTimeoutCh: make(chan bool, 2),
	}, nil
}

func (n *Node) Start() {
	go n.Loop()
	go n.TimeoutLoop()
	for {
		switch role := n.state.role; role {
		case FOLLOWER:

		case CANDIDATE:

		case LEADER:

		}
	}
}

func (n *Node) TimeoutLoop() {
	r := rand.New(rand.NewSource(int64(time.Now().Second())))
	for {
		if n.state.role == FOLLOWER {
			timeout := (r.Int() % 2) * 2 * DEFAULT_TIMEOUT_MS
			fmt.Println("timeout for ", time.Millisecond*time.Duration(timeout))
			select {
			case <-n.resetTimeoutCh:
				continue
			case <-time.After(time.Millisecond * time.Duration(timeout)):
				fmt.Println("node timed out waiting for dear leader")
				n.electionTriggerCh <- true
			}
		}
	}
}

func (n *Node) StandForElection() {
	n.state.
}

func (n *Node) Loop() {
	for {
		select {
		case <-n.stepDownCh:
			n.state.role = FOLLOWER
		case <-n.electionTriggerCh:
			n.StandForElection()
		}
	}
}

// this is for receiving RequestVote from another candidate node
func (n *Node) RequestVote(ctx context.Context, msg *RequestVoteMessage) (*RequestVoteReply, error) {
	if msg.Term > n.state.persistentState.currentTerm {
		n.state.persistentState.currentTerm = msg.Term
		n.stepDownCh <- true
	} else if msg.Term < n.state.persistentState.currentTerm {
		return &RequestVoteReply{
			Term:        n.state.persistentState.currentTerm,
			VoteGranted: false,
		}, nil
	}

	votedFor := n.state.persistentState.votedFor

	sameTerm := msg.Term == n.state.persistentState.currentTerm
	validVote := (votedFor == msg.CandidateId) || (votedFor == -1)

	lastLogTermOfServer := n.state.persistentState.log[len(n.state.persistentState.log)-1].term
	lastLogIdxOfServer := int32(len(n.state.persistentState.log) - 1)

	// "candidate's log is at least as complete as local log"
	// what does this mean?
	// candidateLogTerm >= serverLogTerm
	// candidateLogIdx >= serverLogIdx
	// this is probably wrong though
	validLogTerm := msg.LastLogTerm >= lastLogTermOfServer
	validLogIdx := msg.LastLogIndex >= lastLogIdxOfServer

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
