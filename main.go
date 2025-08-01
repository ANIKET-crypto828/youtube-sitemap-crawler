package main

import (
"fmt"
"log"
"net/http"
"strings"
"math/rand"
"time"
"github.com/PuerkitoBio/goquery"
)

type SeoData struct{
	URL       string
	Title     string
	H1				string
	MetaDescription string
	StatusCode int
} 

type Parser interface{
getSEOData(resp *http.Response) (SeoData, error)
}

type DefaultParser struct{

}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/10.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:56.0) Gecko/20100101 Firefox/56.0",
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/537.36",
}

func randomUserAgent() string {
	rand.Seed(time.Now().UnixNano())
	randNum := rand.Intn(len(userAgents))
	return userAgents[randNum]
}

func isSitemap(urls []string) ([]string, []string){
	 sitemapFiles := []string{}
	 pages := []string{}
	 for _,page := range urls{
		foundSitemap := strings.Contains(page, "xml")
		  if foundSitemap == true{
				fmt.Println("Found sitemap:", page)
				sitemapFiles = append(sitemapFiles, page)
			} else {
				pages = append(pages, page)
			}
	 }
	 return sitemapFiles, pages
}

func extractSiteMapURLs(startURL string) []string {
	worklist := make(chan []string)
	toCrawl := []string{}
  var n int
  n++
  go func(){worklist <- []string{startURL}}()

	for ; n>0 ; n--{

	list := <-worklist
	for _, link := range list {
		n++
		go func(link string){
			response, err := makeRequest(link)
			if err != nil {
				log.Printf("Error retrieving URL: %s", link)
			}
			urls, _ := extractUrls(response)
			if err != nil {
				log.Printf("Error extracting document from response, URL:%s", link)
			}
			sitemapFiles, pages := isSitemap(urls)
			if sitemapFiles != nil {
				worklist <- sitemapFiles
			}
			for _, page := range pages{
				toCrawl = append(toCrawl, page)
			}
		}(link)
	}
}
	return toCrawl
}


func makeRequest(url string) (*http.Response, error) {
client := http.Client{
	Timeout: 10 * time.Second,
}
req, err := http.NewRequest("GET", url, nil)
req.Header.Set("User-Agent", randomUserAgent())
if err != nil {
		return nil, err
}

res, err := client.Do(req)
if err != nil {
		return nil, err
}
return res, nil
}

func scrapeURLs(urls []string, parser Parser, concurrency int)[]SeoData {

  tokens := make(chan struct{}, concurrency)
	var n int 
	worklist := make(chan []string) 
	results := []SeoData{}

	n++
	go func() {worklist <- urls}()
	for ; n > 0; n--{
		list := <-worklist
		for _, url := range list {
			if url != "" {
				n++
				go func(url string, token chan struct{}) {
           log.Printf("Requesting URL: %s", url)
					 res, err := scrapePage(url, tokens, parser)
					 if err != nil {
						 log.Printf("Encountered error, URL:%s", url)
					 } else {
						 results = append(results, res)
					 }
					 worklist <- []string{url}
				}(url, tokens)
			}
		}
	}
	return results
}

func extractUrls(response *http.Response) ([]string, error) {
	doc, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return nil, err
	}
	results := []string{}
	sel := doc.Find("loc")
	for i := range sel.Nodes {
	loc := sel.Eq(i).Text()
	results = append(results, loc)
}
	return results, nil
}

func scrapePage(url string, tokens chan struct{},parser Parser)(SeoData, error) {
  
	 res, err := crawlPage(url, tokens)
	 if err != nil {
		 return SeoData{}, err
	 }
	 data, err := parser.getSEOData(res)
	 if err != nil {
		 return SeoData{}, err
	 }
	 return data, nil
}

func crawlPage(url string, tokens chan struct{})(*http.Response, error) {
   
	  tokens <- struct{}{}
		resp, err := makeRequest(url)
		<- tokens
		if err != nil {
			return nil, err
		}
		return resp, err
}

func (d DefaultParser)getSEOData(resp *http.Response)(SeoData, error) {
		doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			return SeoData{}, err
		}
		result := SeoData{}
		result.URL = resp.Request.URL.String()
		result.StatusCode = resp.StatusCode
		result.Title = doc.Find("title").First().Text()
		result.H1 = doc.Find("h1").First().Text()
		desc, _ := doc.Find("meta[name^=description]").Attr("content")
    result.MetaDescription = desc
		return result, nil
}



func ScrapeSiteMap(url string, parser Parser, concurrency int) []SeoData{
	results := extractSiteMapURLs(url) 
	res := scrapeURLs(results, parser, concurrency)
	return res
}

func main() {
	p := DefaultParser{}
	results := ScrapeSiteMap("https://www.quicksprout.com/sitemap.xml", p, 10)
	for _, result := range results {
	fmt.Println(result)
}

}