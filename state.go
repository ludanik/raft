package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// load persistent state
func (n *Node) LoadPersistentState() error {
	file, err := os.ReadFile(fmt.Sprintf("log%d", n.nodeId))
	if err != nil {
		slog.Error("Log file not found")
		// probably because it doesnt exist
		// return empty persistentState and
		// and save it when u save log
		// Initialize with sentinel entry at index 0
		entries := []LogEntry{{term: 0, command: ""}}

		n.log = entries
		n.currentTerm = 0
		n.votedFor = -1

		return nil
	}

	// first line is (currentTerm, votedFor)
	// every line after is the log
	// save in one file cuz im lazy
	lines := strings.Split(string(file), "\n")

	valLine := strings.Split(lines[0], ",")
	currentTerm, err := strconv.Atoi(valLine[0])
	if err != nil {
		return err
	}

	votedFor, err := strconv.Atoi(valLine[1])
	if err != nil {
		return err
	}

	fmt.Println("currTerm votedFor", currentTerm, votedFor)
	
	// Initialize with sentinel entry at index 0 if no log entries
	entries := []LogEntry{{term: 0, command: ""}}

	for idx := 1; idx < len(lines); idx++ {
		if (len(lines[idx]) < 1) || (lines[idx] == "") {
			continue
		}

		splitLine := strings.Split(lines[idx], ",")
		term, err := strconv.Atoi(splitLine[0])
		if err != nil {
			return err
		}

		entry := LogEntry{term: int32(term), command: splitLine[1]}
		entries = append(entries, entry)
	}

	n.log = entries
	n.currentTerm = int32(currentTerm)
	n.votedFor = int32(votedFor)
	return nil
}

// only need to save log, currentTerm, votedFor
func (n *Node) SavePersistentState() error {
	// just overwrite file for now

	line1 := fmt.Sprintf("%d,%d\n", n.currentTerm, n.votedFor)

	file, err := os.Create(fmt.Sprintf("log%d", n.nodeId))
	if err != nil {
		return err
	}

	w := bufio.NewWriter(file)
	_, err = w.WriteString(line1)
	if err != nil {
		return err
	}

	log := n.log
	var str string
	// Skip sentinel entry at index 0 when saving
	for idx := 1; idx < len(log); idx++ {
		entry := log[idx]
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
