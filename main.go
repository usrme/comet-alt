package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/exp/maps"
)

func main() {
	if err := checkGitInPath(); err != nil {
		fail("Error: %s", err)
	}

	gitRoot, err := findGitDir()
	if err != nil {
		fail("Error: %s", err)
	}

	if err := os.Chdir(gitRoot); err != nil {
		fail("Error: could not change directory: %s", err)
	}

	noAddedFiles, _ := noFilesInStaging()
	if noAddedFiles {
		fail("Error: no files added to staging")
	}

	prefixes, signOff, config, err := loadConfig()
	if err != nil {
		fail("Error: %s", err)
	}

	changedFiles, err := getChangedFiles()
	if err != nil {
		fail("Error: %s", err)
	}

	commitMessages := []string{}
	if len(os.Args) > 1 && os.Args[1] == "-m" {
		commitMessages, err = findCommitMessages(os.Args[2])
		if err != nil {
			fail("Error: %s", err)
		}
	}

	uniqueMessages := formUniqueMessages(commitMessages, config.FindAllCommitMessages)
	uniquePaths := formUniquePaths(changedFiles, config.ScopeCompletionOrder)
	m := newModel(prefixes, config, uniquePaths, uniqueMessages)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fail("Error: %s", err)
	}

	fmt.Println("")

	if !m.Finished() {
		fail("Aborted.")
	}

	msg, withBody := m.CommitMessage()
	if err := commit(msg, withBody, signOff); err != nil {
		fail("Error creating commit: %s", err)
	}
}

func fail(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func formUniqueMessages(messages []string, findAllCommitMessages bool) []string {
	uniqueMap := make(map[string]bool)
	var msg string
	for _, m := range messages {
		if !findAllCommitMessages {
			// Given conventional commit adherence, the semicolon can be assumed
			// to be a safe enough delimiter upon which to separate prefix, an
			// optional scope, and the message
			s := strings.Split(m, ":")
			// If m does not contain colon then it's not a valid conventional commit
			if len(s) == 1 {
				continue
			}
			msg = strings.TrimSpace(s[1])
		}
		msg = m
		if _, ok := uniqueMap[msg]; ok {
			continue
		}
		uniqueMap[msg] = true
	}
	return maps.Keys(uniqueMap)
}

func formUniquePaths(paths []string, scopeCompletionOrder string) []string {
	uniqueMap := make(map[string]bool)
	var joinedPaths []string
	for _, p := range paths {
		if _, ok := uniqueMap[p]; ok {
			continue
		}
		s := strings.Split(p, "/")
		for j, q := range s {
			// Prevent overflow
			if j+1 > len(s) {
				continue
			}
			// Make sure leafs are added if they don't exist
			if j+1 == len(s) {
				if _, ok := uniqueMap[q]; !ok {
					uniqueMap[q] = true
				}
			}
			joinedPaths = append(joinedPaths, q)
			joined := strings.Join(joinedPaths, "/")
			if _, ok := uniqueMap[joined]; ok {
				continue
			}
			uniqueMap[joined] = true
		}
		joinedPaths = []string{}
	}

	uniquePaths := maps.Keys(uniqueMap)
	sort.Slice(uniquePaths, func(i, j int) bool {
		if scopeCompletionOrder == "ascending" {
			return len(uniquePaths[i]) < len(uniquePaths[j])
		}
		return len(uniquePaths[i]) > len(uniquePaths[j])
	})
	return uniquePaths
}
