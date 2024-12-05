package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if err := findGitDir(); err != nil {
		fail(err.Error())
	}

	stagedFiles, err := filesInStaging()
	if err != nil {
		fail(err.Error())
	}

	commitSearchTerm := ""
	if len(os.Args) > 1 && os.Args[1] == "-m" {
		commitSearchTerm = os.Args[2]
	}

	config := loadConfig()

	tracker, err := NewRuntimeTracker("")
	if err != nil {
		fail("error creating tracker: %s", err)
	}

	if config.StoreRuntime || config.ShowRuntime {
		tracker.Start()
	}

	m := newModel(config, stagedFiles, commitSearchTerm)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fail(err.Error())
	}

	fmt.Println("")
	if !m.Finished() {
		fail("terminated")
	}

	msg, withBody := m.CommitMessage()
	if err := commit(msg, withBody, config.SignOffCommits); err != nil {
		fail("error committing: %s", err)
	}

	if config.StoreRuntime || config.ShowRuntime {
		err := tracker.Stop()
		if err != nil {
			fail("error stopping tracker: %s", err)
		}

		if config.ShowRuntime {
			stats := tracker.GetStats()
			fmt.Println()
			showTable([][]string{
				{"Session", fmt.Sprintf("%f", stats.Session)},
			})
		}
	}

	if config.ShowStats {
		stats := tracker.GetStats()
		showTable([][]string{
			{"Session", fmt.Sprintf("%f", stats.Session)},
			{"Daily", fmt.Sprintf("%f", stats.Daily[stats.CurrentDay])},
			{"Weekly", fmt.Sprintf("%f", stats.Weekly[stats.CurrentWeek])},
			{"Monthly", fmt.Sprintf("%f", stats.Monthly[stats.CurrentMonth])},
			{"Yearly", fmt.Sprintf("%f", stats.Yearly[stats.CurrentYear])},
		})
	}
}

func fail(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
