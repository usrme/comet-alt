package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func filesInStaging() ([]string, error) {
	cmd := exec.Command("git", "diff", "--no-ext-diff", "--cached", "--name-only")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return []string{}, fmt.Errorf(string(output))
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 0 {
		return []string{}, fmt.Errorf("no files added to staging area")
	}
	return files, nil
}

func checkGitInPath() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("cannot find git in PATH: %w", err)
	}
	return nil
}

func findGitDir() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf(string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

func commit(msg string, body bool, signOff bool) error {
	gitArgs := os.Args[1:]
	if len(os.Args) > 1 && os.Args[1] == "-m" {
		gitArgs = os.Args[3:]
	}
	args := append([]string{
		"commit", "-m", msg,
	}, gitArgs...)
	if body {
		args = append(args, "-e")
	}
	if signOff {
		args = append(args, "-s")
	}
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
