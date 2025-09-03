package main

import (
	"fmt"
	"net/http"
)

// POST an entry to leader node from curl or whatever
// if we are follower or candidate redirect to leader
func (n *Node) AddEntry(w http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(w, "hello\n")
}

// listen for client requests
// redirect to leader node if follower
func (n *Node) StartHttpServer() {
	http.HandleFunc("/addentry", n.AddEntry)
	http.ListenAndServe(":8090", nil)
}
