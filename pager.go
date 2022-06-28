package main

import (
	api "YuigaWada/sbox/wrapper"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
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
	rawPage           api.Page
	page              api.ScrapboxPage
	ready             bool
	viewport          viewport.Model
	progress          progress.Model
	sublist           subListModel
	paginator         paginator.Model
	visibleItemLength int
	viewDepth         *int
	currentDepth      int
}

type subListModel struct {
	index int
}

func (m subListModel) getCursor(perPage int) int {
	return m.index % perPage
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
			m.paginator.SetTotalPages(len(page.Links))
			m.page = page
			m.ready = true
			m.paginator.PerPage = int(float64(height) * (1 - heightAlpha))
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = int(float64(msg.Height)*heightAlpha) - verticalMarginHeight
		}

		m.viewport.YPosition = headerHeight + 1
		// m.sublist.viewport.YPosition = height/2 + headerHeight
	case tea.KeyMsg:
		if *m.viewDepth != m.currentDepth {
			break
		}

		// sublist
		switch msg.String() {
		case "esc", "c", "q":
			*m.viewDepth--
			return m, tea.Quit
		case "left", "k":
			if m.sublist.index > 0 {
				m.sublist.index--
				if (m.sublist.index+1)%m.paginator.PerPage == 0 {
					m.paginator.PrevPage()
				}
			}
		case "right", "j":
			if m.sublist.index+1 < len(m.page.Links) {
				m.sublist.index++
			}

			if m.sublist.index%m.paginator.PerPage == 0 && !m.paginator.OnLastPage() {
				m.paginator.NextPage()
			}
			break
		case "enter", " ":
			link := m.page.Links[m.sublist.index]
			if isUrl, url := hasUrl(link); isUrl {
				if err := webbrowser.Open(url); err != nil {
					log.Fatal(err)
				}
				break
			}
			RunPager(api.MakePage(m.rawPage.User, link), m.viewDepth)
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
	start, end := m.paginator.GetSliceBounds(len(m.page.Links))
	m.visibleItemLength = end - start + 1
	perPage := m.paginator.PerPage
	pageCount := m.paginator.Page
	p := pageCount * perPage
	for i, link := range m.page.Links[p:min(p+perPage, len(m.page.Links))] {
		cursor := " "
		if m.sublist.getCursor(perPage) == i {
			cursor = ">"
		}
		subListView := fmt.Sprintf("%s %s", cursor, link)
		if m.sublist.getCursor(perPage) == i {
			subListView = style.Render(subListView)
		}
		baseView += subListView + "\n"
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

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func RunPager(rawPage api.Page, viewDepth *int) {
	*viewDepth++
	model := pagerModel{rawPage: rawPage,
		paginator:    paginator.NewModel(),
		viewDepth:    viewDepth,
		currentDepth: *viewDepth}

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
