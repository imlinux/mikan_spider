package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/autobrr/go-qbittorrent"
	_ "github.com/mattn/go-sqlite3"
	"io/fs"
	"log"
	"os"
	"testing"
)

func Test_1(t *testing.T) {
	client := qbittorrent.NewClient(qbittorrent.Config{
		Host:     "http://192.168.3.91:8181",
		Username: "admin",
		Password: "111111",
	})

	ctx := context.Background()

	if err := client.LoginCtx(ctx); err != nil {
		log.Fatalf("could not log into client: %q", err)
	}

	client.DeleteTorrents([]string{"a7dbefdee3dcc6fef1c4e29e01b3e654765f71f7"}, true)
}

func Test_2(t *testing.T) {
	db, err := sql.Open("sqlite3", "./db.data")
	if err == nil {
		_, err := db.Exec("create table if not exists rss_info(id text not null primary key, name text)")
		if err == nil {
			_, err := db.Exec("insert into rss_info values (?, ?)", "1", "xx")
			if err == nil {
				r, err := db.Query("select * from rss_info")
				if err == nil {
					var id, name string
					for r.Next() {
						r.Scan(&id, &name)
						fmt.Println(id, name)
					}
					return
				}
			}
		}
	}
	t.Fatal(err)
}

func Test_3(t *testing.T) {

	a := 1

	{
		a := 2
		fmt.Println(a)
	}

	fmt.Println(a)
}

func Test_4(t *testing.T) {
	err := fs.WalkDir(os.DirFS("/home/dong/Downloads"), ".", func(path string, d fs.DirEntry, err error) error {
		fmt.Println(path)
		return errors.New("ceshi")
	})
	fmt.Println(err)
}
