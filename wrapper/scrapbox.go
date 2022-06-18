package wrapper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type Page struct {
	Title_  string `json:"title"`
	ID      string `json:"id"`
	BaseUrl string
	ApiUrl  string
}

func (p Page) Description() string {
	res, err := url.PathUnescape(p.ApiUrl)
	if err != nil {
		panic("failed to unescape url")
	}
	return res
}

func (p Page) Title() string {
	return p.Title_
}

func (p Page) FilterValue() string {
	return p.Title_
}

func (p Page) Read() ScrapboxPage {
	const errMsg = "Failed to open URL .... :("
	if len(p.ApiUrl) == 0 {
		return errMsg
	}

	resp, err := http.Get(p.ApiUrl)
	if err != nil {
		return errMsg
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errMsg
	}

	content := ScrapboxPage(body)
	return content.parse()
}

var _ list.DefaultItem = (*Page)(nil)

type ScrapUser struct {
	Project string
}

func (user *ScrapUser) getReadApi() string {
	return fmt.Sprintf("https://scrapbox.io/api/pages/%s", user.Project)
}

func (user *ScrapUser) getDetailApi(title string) string {
	return fmt.Sprintf("https://scrapbox.io/api/pages/%s/%s/text", user.Project, url.PathEscape(title))
}

type Pager struct {
	skip  int
	limit int
}

func MakePager() Pager {
	return Pager{0, 100}
}

func (p *Pager) Read(user ScrapUser) []Page {
	url := user.getReadApi()
	url = fmt.Sprintf("%s?skip=%d&limit=%d", url, p.skip, p.limit)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	var pages []Page
	result := struct{ Pages []Page }{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	pages = result.Pages
	p.skip += len(pages)

	for i, page := range pages {
		pages[i].ApiUrl = user.getDetailApi(page.Title_)
	}

	return pages
}
func (p Pager) Write(title string, body string) {

}

type ScrapboxPage string

func (spage *ScrapboxPage) parse() ScrapboxPage {
	rex := regexp.MustCompile(`\[([^(\[|\])]+)\]`)
	patterns := rex.FindAllStringSubmatch(string(*spage), -1)

	if len(patterns) == 0 {
		return *spage
	}

	slice := strings.Split(spage.ToString(), "\n")
	dotStr := lipgloss.NewStyle().Bold(true).Render("・")
	for i, str := range slice {
		str := []rune(str)
		if i == 0 {
			continue
		}
		if len(str) == 0 || str[0] != ' ' {
			continue
		}
		for j := 0; j < len(str); j++ {
			if !unicode.IsSpace(str[j]) && j > 0 {
				slice[i] = slice[i][:j] + dotStr + slice[i][j:]
				break
			}
		}
	}

	*spage = ScrapboxPage(strings.Join(slice, "\n"))

	var style = lipgloss.NewStyle().Foreground(lipgloss.Color("201"))
	for _, pattern := range patterns {
		for _, link := range pattern {
			decoLink := fmt.Sprintf("[%s]", string(link))
			s := strings.Replace(string(*spage), decoLink, style.Render(link), -1)
			*spage = ScrapboxPage(s)
		}
	}

	return *spage
}

func (spage ScrapboxPage) ToString() string {
	return string(spage)
}
