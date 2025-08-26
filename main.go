package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// make the log first ezpz
type Role int

const (
	FOLLOWER Role = iota
	CANDIDATE
	LEADER
)

type LogEntry struct {
	term    int
	command string
}

type PersistentState struct {
	log         []LogEntry
	currentTerm int
	votedFor    string
}

// load persistent state
func LoadState() (*PersistentState, error) {
	file, err := os.ReadFile("log")
	if err != nil {
		// probably because it doesnt exist
		// return empty persistentState and
		// and save it when u save log
		entries := make([]LogEntry, 0)

		return &PersistentState{
			log:         entries,
			currentTerm: 0,
			votedFor:    "",
		}, nil
	}

	// first line is (currentTerm, votedFor)
	// every line after is the log
	// save in one file cuz im lazy
	lines := strings.Split(string(file), "\n")

	valLine := strings.Split(lines[0], ",")
	currentTerm, err := strconv.Atoi(valLine[0])
	if err != nil {
		return nil, err
	}

	votedFor := valLine[1]
	fmt.Println("currTerm votedFor", currentTerm, votedFor)
	entries := make([]LogEntry, len(lines)-1)

	for idx := 1; idx < len(lines); idx++ {
		if (len(lines[idx]) < 1) || (lines[idx] == "") {
			continue
		}

		splitLine := strings.Split(lines[idx], ",")
		term, err := strconv.Atoi(splitLine[0])
		if err != nil {
			return nil, err
		}

		entry := LogEntry{term, splitLine[1]}
		entries[idx-1] = entry
	}

	return &PersistentState{
		log:         entries,
		currentTerm: currentTerm,
		votedFor:    votedFor,
	}, nil
}

// only need to save log, currentTerm, votedFor
func SaveState(state *State) error {
	// just overwrite file for now
	currentTerm := state.persistentState.currentTerm
	votedFor := state.persistentState.votedFor
	log := state.persistentState.log

	line1 := fmt.Sprintf("%d,%s\n", currentTerm, votedFor)

	file, err := os.Create("log")
	if err != nil {
		return err
	}

	w := bufio.NewWriter(file)
	_, err = w.WriteString(line1)
	if err != nil {
		return err
	}

	var str string
	for idx, entry := range log {
		if idx == len(log)-1 {
			str = fmt.Sprintf("%d,%s", entry.term, entry.command)
		} else {
			str = fmt.Sprintf("%d,%s\n", entry.term, entry.command)
		}
		_, err = w.WriteString(str)
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	return nil
}

type LeaderState struct {
	nextIndex  int
	matchIndex int
}

type State struct {
	persistentState *PersistentState
	leaderState     LeaderState

	commitIndex int
	lastApplied int
	role        Role
}

func InitState() (*State, error) {
	pState, err := LoadState()
	if err != nil {
		return nil, err
	}

	lState := LeaderState{0, 0}

	return &State{
		persistentState: pState,
		leaderState:     lState,
		role:            FOLLOWER,
	}, nil
}

type Node struct {
	state   *State
	cluster map[int]string
	nodeId  int
	UnimplementedRaftServiceServer
}

func NewNode(cluster map[int]string, id int) (*Node, error) {
	state, err := InitState()
	if err != nil {
		return nil, err
	}

	return &Node{
		state:   state,
		cluster: cluster,
		nodeId:  id,
	}, nil
}

// this is for receiving RequestVote from another candidate node
func (n *Node) RequestVote(context.Context, *RequestVoteMessage) (*RequestVoteReply, error) {
	return &RequestVoteReply{
		Term:        1,
		VoteGranted: false,
	}, nil
}

// this is for receiving AppendEntries from a leader node
func (n *Node) AppendEntries(context.Context, *AppendEntriesMessage) (*AppendEntriesReply, error) {
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
	for _, node := range nodes {
		splitNode := strings.Split(node, ":")
		nodeId, err := strconv.Atoi(splitNode[0])
		if err != nil {
			panic("Unable to read cluster passed in args")
		}
		cluster[nodeId] = splitNode[1]
	}

	node, err := NewNode(cluster, *nodePtr)
	if err != nil {
		panic("unable to start node")
	}

	lis, err := net.Listen("tcp", ":9001")

	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	RegisterRaftServiceServer(grpcServer, node)

	go grpcServer.Serve(lis)

	conn, err := grpc.NewClient("localhost:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	c := NewRaftServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	test, err := c.RequestVote(ctx, &RequestVoteMessage{})
	if err != nil {
		panic(err)
	}
	fmt.Println(test)
}
