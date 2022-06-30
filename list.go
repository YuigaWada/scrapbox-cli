package main

import (
	api "YuigaWada/sbox/wrapper"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	bubleList "github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

type listModel struct {
	list     list.Model
	viewport viewport.Model
	pages    []api.Page
	load     func() tea.Msg
	ready    bool
}

type pagesLoadedMsg struct {
	pages []api.Page
}

func (m listModel) Init() tea.Cmd {
	return tea.Batch(
		m.load,
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
		m.list.SetSize(msg.Width, msg.Height)
		Width = msg.Width
		Height = msg.Height

		if !m.ready {
			m.viewport.SetContent(m.list.Title)
			m.ready = true
		}

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
			for _, item := range m.pages {
				if m.list.SelectedItem().FilterValue() == item.Title_ {
					model := MakeViewer(item)
					model.parent = &m
					return model, MakeInitMsg
				}
			}
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

func makeLipglossStyles() bubleList.DefaultItemStyles {
	styles := bubleList.NewDefaultItemStyles()
	styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(mainColor).
		Foreground(mainColor).
		Padding(0, 0, 0, 1)

	styles.SelectedDesc = styles.SelectedTitle.Copy().
		Foreground(mainColor)
	return styles
}

func makeListDelegate() bubleList.DefaultDelegate {
	delegate := bubleList.NewDefaultDelegate()
	delegate.Styles = makeLipglossStyles()
	return delegate
}

func makeList() list.Model {
	lm := list.New(nil, makeListDelegate(), 1, 1)
	lm.Title = "sbox"
	return lm
}

func MakeListModel(loadFunc func() tea.Msg) listModel {
	m := listModel{
		list:  makeList(),
		load:  loadFunc,
		ready: false,
	}

	m.list.StartSpinner()
	return m
}
