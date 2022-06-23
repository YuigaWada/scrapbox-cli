package main

import (
	api "YuigaWada/sbox/wrapper"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/toqueteos/webbrowser"
)

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

type pagerModel struct {
	rawPage  api.Page
	page     api.ScrapboxPage
	ready    bool
	viewport viewport.Model
	progress progress.Model
	sublist  subListModel
}

type subListModel struct {
	cursor int
}

func (m pagerModel) Init() tea.Cmd {
	return nil
}

func (m pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}

		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight
		height := msg.Height - verticalMarginHeight
		heightAlpha := 3 / 4.0
		m.viewport.HighPerformanceRendering = false

		if !m.ready {
			m.viewport = viewport.New(msg.Width, int(float64(height)*heightAlpha))

			page, err := m.rawPage.Read(mainColor)
			if err != nil {
				log.Fatal(err)
			}

			m.viewport.SetContent(page.Content)
			m.page = page
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = int(float64(msg.Height)*heightAlpha) - verticalMarginHeight
		}

		m.viewport.YPosition = headerHeight + 1
		// m.sublist.viewport.YPosition = height/2 + headerHeight
	case tea.KeyMsg:
		// sublist
		switch msg.String() {
		case "esc", "ctrl+c", "c", "q":
			return m, tea.Quit
		case "left", "k":
			if m.sublist.cursor > 0 {
				m.sublist.cursor--
			}
		case "right", "j":
			if m.sublist.cursor < len(m.page.Links)-1 {
				m.sublist.cursor++
			}
		case "enter", " ":
			link := m.page.Links[m.sublist.cursor]
			if isUrl, url := hasUrl(link); isUrl {
				if err := webbrowser.Open(url); err != nil {
					log.Fatal(err)
				}
				break
			}
			RunPager(api.MakePage(m.rawPage.User, link))
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m pagerModel) View() string {
	if !m.ready {
		return "Fetching...\n" + m.progress.View()
	}

	var style = lipgloss.NewStyle().Foreground(mainColor)
	baseView := fmt.Sprintf("%s\n%s\n%s\n", m.headerView(), m.viewport.View(), m.footerView())
	for i, link := range m.page.Links {
		cursor := " "
		if m.sublist.cursor == i {
			cursor = ">"
		}
		subListView := fmt.Sprintf("%s %s\n", cursor, link)
		if m.sublist.cursor == i {
			subListView = style.Render(subListView)
		}
		baseView += subListView
	}

	return baseView
}

func (m pagerModel) headerView() string {
	title := titleStyle.Render(m.rawPage.Title_)
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m pagerModel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func hasUrl(text string) (bool, string) {
	rex := regexp.MustCompile(`https?://[\w!?/+\-_~;.,*&@#$%()'[\]]+`)
	patterns := rex.FindAllStringSubmatch(text, -1)
	if len(patterns) > 0 {
		return true, patterns[0][0]
	}
	return false, ""
}

func RunPager(rawPage api.Page) {
	model := pagerModel{rawPage: rawPage}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if err := p.Start(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}
