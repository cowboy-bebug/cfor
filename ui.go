package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
	DeleteKey1   = KeyStyle.Render("Backspace")
	DeleteKey2   = KeyStyle.Render("d")
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
	ToDelete   = HelpStyle.Render("to delete entry")
	ToRerun    = HelpStyle.Render("to rerun")
)

// help messages
var (
	Navigate = fmt.Sprintf("  %s %s %s %s %s\n", Use, NavigateKey1, Or, NavigateKey2, ToNavigate)
	Proceed  = fmt.Sprintf("  %s %s %s\n", Press, ProceedKey, ToProceed)
	Rerun    = fmt.Sprintf("  %s %s %s\n", Press, RerunKey, ToRerun)
	Delete   = fmt.Sprintf("  %s %s %s %s %s\n", Press, DeleteKey1, Or, DeleteKey2, ToDelete)
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

type Table struct {
	table   table.Model
	quit    bool
	ogTotal float64
}

func (m Table) Init() tea.Cmd {
	return nil
}

func (m Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			return m, tea.Quit
		case "backspace", "d":
			selectedRow := m.table.SelectedRow()
			if selectedRow[0] == "TOTAL" {
				return m, nil
			}

			date := selectedRow[0]
			if err := DeleteCostEntry(Today(date)); err != nil {
				return m, nil
			}

			costs, err := GetCosts()
			if err != nil {
				return m, nil
			}

			newModel := NewTableModel(costs)
			newModel.ogTotal = m.ogTotal
			newModel.table.SetCursor(0)

			rows := newModel.table.Rows()
			for i, row := range rows {
				if row[0] == "TOTAL" {
					rows[i] = table.Row{"TOTAL", fmt.Sprintf("%.5f", m.ogTotal)}
					break
				}
			}
			newModel.table.SetRows(rows)

			m.table = newModel.table
			m.ogTotal = newModel.ogTotal

			return m, nil
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Table) View() string {
	return m.table.View() +
		strings.Repeat("\n", 3) +
		Navigate + Delete + Exit
}

func NewTableModel(costs Costs) Table {
	columns := []table.Column{
		{Title: "Date", Width: 15},
		{Title: "Cost ($)", Width: 15},
	}

	rows := []table.Row{}
	var totalCost float64

	thisRepoIndex := 0
	today := time.Now().Format("2006-01-02")
	for date, cost := range costs {
		rows = append(rows, table.Row{string(date), fmt.Sprintf("%.5f", cost)})
		totalCost += float64(cost)

		if string(date) == today {
			thisRepoIndex = len(rows) - 1
		} else {
			thisRepoIndex++
		}
	}
	rows = append(rows, table.Row{"TOTAL", fmt.Sprintf("%.5f", totalCost)})

	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
	)
	t.SetCursor(thisRepoIndex)

	s := table.DefaultStyles()
	s.Header = TableHeaderStyle
	s.Selected = SelectedItemStyle.Padding(0, 0)
	t.SetStyles(s)

	return Table{table: t, quit: false, ogTotal: totalCost}
}

func CostTableModel(costs Costs) error {
	model := NewTableModel(costs)
	p := tea.NewProgram(model)

	_, err := p.Run()
	if err != nil {
		return err
	}

	if model.quit {
		return QuitError{}
	}

	return nil
}
