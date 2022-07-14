package api

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
	Title_  string   `json:"title"`
	ID      string   `json:"id"`
	LinksLc []string `json:"linksLc"`
	BaseUrl string
	ApiUrl  string
	User    ScrapUser
}

type DetailPage struct {
	RelatedPage RelatedPage `json:"relatedPages"`
}

type RelatedPage struct {
	Links1hop []Page `json:"links1hop"`
	Links2hop []Page `json:"links2hop"`
}

var scrapLinkRegex = regexp.MustCompile(`\[([^($|\*)][^(\[|\])]+)\]`)
var boldRegex = regexp.MustCompile(`\[\*\s([^(\[|\])]+)\]`)

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

	content := ScrapboxPage{Content: string(body)}
	return content.parse(mainColor), nil
}

func (p Page) GetNhopLinks() ([]Page, error) {
	resp2, err := http.Get(p.User.getDetailApi(p.Title_))
	if err != nil {
		return nil, err
	}

	defer resp2.Body.Close()
	var linkPages []Page
	result := DetailPage{}
	err = json.NewDecoder(resp2.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	linkPages = append(linkPages, result.RelatedPage.Links1hop...)
	linkPages = append(linkPages, result.RelatedPage.Links2hop...)
	return linkPages, nil
}

func MakePage(user ScrapUser, title string) Page {
	return Page{Title_: title, ApiUrl: user.getTextApi(title)}
}

var _ list.DefaultItem = (*Page)(nil)

type ScrapUser struct {
	Project string
}

func (user ScrapUser) getReadApi() string {
	return fmt.Sprintf("https://scrapbox.io/api/pages/%s", user.Project)
}

func (user ScrapUser) getTextApi(title string) string {
	return fmt.Sprintf("https://scrapbox.io/api/pages/%s/%s/text", user.Project, url.PathEscape(title))
}

func (user ScrapUser) getDetailApi(title string) string {
	return fmt.Sprintf("https://scrapbox.io/api/pages/%s/%s", user.Project, url.PathEscape(title))
}

type Pager struct {
	skip  int
	limit int
}

func MakePager() Pager {
	return Pager{0, 500}
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
		pages[i].ApiUrl = user.getTextApi(page.Title_)
		pages[i].User = user
	}

	return pages
}
func (p Pager) Write(title string, body string) {

}

type ScrapboxPage struct {
	Content string
	Links   []Link
}

type Link struct {
	Title string
	Tag   string
}

func isSpace(target rune) bool {
	return unicode.IsSpace(target) || target == 0x3000
}

func (spage *ScrapboxPage) parse(mainColor lipgloss.Color) ScrapboxPage {
	slice := strings.Split(spage.Content, "\n")
	dotStr := lipgloss.NewStyle().Bold(true).Render("・")
	for i, str := range slice {
		str := []rune(str)
		if i == 0 {
			continue
		}
		if len(str) == 0 || !isSpace(str[0]) {
			continue
		}
		for j := 1; j < len(str); j++ {
			if !isSpace(str[j]) {
				s := append(str[:j], append([]rune(dotStr), str[j:]...)...)
				slice[i] = string(s)
				break
			}
		}
	}

	*spage = ScrapboxPage{strings.Join(slice[1:], "\n"), spage.Links}
	renderBold(spage)
	renderLinks(spage, mainColor)

	return *spage
}

func renderLinks(spage *ScrapboxPage, mainColor lipgloss.Color) {
	var style = lipgloss.NewStyle().Foreground(mainColor)
	r := func(body string, matched string) string {
		decoLink := fmt.Sprintf("[%s]", string(matched))
		link := Link{Title: matched, Tag: ""}
		if !contains(spage.Links, link) {
			spage.Links = append(spage.Links, link)
		}
		return strings.Replace(body, decoLink, style.Render(matched), -1)
	}

	render(scrapLinkRegex, spage, r)
}

func renderBold(spage *ScrapboxPage) {
	var style = lipgloss.NewStyle().Bold(true)
	r := func(body string, matched string) string {
		return strings.Replace(body, fmt.Sprintf("[* %s]", matched), style.Render(matched), -1)
	}

	render(boldRegex, spage, r)
}

func render(regex *regexp.Regexp, spage *ScrapboxPage, renderAction func(string, string) string) {
	patterns := regex.FindAllStringSubmatch(spage.Content, -1)
	if len(patterns) == 0 {
		return
	}

	for _, pattern := range patterns {
		for i, matched := range pattern {
			if i == 0 {
				continue
			}
			spage.Content = renderAction(spage.Content, matched)
		}
	}
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
