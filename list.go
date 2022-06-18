package main

import (
	api "YuigaWada/sbox/wrapper"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

type listModel struct {
	list     list.Model
	viewport viewport.Model
	pages    []api.Page
}

type pagesLoadedMsg struct {
	pages []api.Page
}

func (m listModel) loadPages() tea.Msg {
	config := LoadConfig()
	pager := api.MakePager()
	user := api.ScrapUser{config.project}
	rawPages := pager.Read(user)
	return pagesLoadedMsg{rawPages}
}

func (m listModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadPages,
		m.list.StartSpinner(),
	)
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := footerHeight

		m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
		m.viewport.YPosition = 0
		m.viewport.HighPerformanceRendering = false
		m.viewport.SetContent(m.list.Title)
		m.viewport.YPosition = 0
		m.list.SetSize(msg.Width, msg.Height)

	case pagesLoadedMsg:
		items := make([]list.Item, len(msg.pages))
		for i, page := range msg.pages {
			items[i] = page
		}
		m.list.StopSpinner()
		m.list.SetItems(items)
		m.pages = msg.pages

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			RunPager(m.pages[m.list.Index()]) // todo: m.list.SelectedItem()
		}
	}

	return m, cmd
}

func (m listModel) View() string {
	// return fmt.Sprintf("%s \n%s \n%s", m.list.View(), m.viewport.View(), m.footerView())
	return fmt.Sprintf("%s", m.list.View())
}

func (m listModel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MakeListModel() listModel {
	m := listModel{
		list: list.New(nil, list.NewDefaultDelegate(), 1, 1),
	}

	m.list.StartSpinner()
	return m
}
