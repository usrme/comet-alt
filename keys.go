package main

import "github.com/charmbracelet/bubbles/key"

type customKeyMap struct {
	Cycle key.Binding
}

var customKeys = customKeyMap{
	Cycle: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "cycle through commit messages or changed file paths"),
	),
}
