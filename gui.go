package main

import (
	"fmt"
	"io"
	"os/exec"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/exp/maps"
)

const (
	defaultWidth = 40
	listHeight   = 15
)

var (
	// #81a1c1: nord9
	// #88c0d0: nord8
	filterPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#81a1c1", Dark: "#88c0d0"})
	// #5e81ac: nord10
	// #8fbcbb: nord7
	filterCursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#5e81ac", Dark: "#8fbcbb"})
	titleStyle           = lipgloss.NewStyle().MarginLeft(2)
	itemStyle            = lipgloss.NewStyle().PaddingLeft(4)
	characterCountColors = lipgloss.AdaptiveColor{Light: "#8dacb6", Dark: "240"}
	// #d08770: nord12
	// #a3be8c: nord13
	selectedItemColors   = lipgloss.AdaptiveColor{Light: "#d08770", Dark: "#a3be8c"}
	selectedItemStyle    = lipgloss.NewStyle().Foreground(selectedItemColors)
	selectedItemPadded   = selectedItemStyle.Copy().PaddingLeft(2)
	itemDescriptionStyle = lipgloss.NewStyle().PaddingLeft(2).Faint(true)
	paginationStyle      = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle            = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle        = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	versionStyle         = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9b9b9b", Dark: "#5c5c5c"}).Render
	scopeInputText       = "What is the scope?"
	msgInputText         = "What is the commit message?"
	bodyInputText        = "Do you need to specify a body/footer?"
	constrainInput       bool
	totalInputCharLimit  int
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
		output = selectedItemPadded.Render("» " + str)
	} else {
		output = itemStyle.Render(str)
	}
	output += itemDescriptionStyle.PaddingLeft(15 - len(str)).Render(i.Description())

	_, _ = fmt.Fprint(w, output)
}

type (
	stagedFilesMsg    []string
	commitMessagesMsg []string
)

type model struct {
	chosenPrefix           bool
	chosenScope            bool
	chosenMsg              bool
	chosenBody             bool
	specifyBody            bool
	prefix                 string
	prefixDescription      string
	scope                  string
	msg                    string
	prefixList             list.Model
	msgInput               textinput.Model
	scopeInput             textinput.Model
	ynInput                textinput.Model
	constrainInput         bool
	totalInputCharLimit    int
	previousInputTexts     string
	typed                  int
	quitting               bool
	stagedFiles            []string
	scopeCompletionOrder   string
	stagedFilePathSegments []string
	scopeInputIndex        int
	commitSearchTerm       string
	findAllCommitMessages  bool
	commitMessages         []string
	messageInputIndex      int
}

func newModel(prefixes []list.Item, config *config, stagedFiles []string, scopeCompletionOrder, commitSearchTerm string, findAllCommitMessages bool) *model {
	prefixList := list.New(prefixes, itemDelegate{}, defaultWidth, listHeight)
	prefixList.Title = "What are you committing?"
	prefixList.SetShowStatusBar(false)
	prefixList.SetFilteringEnabled(true)
	prefixList.Styles.Title = titleStyle
	prefixList.Styles.PaginationStyle = paginationStyle
	prefixList.Styles.HelpStyle = helpStyle
	prefixList.FilterInput.PromptStyle = filterPromptStyle
	prefixList.FilterInput.CursorStyle = filterCursorStyle

	scopeInput := textinput.New()
	scopeInput.Placeholder = "Scope"

	// when no limit was defined a default of 0 is used
	if config == nil || config.ScopeInputCharLimit == 0 {
		scopeInput.CharLimit = 16
		scopeInput.Width = 20
	} else {
		scopeInput.CharLimit = config.ScopeInputCharLimit
		scopeInput.Width = config.ScopeInputCharLimit
	}

	commitInput := textinput.New()
	commitInput.Placeholder = "Commit message"

	// when no limit was defined a default of 0 is used
	if config == nil || config.CommitInputCharLimit == 0 {
		commitInput.CharLimit = 100
		commitInput.Width = 50
	} else {
		commitInput.CharLimit = config.CommitInputCharLimit
		commitInput.Width = config.CommitInputCharLimit
	}

	bodyConfirmation := textinput.New()
	bodyConfirmation.Placeholder = "y/N"
	bodyConfirmation.CharLimit = 1
	bodyConfirmation.Width = 20

	if config == nil || config.TotalInputCharLimit == 0 {
		constrainInput = false
	} else {
		constrainInput = true
		totalInputCharLimit = config.TotalInputCharLimit
	}

	bindings := []key.Binding{
		customKeys.Cycle,
	}
	prefixList.AdditionalShortHelpKeys = func() []key.Binding { return bindings }
	prefixList.AdditionalFullHelpKeys = func() []key.Binding { return bindings }

	return &model{
		prefixList:            prefixList,
		scopeInput:            scopeInput,
		msgInput:              commitInput,
		ynInput:               bodyConfirmation,
		constrainInput:        constrainInput,
		totalInputCharLimit:   totalInputCharLimit,
		stagedFiles:           stagedFiles,
		scopeCompletionOrder:  scopeCompletionOrder,
		commitSearchTerm:      commitSearchTerm,
		findAllCommitMessages: findAllCommitMessages,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		formUniquePaths(m.stagedFiles, m.scopeCompletionOrder),
		findCommitMessages(m.commitSearchTerm, m.findAllCommitMessages),
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
	case stagedFilesMsg:
		m.stagedFilePathSegments = msg
		return m, nil
	case commitMessagesMsg:
		m.commitMessages = msg
		return m, nil
	}
	return m, nil
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
			selectedItemStyle.Render(fmt.Sprintf("%s: %s", m.prefix, m.prefixDescription)),
		)
		m.typed = len(m.prefix) + len("(): ")
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
			m.typed += len(m.scope)
			m.previousInputTexts = fmt.Sprintf(
				"%s%s %s\n",
				m.previousInputTexts,
				scopeInputText,
				selectedItemStyle.Render(m.scope),
			)
			m.msgInput.Focus()
		case tea.KeyTab:
			m.scopeInput.SetValue(m.stagedFilePathSegments[m.scopeInputIndex])
			if m.scopeInputIndex+1 == len(m.stagedFilePathSegments) {
				m.scopeInputIndex = 0
				return m, nil
			}
			m.scopeInputIndex += 1
			m.scopeInput.CursorEnd()
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
			m.typed += len(m.msg)
			m.previousInputTexts = fmt.Sprintf(
				"%s%s %s\n",
				m.previousInputTexts,
				msgInputText,
				selectedItemStyle.Render(m.msg),
			)
			m.ynInput.Focus()
		case tea.KeyTab:
			if len(m.commitMessages) > 0 {
				m.msgInput.SetValue(m.commitMessages[m.messageInputIndex])
				if m.messageInputIndex+1 == len(m.commitMessages) {
					m.messageInputIndex = 0
					return m, nil
				}
				m.messageInputIndex += 1
				m.msgInput.CursorEnd()
			}
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
				selectedItemStyle.Render(strconv.FormatBool(m.specifyBody)),
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

