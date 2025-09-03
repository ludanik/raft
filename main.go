package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"

	"google.golang.org/grpc"
)

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
	go node.StartHttpServer()

	fmt.Println("Listening on port:" + nodePort)
	grpcServer.Serve(lis)
}
