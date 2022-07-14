package main

import (
	"YuigaWada/sbox/api"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli/v2"
)

const (
	mainColor = lipgloss.Color("#17c0eb")
)

var Width int = 0
var Height int = 0

func loadPages() tea.Msg {
	config := LoadConfig()
	s := api.Scrapbox{Project: config.project}
	pages := s.Read()
	return pagesLoadedMsg{pages}
}

func getCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:    "register",
			Aliases: []string{"r"},
			Usage:   "register your project name",
			Action: func(c *cli.Context) error {
				name := c.Args().First()

				config := Config{project: name}
				config.save("config.json")

				fmt.Println("success: Register your project name successfully!")
				fmt.Printf("Project Name: %s", name)

				return nil
			},
		},
		{
			Name:    "view",
			Aliases: []string{"v"},
			Usage:   "complete a task on the list",
			Action: func(c *cli.Context) error {
				prog := tea.NewProgram(MakeListModel(loadPages))
				err := prog.Start()
				if err != nil {
					log.Fatal(err)
				}
				return nil
			},
		},
	}
}

func main() {
	app := &cli.App{
		Name:     "sbox",
		Usage:    "a simple viewer for scrapbox",
		Commands: getCommands(),
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func MakeInitMsg() tea.Msg {
	return tea.WindowSizeMsg{
		Width:  Width,
		Height: Height,
	}
}
