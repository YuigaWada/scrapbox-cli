package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/YuigaWada/sbox/api"

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
	config, err := LoadConfig()
	for err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		cli.Exit("", 1)
	}

	p := api.Paginate{Skip: 0, Limit: 500}
	s := api.Scrapbox{Project: config.Project, Paginate: p}
	pages := s.Read()
	return pagesLoadedMsg{pages}
}

func registerProject(name string) error {
	if len(name) == 0 {
		fmt.Fprintln(os.Stderr, "project name is required")
		return errors.New("project name is required")
	}

	config := Config{Project: name}
	err := config.save()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return err
	}

	fmt.Println("success: Register your project name successfully!")
	fmt.Printf("Project Name: %s", name)

	return nil
}

func run() error {
	_, cerr := LoadConfig()
	for cerr != nil {
		name := ""
		fmt.Printf("Your project name?\n>")
		fmt.Scanf("%s", &name)
		cerr = registerProject(name)
	}
	prog := tea.NewProgram(MakeListModel(loadPages))
	err := prog.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
	}
	return err
}

func getCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:    "register",
			Aliases: []string{"r"},
			Usage:   "register your project name",
			Action: func(c *cli.Context) error {
				name := c.Args().First()
				err := registerProject(name)
				return err
			},
		},
		{
			Name:    "view",
			Aliases: []string{"v"},
			Usage:   "complete a task on the list",
			Action: func(c *cli.Context) error {
				err := run()
				return err
			},
		},
	}
}

func main() {
	app := &cli.App{
		Name:     "sbox",
		Usage:    "a simple viewer for scrapbox",
		Commands: getCommands(),
		Action: func(*cli.Context) error {
			err := run()
			return err
		},
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
