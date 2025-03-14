package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type QuitError struct{}
type RerunError struct{}

func (q QuitError) Error() string {
	return "quitting"
}

func (q RerunError) Error() string {
	return "rerunning"
}

func HandleQuitError(err error) {
	if errors.Is(err, QuitError{}) {
		os.Exit(0)
	}
}

type CmdSelector struct {
	cmds     []string
	cursor   int
	selected string
	quit     bool
	rerun    bool
}

func NewCmdSelector(cmds []string) *CmdSelector {
	return &CmdSelector{
		cmds:     cmds,
		cursor:   0,
		selected: "",
		quit:     false,
		rerun:    false,
	}
}

func (m *CmdSelector) Init() tea.Cmd {
	return nil
}

func (m *CmdSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.cmds) - 1
			}
		case "down", "j":
			if m.cursor < len(m.cmds)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "r":
			m.rerun = true
			return m, tea.Quit
		case "enter", " ":
			m.selected = m.cmds[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

// Colors
var (
	MutedGray       = lipgloss.Color("#A1A1AA")
	MutedPurpleBlue = lipgloss.Color("#5A3FC0")
	NeuralGrey      = lipgloss.Color("#BDBDBD")
	SlateBlue       = lipgloss.Color("#64748B")
	SoftGreen       = lipgloss.Color("#6FCF97")
	WarmOrange      = lipgloss.Color("#F4A261")
	White           = lipgloss.Color("#FFFFFF")
)

// Styles
var (
	TitleStyle        = lipgloss.NewStyle()
	ItemStyle         = lipgloss.NewStyle().Padding(0, 1)
	SelectedItemStyle = lipgloss.NewStyle().Foreground(White).Background(SlateBlue).Padding(0, 1)
	CheckedStyle      = lipgloss.NewStyle().Foreground(SoftGreen)
	UncheckedStyle    = lipgloss.NewStyle().Foreground(NeuralGrey)
	HelpStyle         = lipgloss.NewStyle().Foreground(MutedGray)
	KeyStyle          = lipgloss.NewStyle().Foreground(WarmOrange).Bold(true)
	TableHeaderStyle  = lipgloss.NewStyle().Foreground(SoftGreen).Bold(true)
)

// keybindings
var (
	NavigateKey1 = KeyStyle.Render("↑/↓")
	NavigateKey2 = KeyStyle.Render("k/j")
	ProceedKey   = KeyStyle.Render("Enter")
	RerunKey     = KeyStyle.Render("r")
	ExitKey1     = KeyStyle.Render("Ctrl+c")
	ExitKey2     = KeyStyle.Render("q")
)

// words
var (
	Use        = HelpStyle.Render("Use")
	Press      = HelpStyle.Render("Press")
	Or         = HelpStyle.Render("or")
	ToNavigate = HelpStyle.Render("to navigate")
	ToProceed  = HelpStyle.Render("to proceed")
	ToExit     = HelpStyle.Render("to exit")
	ToRerun    = HelpStyle.Render("to rerun")
)

// help messages
var (
	Navigate = fmt.Sprintf("  %s %s %s %s %s\n", Use, NavigateKey1, Or, NavigateKey2, ToNavigate)
	Proceed  = fmt.Sprintf("  %s %s %s\n", Press, ProceedKey, ToProceed)
	Rerun    = fmt.Sprintf("  %s %s %s\n", Press, RerunKey, ToRerun)
	Exit     = fmt.Sprintf("  %s %s %s %s %s\n", Press, ExitKey1, Or, ExitKey2, ToExit)
)

func (m *CmdSelector) View() string {
	s := "\nChoose a command:\n"
	for i, choice := range m.cmds {
		cursor := " "
		style := ItemStyle

		if i == m.cursor {
			cursor = ">"
			style = SelectedItemStyle
		}

		s += fmt.Sprintf("%s %s\n", cursor, style.Render(choice))
	}

	return s + "\n\n" + Navigate + Rerun + Proceed + Exit
}

func SelectCmd(cmds []CmdEntry) (string, error) {
	maxCmdLength := 0
	for _, entry := range cmds {
		if len(entry.Cmd) > maxCmdLength {
			maxCmdLength = len(entry.Cmd)
		}
	}

	commentedCmds := make([]string, len(cmds))
	for i, entry := range cmds {
		if entry.Comment != "" {
			padding := strings.Repeat(" ", maxCmdLength-len(entry.Cmd)+2)
			commentedCmds[i] = fmt.Sprintf("%s%s# %s", entry.Cmd, padding, entry.Comment)
		} else {
			commentedCmds[i] = entry.Cmd
		}
	}

	model := NewCmdSelector(commentedCmds)
	p := tea.NewProgram(model)

	_, err := p.Run()
	if err != nil {
		return "", err
	}

	if model.quit {
		return "", QuitError{}
	}

	if model.rerun {
		return "", RerunError{}
	}

	return cmds[model.cursor].Cmd, nil
}
