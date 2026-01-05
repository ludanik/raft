package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type AddEntryRequest struct {
	Command string `json:"command"`
}

type AddEntryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Leader  int32  `json:"leader,omitempty"`
}

// POST an entry to leader node from curl or whatever
// if we are follower or candidate redirect to leader
func (n *Node) AddEntry(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(AddEntryResponse{
			Success: false,
			Message: "Only POST method is allowed",
		})
		return
	}

	n.mu.Lock()
	role := n.role
	leaderNodeId := n.leaderNodeId
	n.mu.Unlock()

	// If not leader, redirect to leader
	if role != LEADER {
		w.WriteHeader(http.StatusTemporaryRedirect)
		response := AddEntryResponse{
			Success: false,
			Message: "Not the leader",
			Leader:  leaderNodeId,
		}
		
		if leaderNodeId == 0 {
			response.Message = "No leader elected yet, please try again"
		} else {
			response.Message = fmt.Sprintf("Not the leader. Redirect to node %d", leaderNodeId)
		}
		
		json.NewEncoder(w).Encode(response)
		return
	}

	// Parse request body
	var entryReq AddEntryRequest
	if err := json.NewDecoder(req.Body).Decode(&entryReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AddEntryResponse{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	if entryReq.Command == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(AddEntryResponse{
			Success: false,
			Message: "Command cannot be empty",
		})
		return
	}

	// Send the command to the leader's newEntryCh
	entry := LogEntry{
		term:    n.currentTerm,
		command: entryReq.Command,
	}

	select {
	case n.newEntryCh <- entry:
		slog.Info("Received new command", "command", entryReq.Command)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AddEntryResponse{
			Success: true,
			Message: "Command accepted",
		})
	default:
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(AddEntryResponse{
			Success: false,
			Message: "Leader is busy, please try again",
		})
	}
}

// listen for client requests
// redirect to leader node if follower
func (n *Node) StartHttpServer() {
	http.HandleFunc("/addentry", n.AddEntry)
	slog.Info("Starting HTTP server on :8090")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		slog.Error("HTTP server failed", "error", err)
	}
}
