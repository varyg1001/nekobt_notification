package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/dustin/go-humanize"
)

func loadCache(filename string) (UrlCache, error) {
	var config UrlCache
	byteValue, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("failed to load download config file: %w", err)
		return config, err
	}
	if err = json.Unmarshal(byteValue, &config); err != nil {
		err = fmt.Errorf("error parsing download config file: %w", err)
		return config, err
	}
	return config, nil
}

func (i *Item) HumanSize() string {
	s, err := strconv.ParseUint(i.FilesizeStr, 10, 64)
	if err != nil {
		return "Unknown Size"
	}
	return humanize.IBytes(s)
}

type UrlCache struct {
	LastUrls []string `json:"last_urls"`
}

type MainConfig struct {
	Proxies  map[string]string `json:"proxies"`
	BotToken string            `json:"bot_token"`
	ChatId   string            `json:"chat_id"`
	TopicId  string            `json:"topic_id"`
	Sleep    int               `json:"sleep"`
	Feed     string            `json:"feed"`
}

type Item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	FilesizeStr string `json:"filesize"`
	SubLevel    int    `json:"level"`
}

type Data struct {
	Items []Item `json:"results"`
}

type Feed struct {
	Error bool  `json:"error"`
	Data  *Data `json:"data"`
}

const telegramAPI = "https://api.telegram.org/bot%s/sendMessage"

func post(message, chatID, token, topicID string) {
	log := log.New(os.Stdout, "tg: ", log.LstdFlags)

	if message != "" {
		retries := 0
		maxRetries := 5

		for retries < maxRetries {
			baseURL := fmt.Sprintf(telegramAPI, token)
			params := url.Values{}
			params.Set("text", message)
			params.Set("chat_id", chatID)
			if topicID != "" {
				params.Set("message_thread_id", topicID)
			}
			params.Set("parse_mode", "html")
			params.Set("disable_web_page_preview", "true")

			req, err := http.NewRequest("POST", baseURL, strings.NewReader(params.Encode()))
			if err != nil {
				log.Printf("Error creating request: %v", err)
				continue
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := http.DefaultClient.Do(req)

			if err != nil {
				log.Printf("Warning: %v", err)
				retries++

				if retries < maxRetries {
					log.Printf("Retrying (%d/%d)", retries, maxRetries)
					time.Sleep(time.Duration(retries*retries) * time.Second)
					continue
				}

				log.Printf("[ERROR]: Max retries reached..")
				break
			}

			resp.Body.Close()

			if resp.StatusCode >= 400 {
				log.Printf("[WARNING]: HTTP %d response", resp.StatusCode)
				retries++

				if retries < maxRetries {
					log.Printf("Retrying (%d/%d)", retries, maxRetries)
					time.Sleep(time.Duration(retries*retries) * time.Second)
					continue
				}

				log.Printf("[ERROR]: Max retries reached..")
				break
			}
			break
		}
	}
}

func writeCache(filename string, cache UrlCache) error {
	if len(cache.LastUrls) > 5 {
		cache.LastUrls = cache.LastUrls[len(cache.LastUrls)-5:]
	}
	jsonData, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		err = fmt.Errorf("error encoding json: %w", err)
		return err
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		err = fmt.Errorf("error writing json: %w", err)
		return err
	}

	return nil
}

func getRandomProxy(proxies map[string]string) string {
	var proxy_code string
	k := rand.Intn(len(proxies))
	i := 0
	for _, value := range proxies {
		if i == k {
			proxy_code = value
			break
		}
		i++
	}

	return proxy_code
}

func getMainConfig(filename string) (MainConfig, error) {
	var config MainConfig
	byteValue, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("failed to load config file: %w", err)
		return config, err
	}
	if err = json.Unmarshal(byteValue, &config); err != nil {
		err = fmt.Errorf("error parsing config file: %w", err)
		return config, err
	}
	return config, nil
}

func getFeed(mainConfig MainConfig) (Feed, error) {
	var feed Feed

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithClientProfile(profiles.Chrome_124),
	}

	if len(mainConfig.Proxies) > 0 {
		proxy := getRandomProxy(mainConfig.Proxies)
		options = append(options, tls_client.WithProxyUrl(proxy))
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return feed, err
	}

	req, err := fhttp.NewRequest(http.MethodGet, mainConfig.Feed, nil)
	if err != nil {
		return feed, err
	}

	req.Header = fhttp.Header{
		"accept":     {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"user-agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/20100101 Firefox/138.0"},
	}

	resp, err := client.Do(req)
	if err != nil {
		return feed, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return feed, err
	}

	return feed, nil
}

func main() {
	log := log.New(os.Stdout, "nekobt_notification: ", log.LstdFlags)
	mainConfig, err := getMainConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	var CacheFile UrlCache
	if _, err := os.Stat("cache.json"); err == nil {
		CacheFile, err = loadCache("cache.json")
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Bot started")

	for {
		feed, err := getFeed(mainConfig)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Println("[ERROR] ", err)
				time.Sleep(time.Duration(mainConfig.Sleep*2) * time.Second)
				continue
			} else {
				log.Println("[ERROR] ", err)
			}
		}

		if feed.Error {

		}

		var CleanItems []Item
		for _, item := range feed.Data.Items {
			if slices.Contains(CacheFile.LastUrls, item.ID) {
				break
			}
			CleanItems = append(CleanItems, item)
		}

		for i := len(CleanItems) - 1; i >= 0; i-- {
			post(
				fmt.Sprintf(
					"\n<b>%v</b>\n\n- <b>Size</b>: %v\n- <b>Sub level</b>: %d\n- <b>Link</b>: <a href=\"https://nekobt.to/torrents/%v\">View site</a> | <a href=\"https://nekobt.to/api/v1/torrents/%v/download?public=true\">Torrent file</a>",
					CleanItems[i].Title,
					CleanItems[i].HumanSize(),
					CleanItems[i].SubLevel,
					CleanItems[i].ID,
					CleanItems[i].ID,
				),
				mainConfig.ChatId,
				mainConfig.BotToken,
				mainConfig.TopicId,
			)
			log.Println("Title: ", CleanItems[i].Title)
			CacheFile.LastUrls = append(CacheFile.LastUrls, CleanItems[i].ID)
			if err := writeCache("cache.json", CacheFile); err != nil {
				log.Println("[ERROR] ", err)
				continue
			}
		}

		time.Sleep(time.Duration(mainConfig.Sleep*2) * time.Second)
	}

}
