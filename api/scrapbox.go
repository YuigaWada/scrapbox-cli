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

// for json parser
type rawPage struct {
	Title   string   `json:"title"`
	Id      string   `json:"id"`
	LinksLc []string `json:"linksLc"`
}

func (p rawPage) convert() Page {
	return Page{Title_: p.Title}
}

// ** struct **
type Scrapbox struct {
	Project  string
	Pages    []Page
	paginate Paginate
}

type Page struct {
	// BaseUrl string
	Title_  string
	Content string
	Links   []Link
	apiUrl  string
	Sbox    *Scrapbox
}

type Link struct {
	Title string
	Tag   string
}

type Paginate struct {
	skip  int
	limit int
}

// ** var / util **

var scrapLinkRegex = regexp.MustCompile(`\[([^($|\*)][^(\[|\])]+)\]`)
var boldRegex = regexp.MustCompile(`\[\*\s([^(\[|\])]+)\]`)
var _ list.DefaultItem = (*Page)(nil)

func (p Page) Description() string {
	res, err := url.PathUnescape(p.apiUrl)
	if err != nil {
		panic("failed to unescape url")
	}
	return res
}

func (s Scrapbox) getReadApi() string {
	return fmt.Sprintf("https://scrapbox.io/api/pages/%s", s.Project)
}

func (s Scrapbox) getTextApi(title string) string {
	return fmt.Sprintf("https://scrapbox.io/api/pages/%s/%s/text", s.Project, url.PathEscape(title))
}

func (s Scrapbox) getDetailApi(title string) string {
	return fmt.Sprintf("https://scrapbox.io/api/pages/%s/%s", s.Project, url.PathEscape(title))
}

// ** func **

func (p *Page) Read(mainColor lipgloss.Color) (*Page, error) {
	const errMsg = "Failed to open URL .... :("
	if len(p.apiUrl) == 0 {
		return p, fmt.Errorf(errMsg)
	}

	resp, err := http.Get(p.apiUrl)
	if err != nil {
		return p, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return p, err
	}

	p.Content = string(body)
	return p.parse(mainColor), nil
}

func (p Page) GetNhopLinks() ([]Link, error) {
	resp2, err := http.Get(p.Sbox.getDetailApi(p.Title_))
	if err != nil {
		return nil, err
	}

	defer resp2.Body.Close()
	var linkPages []rawPage
	result := struct {
		RelatedPage struct {
			Links1hop []rawPage `json:"links1hop"`
			Links2hop []rawPage `json:"links2hop"`
		} `json:"relatedPages"`
	}{}

	err = json.NewDecoder(resp2.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	linkPages = append(linkPages, result.RelatedPage.Links1hop...)
	linkPages = append(linkPages, result.RelatedPage.Links2hop...)

	links := []Link{}
	for _, link := range linkPages {
		tag := ""
		if len(link.LinksLc) > 0 {
			tag = link.LinksLc[0]
		}
		links = append(links, Link{Title: link.Title, Tag: tag})
	}

	return links, nil
}

func MakePage(s *Scrapbox, title string) Page {
	return Page{Title_: title, apiUrl: s.getTextApi(title), Sbox: s}
}

func (s *Scrapbox) Read() []Page {
	url := s.getReadApi()
	url = fmt.Sprintf("%s?skip=%d&limit=%d", url, s.paginate.skip, s.paginate.limit)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	var pages []Page
	result := struct{ Pages []rawPage }{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	pages = []Page{}
	for _, p := range result.Pages {
		pages = append(pages, p.convert())
	}

	for i, page := range pages {
		pages[i].apiUrl = s.getTextApi(page.Title_)
		pages[i].Sbox = s
	}

	s.paginate.skip += len(pages)
	return pages
}
func (s *Scrapbox) Write(title string, body string) {

}

func isSpace(target rune) bool {
	return unicode.IsSpace(target) || target == 0x3000
}

func (page *Page) parse(mainColor lipgloss.Color) *Page {
	slice := strings.Split(page.Content, "\n")
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

	page.Content = strings.Join(slice[1:], "\n")
	renderBold(page)
	renderLinks(page, mainColor)

	return page
}

func renderLinks(page *Page, mainColor lipgloss.Color) {
	var style = lipgloss.NewStyle().Foreground(mainColor)
	r := func(body string, matched string) string {
		decoLink := fmt.Sprintf("[%s]", string(matched))
		link := Link{Title: matched, Tag: ""}
		if !contains(page.Links, link) {
			page.Links = append(page.Links, link)
		}
		return strings.Replace(body, decoLink, style.Render(matched), -1)
	}

	render(scrapLinkRegex, page, r)
}

func renderBold(page *Page) {
	var style = lipgloss.NewStyle().Bold(true)
	r := func(body string, matched string) string {
		return strings.Replace(body, fmt.Sprintf("[* %s]", matched), style.Render(matched), -1)
	}

	render(boldRegex, page, r)
}

func render(regex *regexp.Regexp, page *Page, renderAction func(string, string) string) {
	patterns := regex.FindAllStringSubmatch(page.Content, -1)
	if len(patterns) == 0 {
		return
	}

	for _, pattern := range patterns {
		for i, matched := range pattern {
			if i == 0 {
				continue
			}
			page.Content = renderAction(page.Content, matched)
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

func (p Page) Title() string {
	return p.Title_
}

func (p Page) FilterValue() string {
	return p.Title_
}