func renderCurrentLimit(m *model, charLimit int, input string) string {
	var limit, inputLength int
	if m.constrainInput {
		limit = m.totalInputCharLimit
		inputLength = len(m.prefix) + len("(): ") + len(input) + len(m.scope)
	} else {
		limit = charLimit
		inputLength = len(input)
	}

	padWidth := len(strconv.Itoa(limit))
	count := fmt.Sprintf(fmt.Sprintf("%%0%dd", padWidth), inputLength)

	return lipgloss.NewStyle().Foreground(characterCountColors).Render(fmt.Sprintf(
		"[%s/%d]",
		count,
		limit,
	))
}

func (m *model) View() string {
	lengthExceedMessage := "Number of characters equals total input limit. Value will be left blank"

	m.prefixList.NewStatusMessage(versionStyle(pkgVersion()))

	switch {
	case !m.chosenPrefix:
		return "\n" + m.prefixList.View()
	case !m.chosenScope:
		limit := renderCurrentLimit(m, m.scopeInput.CharLimit, m.scopeInput.Value())

		if m.constrainInput {
			m.scopeInput.CharLimit = m.totalInputCharLimit - m.typed
			if m.scopeInput.CharLimit == 0 {
				m.scopeInput.Placeholder = lengthExceedMessage
				m.scopeInput.EchoMode = textinput.EchoNone
				m.scopeInput.SetValue("")
			}
		}

		return titleStyle.Render(fmt.Sprintf(
			"%s%s (Enter to skip / Esc to cancel) %s:\n%s",
			m.previousInputTexts,
			scopeInputText,
			limit,
			m.scopeInput.View(),
		))
	case !m.chosenMsg:
		limit := renderCurrentLimit(m, m.msgInput.CharLimit, m.msgInput.Value())

		if m.constrainInput {
			m.msgInput.CharLimit = m.totalInputCharLimit - m.typed
			if m.msgInput.CharLimit == 0 {
				m.msgInput.Placeholder = lengthExceedMessage
				m.msgInput.EchoMode = textinput.EchoNone
				m.msgInput.SetValue("")
			}
		}

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

func formUniquePaths(stagedFiles []string, scopeCompletionOrder string) tea.Cmd {
	return func() tea.Msg {
		uniqueMap := make(map[string]bool)
		var joinedPaths []string
		for _, p := range stagedFiles {
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
		return stagedFilesMsg(uniquePaths)
	}
}

func findCommitMessages(grep string, findAll bool) tea.Cmd {
	return func() tea.Msg {
		if grep == "" {
			return commitMessagesMsg([]string{})
		}
		cmd := exec.Command("git", "log", "--oneline", "--pretty=format:%s", "--grep="+grep)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return commitMessagesMsg([]string{})
		}

		messages := strings.Split(strings.TrimSpace(string(output)), "\n")
		uniqueMap := make(map[string]bool)
		var msg string
		for _, m := range messages {
			msg = m
			if !findAll {
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
			if _, ok := uniqueMap[msg]; ok {
				continue
			}
			uniqueMap[msg] = true
		}
		return commitMessagesMsg(maps.Keys(uniqueMap))
	}
}

func pkgVersion() string {
	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}
	return version
}

func showTable(data [][]string) {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Period", "Time"})
	table.SetHeaderColor(tablewriter.Colors{tablewriter.FgGreenColor}, tablewriter.Colors{tablewriter.FgGreenColor})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("-")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data)
	table.Render()
	lines := strings.Split(tableString.String(), "\n")
	for i := 0; i < len(lines); i++ {
		fmt.Printf("  %s\n", lines[i])
	}
}
