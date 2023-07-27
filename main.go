package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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

	stagedFiles, err := filesInStaging()
	if err != nil {
		fail("Error: %s", err)
	}

	prefixes, signOff, config, err := loadConfig()
	if err != nil {
		fail("Error: %s", err)
	}

	commitSearchTerm := ""
	if len(os.Args) > 1 && os.Args[1] == "-m" {
		commitSearchTerm = os.Args[2]
	}

	m := newModel(prefixes, config, stagedFiles, config.ScopeCompletionOrder, commitSearchTerm, config.FindAllCommitMessages)
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
