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

	// Get last log entry's term and index
	lastLogIdx := int32(len(n.log) - 1)
	lastLogTerm := n.log[lastLogIdx].term

	// "candidate's log is at least as complete as local log"
	// According to Raft: candidate's log is up-to-date if:
	// - last log term is greater, OR
	// - last log terms are equal AND last log index is greater or equal
	logIsUpToDate := (msg.LastLogTerm > lastLogTerm) || 
		(msg.LastLogTerm == lastLogTerm && msg.LastLogIndex >= lastLogIdx)

	if sameTerm && validVote && logIsUpToDate {
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

	// Update leader node ID
	n.leaderNodeId = msg.LeaderId

	// Check if log contains an entry at prevLogIndex with matching term
	if int(msg.PrevLogIndex) >= len(n.log) || n.log[msg.PrevLogIndex].term != msg.PrevLogTerm {
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
	// If an existing entry conflicts with a new one (same index but different terms),
	// delete the existing entry and all that follow it
	for idx, entry := range msg.Entries {
		logIdx := int(msg.PrevLogIndex) + 1 + idx
		
		// if log doesn't have entry at this index, append it
		if logIdx >= len(n.log) {
			n.log = append(n.log, LogEntry{term: entry.Term, command: entry.Command})
		} else if n.log[logIdx].term != entry.Term {
			// conflict: delete existing entry and all that follow
			n.log = n.log[:logIdx]
			n.log = append(n.log, LogEntry{term: entry.Term, command: entry.Command})
		}
		// else: entry matches, continue
	}

	// update commit index
	if msg.CommitIndex > n.commitIndex {
		// commitIndex should be min(leaderCommit, index of last new entry)
		lastNewEntryIdx := int32(len(n.log) - 1)
		if msg.CommitIndex < lastNewEntryIdx {
			n.commitIndex = msg.CommitIndex
		} else {
			n.commitIndex = lastNewEntryIdx
		}
	}

	// save state after modifying log
	n.SavePersistentState()

	return &AppendEntriesReply{
		Term:    msg.Term,
		Success: true,
	}, nil
}
