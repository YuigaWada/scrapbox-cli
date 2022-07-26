package main

import (
	"YuigaWada/sbox/api"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
	"github.com/toqueteos/webbrowser"
)

const (
	padding  = 2
	maxWidth = 80
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
var linkRegex = regexp.MustCompile(`https?://[\w!?/+\-_~;.,*&@#$%()'[\]]+`)

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

type viewerModel struct {
	parent            interface{}
	page              api.Page
	ready             bool
	linkLoaded        bool
	viewport          viewport.Model
	linkSpinner       spinner.Model
	sublist           subListModel
	paginator         paginator.Model
	visibleItemLength int
}

type nHopLinks struct {
	links []api.Link
}

type subListModel struct {
	index int
}

func (m subListModel) getCursor(perPage int) int {
	return m.index % perPage
}

func (m viewerModel) Init() tea.Cmd {
	return nil
}

func (m viewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		linkSpinnerHeight := lipgloss.Height(m.linkSpinner.View())
		verticalMarginHeight := headerHeight + footerHeight + linkSpinnerHeight
		height := msg.Height - int(1.2*float64(verticalMarginHeight))
		heightAlpha := 3 / 4.0
		m.viewport.HighPerformanceRendering = false
		Width = msg.Width
		Height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, int(float64(height)*heightAlpha))

			page, err := m.page.Read(mainColor)
			if err != nil {
				log.Fatal(err)
			}

			m.viewport.SetContent(wrap.String(wordwrap.String(page.Content, msg.Width), msg.Width))
			m.paginator.SetTotalPages(len(page.Links))
			m.page = *page
			m.ready = true
			m.paginator.PerPage = int(float64(height) * (1 - heightAlpha))

			cmd := func() tea.Msg { // load nHopLinks on goroutine
				links, err := m.page.GetNhopLinks()
				if err != nil {
					return err
				}
				return nHopLinks{links}
			}
			cmds = append(cmds, m.linkSpinner.Tick)
			cmds = append(cmds, cmd)
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = int(float64(msg.Height)*heightAlpha) - verticalMarginHeight
		}

		m.viewport.YPosition = headerHeight + 1
		// m.sublist.viewport.YPosition = height/2 + headerHeight
	case nHopLinks:
		m.page.Links = append(m.page.Links, msg.links...)
		m.paginator.SetTotalPages(len(m.page.Links))
		m.linkLoaded = true
	case tea.KeyMsg:
		// sublist
		switch msg.String() {
		case "esc", "c", "q":
			return m.parent.(tea.Model), nil
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
			if isUrl, url := hasUrl(link.Title); isUrl {
				if err := webbrowser.Open(url); err != nil {
					log.Fatal(err)
				}
				break
			}
			model := MakeViewer(api.MakePage(m.page.Sbox, link.Title))
			model.parent = &m
			return model, MakeInitMsg
		}
	}

	m.linkSpinner, cmd = m.linkSpinner.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m viewerModel) View() string {
	if !m.ready {
		return "Fetching...\n"
	}

	var style = lipgloss.NewStyle().Foreground(mainColor)
	baseView := fmt.Sprintf("%s\n%s\n%s\n[Link]\n", m.headerView(), m.viewport.View(), m.footerView())
	start, end := m.paginator.GetSliceBounds(len(m.page.Links))
	m.visibleItemLength = end - start + 1
	perPage := m.paginator.PerPage
	pageCount := m.paginator.Page
	p := pageCount * perPage
	for i, link := range m.page.Links[p:min(p+perPage, len(m.page.Links))] {
		cursor := " "
		prefix := ""
		if m.sublist.getCursor(perPage) == i {
			cursor = ">"
		}

		if len(link.Tag) > 0 {
			prefix = fmt.Sprintf("[%s] ==> ", link.Tag)
		}

		subListView := fmt.Sprintf("%s %s%s", cursor, prefix, link.Title)
		if m.sublist.getCursor(perPage) == i {
			subListView = style.Render(subListView)
		}
		baseView += subListView + "\n"
	}

	if !m.linkLoaded {
		baseView += m.linkSpinner.View()
	}

	return baseView
}

func (m viewerModel) headerView() string {
	title := titleStyle.Render(m.page.Title())
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m viewerModel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func hasUrl(text string) (bool, string) {
	patterns := linkRegex.FindAllStringSubmatch(text, -1)
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

func MakeViewer(page api.Page) viewerModel {
	s := spinner.New()
	s.Spinner = spinner.Line
	model := viewerModel{page: page,
		paginator:   paginator.NewModel(),
		linkLoaded:  false,
		ready:       false,
		linkSpinner: s}
	return model
}
