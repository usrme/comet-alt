package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if err := checkGitInPath(); err != nil {
		fail(err.Error())
	}

	gitRoot, err := findGitDir()
	if err != nil {
		fail(err.Error())
	}

	if err := os.Chdir(gitRoot); err != nil {
		fail("error changing directory: %s", err)
	}

	stagedFiles, err := filesInStaging()
	if err != nil {
		fail(err.Error())
	}

	prefixes, signOff, config, err := loadConfig()
	if err != nil {
		fail(err.Error())
	}

	commitSearchTerm := ""
	if len(os.Args) > 1 && os.Args[1] == "-m" {
		commitSearchTerm = os.Args[2]
	}

	m := newModel(prefixes, config, stagedFiles, config.ScopeCompletionOrder, commitSearchTerm, config.FindAllCommitMessages)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fail(err.Error())
	}

	fmt.Println("")

	if !m.Finished() {
		fail("terminated")
	}

	msg, withBody := m.CommitMessage()
	if err := commit(msg, withBody, signOff); err != nil {
		fail("error committing: %s", err)
	}
}

func fail(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
