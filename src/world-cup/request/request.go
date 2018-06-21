package request

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var BaseURL string = "http://vtvgo.vn/worldcup2018"

// FetchChannel - fetch latest channel from vtvgo.vn
func FetchChannel(channel string) string {
	liveURL := getLiveURL(channel)

	client := &http.Client{}

	req, _ := http.NewRequest("GET", liveURL, nil)

	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.87 Safari/537.36")
	req.Header.Add("Referer", "http://vtvgo.vn/")
	req.Header.Add("Origin", "http://vtvgo.vn")

	resp, err := client.Do(req)
	if err != nil {
		panic("Can't get data")
	}

	defer resp.Body.Close()

	// convert response Body to string
	textData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	textContent := string(textData)
	textContent = strings.Replace(textContent, ",\n", ",\nhttp://localhost:8080/stream/"+channel+"/", -1)

	return textContent
}

// StreamData - get live content from channel url
func StreamData(channel string, fileStream string) []byte {
	currentDir := currentPath()
	cachedFile := filepath.Join(currentDir, "../caches", channel)
	data, _ := ioutil.ReadFile(cachedFile)

	streamURL := string(data)
	streamURL = strings.Replace(streamURL, channel+"-high.m3u8", fileStream, -1)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", streamURL, nil)

	// add header
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.87 Safari/537.36")
	req.Header.Add("Referer", "http://vtvgo.vn/")
	req.Header.Add("Origin", "http://vtvgo.vn")

	resp, err := client.Do(req)
	if err != nil {
		panic("Can't get data")
	}

	defer resp.Body.Close()

	// convert response Body to string
	streamData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return streamData
}

func getContent(url string) error {
	return nil
}

func getLiveURL(channel string) string {
	// cached
	currentDir := currentPath()
	cachedFile := filepath.Join(currentDir, "../caches", channel)
	fileInfo, err := os.Stat(cachedFile)

	if os.IsNotExist(err) == false {
		data, _ := ioutil.ReadFile(cachedFile)
		nowTime := time.Now()
		modTime := fileInfo.ModTime()

		expiredTime := nowTime.Unix() - 300

		if expiredTime < modTime.Unix() {
			return string(data)
		}
	}

	var channelURL string = BaseURL
	if channel == "vtv3" {
		channelURL = "http://vtvgo.vn/worldcup2018/vtv3.php"
	}

	resp, err := http.Get(channelURL)

	if err != nil {
		log.Fatal(err)
	}

	// wait for page loading finished.
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatal("Can't parse data from URL.")
	}

	// convert response Body to string
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	htmlStr := string(html)
	// @end convert.

	// fmt.Printf("%s", htmlStr)

	// parse HTML
	liveURLRegex, _ := regexp.Compile("(link = \")(.*)(\";)")
	liveURLMatch := liveURLRegex.FindStringSubmatch(htmlStr)

	// define playlistURL
	liveURL := liveURLMatch[2]
	liveURL = strings.Replace(liveURL, "mid.m3u8", "high.m3u8", -1)

	cachedContent := []byte(liveURL)
	ioutil.WriteFile(cachedFile, cachedContent, 0644)

	return liveURL
}

func currentPath() string {
	_, filename, _, _ := runtime.Caller(1)
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		log.Fatal(err)
	}

	return dir
}
