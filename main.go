package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/rss"
)

type Newpaper struct {
	Title   string
	Content string
	Source  string
	PubDate string
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Hello(w http.ResponseWriter, req *http.Request) {
	db, err := sql.Open("sqlite3", "news/news.sqlite")
	check(err)
	defer db.Close()

	now := time.Now()
	twentyFourHoursAgo := now.Add(-24 * time.Hour)

	rows, err := db.Query("SELECT Title, Content, Date FROM news WHERE PubDate > ?", twentyFourHoursAgo)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	mapResponse := make([]map[string]string, 0)
	for rows.Next() {
		newpaper := Newpaper{}
		err := rows.Scan(&newpaper.Title, &newpaper.Content, &newpaper.PubDate)
		if err != nil {
			log.Fatal(err)
		}
		mapResponse = append(mapResponse, map[string]string{"title": newpaper.Title, "description": newpaper.Content})
	}

	// Marshal the response object into JSON format
	jsonData, err := json.Marshal(mapResponse)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}

	// Set content type and write the JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Write(jsonData)
}

func LoadNews() {

	db, err := sql.Open("sqlite3", "news/news.sqlite")
	check(err)
	defer db.Close()

	for {
		now := time.Now()

		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		sinceStartOfDay := now.Sub(startOfDay)

		hours := sinceStartOfDay / time.Hour

		sinceStartOfDay -= hours * time.Hour

		min := sinceStartOfDay / time.Minute

		sinceStartOfDay -= min * time.Minute

		sec := sinceStartOfDay / time.Second

		allTime := hours*3600 + min*60 + sec

		v := 18 * 3600

		if v-int(allTime) > 0 {
			fmt.Println("До запуска осталось: ", v-int(allTime))
			time.Sleep(time.Duration(v)*time.Second - allTime*time.Second)
		} else {
			fmt.Println("До запуска осталось: ", v-int(allTime)+24*3600)
			time.Sleep(time.Duration(v)*time.Second - allTime*time.Second + 24*3600*time.Second)
		}

		resp, err := http.Get("https://habr.com/ru/rss/articles/")
		if err != nil {
			log.Fatalln(err)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		sb := string(body)

		newsList := []Newpaper{}

		fp := rss.Parser{}
		rssFeed, _ := fp.Parse(strings.NewReader(sb))
		for _, item := range rssFeed.Items {

			a := Newpaper{
				Title:   item.Title,
				Content: item.Description,
				Source:  item.Link,
				PubDate: item.PubDate,
			}
			newsList = append(newsList, a)
		}

		resp, err = http.Get("https://www.reddit.com/.rss")
		if err != nil {
			log.Fatal(err)
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fpReddit := gofeed.NewParser()
		feed, err := fpReddit.ParseString(string(body))
		if err != nil {
			log.Fatal(err)
		}
		for _, item := range feed.Items {
			redditItem := Newpaper{
				Title:   item.Title,
				Source:  item.Link,
				Content: item.Content,
				PubDate: item.Published,
			}
			newsList = append(newsList, redditItem)

		}

		for _, news := range newsList {
			_, err := db.Exec("INSERT INTO news (title, content, source, date) VALUES (?, ?, ?, ?)", news.Title, news.Content, news.Source, news.PubDate)
			check(err)
		}
	}
}

func main() {

	// Запуск горутины для загрузки новостей
	go func() {
		// Запуск горутины для загрузки новостей
		LoadNews()
	}()
	http.HandleFunc("/news", Hello)
	http.ListenAndServe(":8080", nil)

}
