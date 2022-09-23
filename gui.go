package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultWidth = 20
	listHeight   = 16
)

var (
	nordAuroraGreen      = lipgloss.Color("#a3be8c")
	nordAuroraYellow     = lipgloss.Color("#ebcb8b")
	nordAuroraOrange     = lipgloss.Color("#d08770")
	darkGrey             = lipgloss.Color("240")
	filterPromptStyle    = lipgloss.NewStyle().Foreground(nordAuroraYellow)
	filterCursorStyle    = lipgloss.NewStyle().Foreground(nordAuroraOrange)
	titleStyle           = lipgloss.NewStyle().MarginLeft(2)
	itemStyle            = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle    = lipgloss.NewStyle().PaddingLeft(2).Foreground(nordAuroraGreen)
	itemDescriptionStyle = lipgloss.NewStyle().PaddingLeft(2).Faint(true)
	paginationStyle      = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle            = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle        = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	scopeInputText       = "What is the scope?"
	msgInputText         = "What is the commit message?"
	bodyInputText        = "Do you need to specify a body/footer?"
)

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(prefix)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.Title())

	var output string
	if index == m.Index() {
		output = selectedItemStyle.Render("» " + str)
	} else {
		output = itemStyle.Render(str)
	}
	output += itemDescriptionStyle.PaddingLeft(12 - len(str)).Render(i.Description())

	_, _ = fmt.Fprint(w, output)
}

type model struct {
	chosenPrefix       bool
	chosenScope        bool
	chosenMsg          bool
	chosenBody         bool
	specifyBody        bool
	prefix             string
	prefixDescription  string
	scope              string
	msg                string
	prefixList         list.Model
	msgInput           textinput.Model
	scopeInput         textinput.Model
	ynInput            textinput.Model
	previousInputTexts string
	quitting           bool
}

func newModel(prefixes []list.Item, config *config) *model {

	// set up list
	prefixList := list.New(prefixes, itemDelegate{}, defaultWidth, listHeight)
	prefixList.Title = "What are you committing?"
	prefixList.SetShowStatusBar(false)
	prefixList.SetFilteringEnabled(true)
	prefixList.Styles.Title = titleStyle
	prefixList.Styles.PaginationStyle = paginationStyle
	prefixList.Styles.HelpStyle = helpStyle
	prefixList.FilterInput.PromptStyle = filterPromptStyle
	prefixList.FilterInput.CursorStyle = filterCursorStyle

	// set up scope prompt
	scopeInput := textinput.New()
	scopeInput.Placeholder = "Scope"

	// when no limit was defined a default of 0 is used
	if config.ScopeInputCharLimit == 0 {
		scopeInput.CharLimit = 16
		scopeInput.Width = 20
	} else {
		scopeInput.CharLimit = config.ScopeInputCharLimit
		scopeInput.Width = config.ScopeInputCharLimit
	}

	// set up commit message prompt
	commitInput := textinput.New()
	commitInput.Placeholder = "Commit message"

	// when no limit was defined a default of 0 is used
	if config.CommitInputCharLimit == 0 {
		commitInput.CharLimit = 100
		commitInput.Width = 50
	} else {
		commitInput.CharLimit = config.CommitInputCharLimit
		commitInput.Width = config.CommitInputCharLimit
	}

	// set up add body confirmation
	bodyConfirmation := textinput.New()
	bodyConfirmation.Placeholder = "y/N"
	bodyConfirmation.CharLimit = 1
	bodyConfirmation.Width = 20

	return &model{
		prefixList: prefixList,
		scopeInput: scopeInput,
		msgInput:   commitInput,
		ynInput:    bodyConfirmation,
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch {
	case !m.chosenPrefix:
		return m.updatePrefixList(msg)
	case !m.chosenScope:
		return m.updateScopeInput(msg)
	case !m.chosenMsg:
		return m.updateMsgInput(msg)
	case !m.chosenBody:
		return m.updateYNInput(msg)
	default:
		return m, tea.Quit
	}
}

func (m *model) Finished() bool {
	return m.chosenBody
}

func (m *model) CommitMessage() (string, bool) {
	prefix := m.prefix
	if m.scope != "" {
		prefix = fmt.Sprintf("%s(%s)", prefix, m.scope)
	}
	return fmt.Sprintf("%s: %s", prefix, m.msg), m.specifyBody
}

func (m *model) continueWithSelectedItem() {
	i, ok := m.prefixList.SelectedItem().(prefix)
	if ok {
		m.prefix = i.Title()
		m.prefixDescription = i.Description()
		m.chosenPrefix = true
		m.previousInputTexts = fmt.Sprintf(
			"\n%s %s\n",
			m.prefixList.Title,
			lipgloss.NewStyle().Foreground(nordAuroraGreen).Render(fmt.Sprintf("%s: %s", m.prefix, m.prefixDescription)),
		)
		m.scopeInput.Focus()
	}
}

func (m *model) updatePrefixList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.prefixList.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
			var index int
			if keypress == "0" && len(m.prefixList.Items()) == 10 {
				// zero-based indexing, so index 9 equals element 10
				index = 9
			} else if keypress == "0" && len(m.prefixList.Items()) < 10 {
				// keep selected item where it was at
				return m, nil
			} else {
				index, _ = strconv.Atoi(keypress)
				index = index - 1
			}
			m.prefixList.Select(index)
			m.continueWithSelectedItem()

		case "enter":
			m.continueWithSelectedItem()
		}
	}

	var cmd tea.Cmd
	m.prefixList, cmd = m.prefixList.Update(msg)
	return m, cmd
}

