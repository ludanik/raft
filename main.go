package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

func LoadState() (*PersistentState, error) {
	file, err := os.ReadFile("log")
	if err != nil {
		// probably because it doesnt exist
		// return nil and create it when u save log
		return nil, err
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

func main() {
	state, err := InitState()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(state.persistentState.log)
}
