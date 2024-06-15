package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Mode string

const (
	Normal Mode = "normal"
	Insert Mode = "insert"
)

type item struct {
	normal, selected, invalid lipgloss.Style
}

var (
	ItemStyle = item{
		normal:   lipgloss.NewStyle().Foreground(lipgloss.Color("170")),
		selected: lipgloss.NewStyle(),
		invalid:  lipgloss.NewStyle().Strikethrough(true),
	}

	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	defaultItemStyle  = lipgloss.NewStyle()
	invalidItemStyle  = lipgloss.NewStyle().Strikethrough(true)

	quitTextStyle = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

// for testing only
func DefaultStyle() lipgloss.Style {
	return lipgloss.
		NewStyle().
		BorderForeground(lipgloss.Color("36")).
		BorderStyle(lipgloss.NormalBorder()).
		Padding(0).
		Margin(0)
}

type model struct {
	width  int
	height int

	mode Mode

	items          Arr[Tab]
	currentItemIdx int
	size           int

	history []string

	msgs []tea.Msg

	input    textinput.Model
	viewport *viewport.Model

	files *Files
}

func initialModel(t Files) model {
	ti := textinput.New()
	ti.Placeholder = "..."
	ti.Prompt = "> "

	// size := term.GetSize()

	// TODO: adjust width/height to all available space
	// TODO: resize
	vp := viewport.New(100, 20)

	// ti.CharLimit = 280
	return model{
		mode:     Normal,
		input:    ti,
		history:  []string{"1", "2", "3"},
		viewport: &vp,
		files:    &t,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) View() string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Left,
		lipgloss.Left,

		lipgloss.JoinVertical(
			lipgloss.Center,
			lipgloss.JoinHorizontal(
				0,
				DefaultStyle().Render(m.headerView()),
				DefaultStyle().Width(80).Render(m.input.View()),
			),
			DefaultStyle().Width(60).Render(m.viewport.View()),
			// fmt.Sprint(m.currentItemIdx+1),
			// m.footerView(),
			// fmt.Sprint(m.msgs),
		),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.msgs = append([]tea.Msg{msg}, m.msgs[:(min(10, len(m.msgs)))]...)

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

	case tea.KeyMsg:

		switch m.mode {
		case Normal:
			switch msg.String() {
			// case "s": // save
			case "f": // search mode
				m.mode = Insert
				m.input.Focus()
			case "k":
				m.LineUp()
			case "j":
				m.LineDown()
			case " ":
				if m.size == 0 {
					return m, nil
				}
				m.items[m.currentItemIdx].ToggleOpenable()
				m.updateViewport()

			case "o":
				limit := 10
				var items Arr[Tab]
				var consumed []string
				for i := 0; i < min(len(m.items), limit); i++ {
					t := m.items[i]
					if t.CanOpen() {
						t.Open()
						t.Consume()
						consumed = append(consumed, t.URL)
					} else {
						items.Append(t)
					}
				}
				m.items = items
				m.size = len(items)
				m.currentItemIdx = m.size - 1

				m.updateViewport()
				defer m.files.RemoveTabs(consumed)

			case "ctrl+c", "q":
				return m, tea.Quit
			}

		case Insert:
			switch msg.String() {
			case "enter":
				val := m.input.Value()
				if strings.Trim(val, " ") != "" {
					m.history = append(func(h []string) []string {
						var o []string
						for _, s := range h {
							if s != "" {
								o = append(o, s)
							}
						}
						return o
					}(m.history), []string{val, ""}...)

					m.items = m.find(val)
					m.size = len(m.items)
					m.currentItemIdx = m.size - 1

					m.input.Reset()

					// TODO: can't select/copy with mouse

					m.updateViewport()

					m.viewport.GotoBottom()
				}
			case "ctrl+k":
				m.LineUp()
			case "ctrl+j":
				m.LineDown()
			case "ctrl+n":
				m.input.SetValue(m.HistoryDown())
			case "ctrl+p":
				m.input.SetValue(m.HistoryUp())
			case "esc":
				m.mode = Normal
				m.input.Reset()
				m.input.Blur()
			default:
				m.input, cmd = m.input.Update(msg)
			}
		}
	}

	return m, cmd
}

func checkbox(tab Tab) string {
	if tab.CanOpen() {
		return fmt.Sprintf("[x] %s", tab.Title)
	}
	return fmt.Sprintf("[ ] %s", tab.Title)
}

func (m *model) find(query string) Arr[Tab] {
	var found Arr[Tab]
	search := strings.ToLower(query)
	for _, data := range m.files.data {
		for _, g := range data.payload.Groups {
			for _, tab := range g.Tabs {
				if tab.Contains(search) {
					tab.SetOpenable()
					found.Append(tab)
				}
			}
		}
	}
	return found
}

func (m *model) LineDown() {
	if m.currentItemIdx < m.size-1 {
		m.currentItemIdx++
		m.viewport.LineDown(1)
		m.updateViewport()
	}
}

func (m *model) LineUp() {
	if m.currentItemIdx > 0 {
		m.currentItemIdx--
		m.viewport.LineUp(1)
		m.updateViewport()
	}
}

func (m *model) HistoryUp() string {
	size := len(m.history)
	item := m.history[size-1]
	m.history = append([]string{item}, m.history[:size-1]...)
	return item
}

func (m *model) HistoryDown() string {
	item := m.history[0]
	m.history = append(m.history[1:], item)
	return item
}

var s1 = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4")).
	PaddingTop(2).
	PaddingLeft(4).
	Inline(true).
	MaxWidth(22)

func (m *model) updateViewport() {
	var out []string

	log.Printf("wid: %d", m.viewport.Style.GetWidth())

	for i := 0; i < len(m.items); i++ {
		tab := m.items[i]
		item := checkbox(tab)

		s := defaultItemStyle
		if i == m.currentItemIdx {
			s = selectedItemStyle
		}

		out = append(out, s.MaxWidth(30).Render(item))
	}
	// s := list.New(out)
	// m.viewport.SetContent(s)
	m.viewport.SetContent(strings.Join(out, "\n"))
	viewport.Sync(*m.viewport)
}

func (m model) headerView() string {
	return string(m.mode)
	// line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	// title := titleStyle.Render("Mr. Pager")
	// return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	return "footer"
	// info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	// line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	// return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}