func (m *model) updateScopeInput(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.chosenScope = true
			m.scope = m.scopeInput.Value()
			m.previousInputTexts = fmt.Sprintf(
				"%s%s %s\n",
				m.previousInputTexts,
				scopeInputText,
				lipgloss.NewStyle().Foreground(nordAuroraGreen).Render(m.scope),
			)
			m.msgInput.Focus()
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.scopeInput, cmd = m.scopeInput.Update(msg)
	return m, cmd
}

func (m *model) updateMsgInput(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.chosenMsg = true
			m.msg = m.msgInput.Value()
			m.previousInputTexts = fmt.Sprintf(
				"%s%s %s\n",
				m.previousInputTexts,
				msgInputText,
				lipgloss.NewStyle().Foreground(nordAuroraGreen).Render(m.msg),
			)
			m.ynInput.Focus()
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.msgInput, cmd = m.msgInput.Update(msg)
	return m, cmd
}

func (m *model) updateYNInput(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.chosenMsg = true
			switch strings.ToLower(m.ynInput.Value()) {
			case "y":
				m.specifyBody = true
			}
			m.chosenBody = true
			m.previousInputTexts = fmt.Sprintf(
				"%s%s %s\n",
				m.previousInputTexts,
				bodyInputText,
				lipgloss.NewStyle().Foreground(nordAuroraGreen).Render(strconv.FormatBool(m.specifyBody)),
			)
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.ynInput, cmd = m.ynInput.Update(msg)
	return m, cmd
}

func renderCurrentLimit(charLimit int, input string) string {
	padWidth := len(strconv.Itoa(charLimit))
	count := fmt.Sprintf(fmt.Sprintf("%%0%dd", padWidth), len(input))
	return lipgloss.NewStyle().Foreground(darkGrey).Render(fmt.Sprintf(
		"[%s/%d]",
		count,
		charLimit,
	))
}

func (m *model) View() string {
	switch {
	case !m.chosenPrefix:
		return "\n" + m.prefixList.View()
	case !m.chosenScope:
		limit := renderCurrentLimit(m.scopeInput.CharLimit, m.scopeInput.Value())

		return titleStyle.Render(fmt.Sprintf(
			"%s%s (Enter to skip / Esc to cancel) %s:\n%s",
			m.previousInputTexts,
			scopeInputText,
			limit,
			m.scopeInput.View(),
		))
	case !m.chosenMsg:
		limit := renderCurrentLimit(m.msgInput.CharLimit, m.msgInput.Value())

		return titleStyle.Render(fmt.Sprintf(
			"%s%s (Esc to cancel) %s:\n%s",
			m.previousInputTexts,
			msgInputText,
			limit,
			m.msgInput.View(),
		))
	case !m.chosenBody:
		return titleStyle.Render(fmt.Sprintf(
			"%s%s (Esc to cancel):\n%s",
			m.previousInputTexts,
			bodyInputText,
			m.ynInput.View(),
		))
	case m.quitting:
		return quitTextStyle.Render("Aborted.\n")
	default:
		return titleStyle.Render(fmt.Sprintf(
			"%s\n---\n",
			m.previousInputTexts,
		))
	}
}
