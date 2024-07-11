package main

import (
	"example/mikan_spider/spider"
	"fmt"
	"time"
)

func run(s *spider.Spider) {
	err := s.GetRss()
	if err != nil {
		fmt.Println(err)
	}

	err = s.SyncQb()
	if err != nil {
		fmt.Println(err)
	}
}

func main() {

	s, err := spider.NewSpider()
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(60 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Println("开始执行")
				run(s)
			}
		}
	}()
	select {}
}
