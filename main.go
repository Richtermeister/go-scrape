package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"time"
	"os"
	"github.com/anaskhan96/soup"
)

func getItems(url string, after time.Time) (items []gofeed.Item, err error)  {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)

	if err != nil {
		return items, err
	}

	for _, item := range feed.Items {
		if item.PublishedParsed.Before(after) {
			continue
		}

		items = append(items, *item)
	}

	return items, nil
}

func getSites() (links []string) {
	resp, err := soup.Get("https://www.craigslist.org/about/sites")
	if err != nil {
		os.Exit(1)
	}
	doc := soup.HTMLParse(resp)
	linkEls := doc.Find("div", "class", "box").FindAll("a")
	for _, link := range linkEls {
		links = append(links, link.Attrs()["href"])
	}

	return links
}

func genUrls(url string, subs []string, terms []string) (urls []string) {
	for _, sub := range subs {
		for _, term := range terms {
			urls = append(urls, url + "search/" + sub + "?format=rss&query=" + term)
		}
	}

	return urls
}

func main()  {
	sites := getSites()

	hoursAgo := 48
	concurrency := 10
	subs := []string{
		"crs",
		"crg",
	}
	terms := []string{
		"need+logo",
	}

	cutoff := time.Now().Add(- time.Hour * time.Duration(hoursAgo))

	hits := make(chan gofeed.Item)
	done := make(chan int)
	throttle := make(chan int, concurrency)

	go func() {
		for link := range hits {
			fmt.Println(link.Link)
			//fmt.Println(link.Description)
			fmt.Println(link.PublishedParsed)
		}
	}()

	urls := []string{}

	for _, site := range sites {
		urls = append(urls, genUrls(site, subs, terms)...)
	}

	for i, url := range urls {
		throttle<-1

		fmt.Println(fmt.Sprintf("[%d of %d] ", i + 1, len(urls)), "Searching " + url)

		go func(url string) {
			defer func() {
				<-throttle
				done<-1
			}()

			items, err := getItems(
				url,
				cutoff,
			)

			if err != nil {
				fmt.Println(err)
				return
			}

			for _, item := range items {
				hits<-item
			}
		}(url)
	}

	for i := 0; i < len(urls); i++ {
		<-done
	}
	close(hits)

	fmt.Println("Done")
}
