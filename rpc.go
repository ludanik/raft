package main

import "context"

// this is for receiving RequestVote from another candidate node
func (n *Node) RequestVote(ctx context.Context, msg *RequestVoteMessage) (*RequestVoteReply, error) {
	// TODO: fix race conditions

	if msg.Term > n.currentTerm {
		n.currentTerm = msg.Term
		n.stepDownCh <- true
	} else if msg.Term < n.currentTerm {
		// tell client to step down
		return &RequestVoteReply{
			Term:        n.currentTerm,
			VoteGranted: false,
		}, nil
	}

	// if we already voted just return
	if n.votedFor != -1 {
		return &RequestVoteReply{
			Term:        n.currentTerm,
			VoteGranted: false,
		}, nil
	}

	sameTerm := msg.Term == n.currentTerm
	validVote := (n.votedFor == msg.CandidateId) || (n.votedFor == -1)

	var lastLogTerm int32
	var lastLogIdx int32
	// this isn't right
	// but i will leave it here for now bcoz i wanna test election
	// TODO: do this more intelligently
	if len(n.log) == 0 {
		lastLogTerm = 0
		lastLogIdx = 0
	} else {
		lastLogTerm = n.log[len(n.log)-1].term
		lastLogIdx = int32(len(n.log) - 1)
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
		n.votedFor = msg.CandidateId
		n.SavePersistentState()

		return &RequestVoteReply{
			Term:        n.currentTerm,
			VoteGranted: true,
		}, nil
	} else {
		return &RequestVoteReply{
			Term:        n.currentTerm,
			VoteGranted: false,
		}, nil
	}
}

// this is for receiving AppendEntries from a leader node
func (n *Node) AppendEntries(ctx context.Context, msg *AppendEntriesMessage) (*AppendEntriesReply, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if msg.Term < n.currentTerm {
		return &AppendEntriesReply{
			Term:    n.currentTerm,
			Success: false,
		}, nil
	}

	if n.log[msg.PrevLogIndex].term != msg.PrevLogTerm {
		return &AppendEntriesReply{
			Term:    msg.Term,
			Success: false,
		}, nil
	}

	// only reset timeout if valid rpc ? doesn't make sense to reset if it's invalid
	n.resetTimeoutCh <- true

	// heartbeat
	if msg.Entries == nil {
		return &AppendEntriesReply{
			Term:    msg.Term,
			Success: true,
		}, nil
	}

	// append entries

	for idx, entry := range msg.Entries {

	}

	return &AppendEntriesReply{
		Term:    msg.Term,
		Success: true,
	}, nil
}
