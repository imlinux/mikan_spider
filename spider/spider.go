package spider

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"github.com/autobrr/go-qbittorrent"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

//https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)

type Spider struct {
	qbClient *qbittorrent.Client

	httpClient *http.Client

	db *sql.DB
}

type DbItem struct {
	Title       string
	Episode     string
	TorrentUrl  string
	TorrentHash string
	State       string
}

type Rss struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Title       string        `xml:"title"`
	Link        string        `xml:"link"`
	Description string        `xml:"description"`
	Item        []ChannelItem `xml:"item"`
}

type ChannelItem struct {
	Guid        string    `xml:"guid"`
	Link        string    `xml:"link"`
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Enclosure   Enclosure `xml:"enclosure"`
}

type Enclosure struct {
	Length int    `xml:"length,attr"`
	Url    string `xml:"url,attr"`
}

func initDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./db.data")
	if err == nil {
		_, err := db.Exec("create table if not exists rss_info(TorrentHash text not null primary key, Title text, Episode text, TorrentUrl text, State Text)")
		if err == nil {
			return db, nil
		}
	}
	return nil, err
}

func initQbClient() (*qbittorrent.Client, error) {
	client := qbittorrent.NewClient(qbittorrent.Config{
		Host:     "http://192.168.3.91:8181",
		Username: "admin",
		Password: "111111",
	})
	ctx := context.Background()
	err := client.LoginCtx(ctx)
	if err == nil {
		return client, err
	} else {
		return nil, err
	}
}

func (s *Spider) GetRss() error {

	re := regexp.MustCompile("\\[(.*?)\\]")
	resp, err := s.httpClient.Get("https://mikanani.me/RSS/MyBangumi?token=ymv8U4E41B76gefoZPFlBw%3d%3d")
	if err != nil {
		return err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	rss := Rss{}
	err = xml.Unmarshal(data, &rss)

	if err != nil {
		return err
	}

	s2Items := make(map[string]ChannelItem)
	if rss.Channel.Item != nil {
		for _, i := range rss.Channel.Item {
			r := re.FindAllString(i.Title, -1)
			v, ok := s2Items[r[2]]
			if !ok || (v.Enclosure.Length < i.Enclosure.Length) {
				s2Items[r[2]] = i
			}
		}
	}
	for k, v := range s2Items {
		dbItem := DbItem{
			Title:       v.Title,
			Episode:     k,
			TorrentUrl:  v.Enclosure.Url,
			TorrentHash: torrentHash(v.Enclosure.Url),
			State:       "Init",
		}
		err := s.saveRss(dbItem)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Spider) SyncQb() error {
	rows, err := s.db.Query("select TorrentUrl from rss_info where State='Init'")
	if err != nil {
		return err
	}
	for rows.Next() {
		var TorrentUrl string
		err = rows.Scan(&TorrentUrl)
		if err != nil {
			return err
		}
		resp, err := s.httpClient.Get(TorrentUrl)
		if err != nil {
			return err
		}
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = s.qbClient.AddTorrentFromMemory(respBody, map[string]string{})
		if err != nil {
			return err
		}
	}
	rows.Close()
	torrent, err := s.qbClient.GetTorrents(qbittorrent.TorrentFilterOptions{})
	if err != nil {
		return err
	}
	for _, t := range torrent {
		if t.State == "pausedUP" {
			_, err := s.db.Exec("update rss_info set State=? where TorrentHash=?", t.State, t.Hash)
			if err != nil {
				return err
			}

			srcFile := strings.ReplaceAll(t.ContentPath, "/downloads", "/volume2/video/download/")
			dstFilename, err := s.filename(t.Hash, t.Name)
			if err != nil {
				return err
			}
			err = os.Rename(srcFile, "/volume2/video/video/海贼王/"+dstFilename)
			if err != nil {
				return err
			}
			err = s.qbClient.DeleteTorrents([]string{t.Hash}, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Spider) filename(torrentHash string, filename string) (string, error) {
	row := s.db.QueryRow("select Episode from rss_info where TorrentHash=?", torrentHash)
	var Episode string
	err := row.Scan(&Episode)
	if err != nil {
		return "", err
	}
	items := strings.Split(filename, ".")
	var ret string
	if len(items) > 1 {
		ret = Episode + "." + items[1]
	} else {
		ret = Episode
	}
	return ret, nil
}

func (s *Spider) saveRss(item DbItem) error {
	r, err := s.db.Query("select count(1) from rss_info where TorrentHash=?", item.TorrentHash)
	if err != nil {
		return err
	}

	r.Next()
	var cnt int
	err = r.Scan(&cnt)
	r.Close()

	if err != nil {
		return nil
	}

	if cnt == 0 {
		fmt.Println("保存: ", item)
		_, err := s.db.Exec("insert into rss_info(TorrentHash, Title, Episode, TorrentUrl, State) values (?, ?, ?, ?, ?)", item.TorrentHash, item.Title, item.Episode, item.TorrentUrl, item.State)
		return err
	}
	return err
}

func torrentHash(torrentUrl string) string {
	items := strings.Split(torrentUrl, "/")
	if len(items) > 0 {
		s := items[len(items)-1]
		items := strings.Split(s, ".")
		if len(items) > 0 {
			return items[0]
		}
	}
	return ""
}
func initHttpClint() (*http.Client, error) {
	proxyServer := "socks5://192.168.3.91:1080"
	proxyServerUrl, err := url.Parse(proxyServer)

	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyServerUrl),
	}

	client := &http.Client{
		Transport: transport,
	}
	return client, nil
}

func NewSpider() (*Spider, error) {
	httpClient, err := initHttpClint()
	if err != nil {
		return nil, err
	}
	qbClient, err := initQbClient()
	if err != nil {
		return nil, err
	}
	db, err := initDb()
	if err != nil {
		return nil, err
	}
	spider := Spider{
		qbClient:   qbClient,
		db:         db,
		httpClient: httpClient,
	}

	return &spider, nil
}
