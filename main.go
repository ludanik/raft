package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// make the log first ezpz
type ServerState int

const (
	FOLLOWER ServerState = iota
	CANDIDATE
	LEADER
)

type LogEntry struct {
	term    int
	command string
}

type PersistentConfig struct {
	log         []LogEntry
	currentTerm int
	votedFor    string
}

func LoadPersistentConfig() (PersistentConfig, error) {
	file, err := os.ReadFile("log")
	if err != nil {
		// probably because it doesnt exist
		// return nil and create it when u save log
		return PersistentConfig{}, err
	}

	// first line is (currentTerm, votedFor)
	// every line after is the log
	// save in one file cuz im lazy
	lines := strings.Split(string(file), "\n")

	valLine := strings.Split(lines[0], ",")
	currentTerm, err := strconv.Atoi(valLine[0])
	if err != nil {
		return PersistentConfig{}, err
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
			return PersistentConfig{}, err
		}

		entry := LogEntry{term, splitLine[1]}
		entries[idx-1] = entry
	}

	return PersistentConfig{
		log:         entries,
		currentTerm: currentTerm,
		votedFor:    votedFor,
	}, nil
}

type LeaderConfig struct {
	nextIndex  int
	matchIndex int
}

type Config struct {
	persistentCfg PersistentConfig
	leaderCfg     LeaderConfig

	commitIndex int
	lastApplied int
	state       ServerState
}

func NewConfig() (Config, error) {
	pCfg, err := LoadPersistentConfig()
	if err != nil {
		return Config{}, err
	}

	leaderCfg := LeaderConfig{0, 0}

	return Config{
		persistentCfg: pCfg,
		leaderCfg:     leaderCfg,
		state:         FOLLOWER,
	}, nil
}

// only need to save log, currentTerm, votedFor
func SaveConfig() error {
	return nil
}

func main() {
	config, err := NewConfig()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(config.persistentCfg.log)
}
