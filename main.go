package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"

	"google.golang.org/grpc"
)

type Peer struct {
	id   int
	addr string
	conn *grpc.ClientConn
	stub RaftServiceClient
}

func NewPeer(id int, addr string) *Peer {
	// try connect to peer
	conn, err := grpc.NewClient(addr)
	if err != nil {
		slog.Error("Could not connect to peer")
		// need to keep trying in background
	}
	client := NewRaftServiceClient(conn)

	return &Peer{
		id:   id,
		addr: addr,
		conn: conn,
		stub: client,
	}
}

type Node struct {
	state  *State
	peers  map[int]Peer
	nodeId int
	UnimplementedRaftServiceServer
}

func NewNode(peers map[int]string, id int) (*Node, error) {
	state, err := InitState()
	if err != nil {
		return nil, err
	}

	return &Node{
		state:  state,
		peers:  peers,
		nodeId: id,
	}, nil
}

func (n *Node) Start() {

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

	peers := make(map[int]string)
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

		peers[nodeId] = splitNode[1]
	}
	fmt.Println(peers)

	if nodePort == "" {
		panic("this node doesnt exist in cluster")
	}

	node, err := NewNode(peers, *nodePtr)
	lis, err := net.Listen("tcp", ":"+nodePort)

	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	RegisterRaftServiceServer(grpcServer, node)

	fmt.Println("Listening on port:" + nodePort)
	grpcServer.Serve(lis)
}
