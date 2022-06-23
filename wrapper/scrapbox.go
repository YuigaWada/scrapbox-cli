package wrapper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
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
	User    *ScrapUser
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

func (p Page) Read(mainColor lipgloss.Color) (ScrapboxPage, error) {
	const errMsg = "Failed to open URL .... :("
	if len(p.ApiUrl) == 0 {
		return ScrapboxPage{}, fmt.Errorf(errMsg)
	}

	resp, err := http.Get(p.ApiUrl)
	if err != nil {
		return ScrapboxPage{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ScrapboxPage{}, err
	}

	content := ScrapboxPage{string(body), []string{}}
	return content.parse(mainColor), nil
}

func MakePage(user *ScrapUser, title string) Page {
	return Page{Title_: title,
		ApiUrl: user.getDetailApi(title)}
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
		pages[i].User = &user
	}

	return pages
}
func (p Pager) Write(title string, body string) {

}

type ScrapboxPage struct {
	Content string
	Links   []string
}

func isSpace(target rune) bool {
	return unicode.IsSpace(target) || target == 0x3000
}

func (spage *ScrapboxPage) parse(mainColor lipgloss.Color) ScrapboxPage {
	rex := regexp.MustCompile(`\[([^($|\*)][^(\[|\])]+)\]`)

	slice := strings.Split(spage.Content, "\n")
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
			if !isSpace(str[j]) && j > 0 {
				slice[i] = slice[i][:j] + dotStr + slice[i][j:]
				break
			}
		}
	}

	*spage = ScrapboxPage{strings.Join(slice[1:], "\n"), []string{}}
	patterns := rex.FindAllStringSubmatch(spage.Content, -1)

	if len(patterns) == 0 {
		return *spage
	}

	var style = lipgloss.NewStyle().Foreground(mainColor)
	for _, pattern := range patterns {
		for i, link := range pattern {
			if i == 0 {
				continue
			}
			decoLink := fmt.Sprintf("[%s]", string(link))
			if !contains(spage.Links, link) {
				spage.Links = append(spage.Links, link)
			}
			spage.Content = strings.Replace(spage.Content, decoLink, style.Render(link), -1)
		}
	}

	return *spage
}

func contains(list interface{}, elem interface{}) bool {
	listV := reflect.ValueOf(list)

	if listV.Kind() == reflect.Slice {
		for i := 0; i < listV.Len(); i++ {
			item := listV.Index(i).Interface()
			if !reflect.TypeOf(elem).ConvertibleTo(reflect.TypeOf(item)) {
				continue
			}
			target := reflect.ValueOf(elem).Convert(reflect.TypeOf(item)).Interface()
			if ok := reflect.DeepEqual(item, target); ok {
				return true
			}
		}
	}
	return false
}
