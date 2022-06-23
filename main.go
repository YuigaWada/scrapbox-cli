package main

import (
	api "YuigaWada/sbox/wrapper"
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

func loadPages() tea.Msg {
	config := LoadConfig()
	pager := api.MakePager()
	user := api.ScrapUser{config.project}
	rawPages := pager.Read(user)
	return pagesLoadedMsg{rawPages}
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
