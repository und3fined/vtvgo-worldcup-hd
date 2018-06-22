package request

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

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
		panic("Can't get channel data from VTV")
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// delete cache file
		removeCache(channel)

		return FetchChannel(channel)
	}

	// convert response Body to string
	textData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	textContent := string(textData)

	go bufferData(textContent, liveURL, channel)
	go removeBufferExpired()

	textContent = strings.Replace(textContent, ",\n", ",\n/stream/"+channel+"/", -1)

	return textContent
}

// StreamData - get live content from channel url
func StreamData(channel string, fileStream string) []byte {
	currentDir := currentPath()
	// load buffer
	bufferFile := filepath.Join(currentDir, "../caches/buffer", fileStream)
	_, err := os.Stat(bufferFile)

	if os.IsNotExist(err) == false {
		log.Printf("Stream data: %s (cached)", fileStream)
		data, _ := ioutil.ReadFile(bufferFile)
		return data
	}

	log.Printf("Stream data: %s (vtv)", fileStream)

	cachedFile := filepath.Join(currentDir, "../caches", channel)
	data, _ := ioutil.ReadFile(cachedFile)

	streamURL := string(data)
	streamURL = strings.Replace(streamURL, channel+"-high.m3u8", fileStream, -1)

	return getStreamData(streamURL)
}

func getContent(url string) error {
	return nil
}

func getLiveURL(channel string) string {
	// cached
	currentDir := currentPath()
	cachedFile := filepath.Join(currentDir, "../caches", channel)
	_, err := os.Stat(cachedFile)

	if os.IsNotExist(err) == false {
		data, _ := ioutil.ReadFile(cachedFile)
		return string(data)
	}

	var channelURL = "http://vtvgo.vn/worldcup2018/index.php"

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
	log.Printf("Save channel cache: %s", channel)

	return liveURL
}

func getStreamData(streamURL string) []byte {
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

	if resp.StatusCode != 200 {
		var tmp []byte
		return tmp
	}

	// convert response Body to string
	streamData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return streamData
}

func bufferData(playlist string, liveURL string, channel string) {
	currentDir := currentPath()
	re := regexp.MustCompile("vtv\\d\\-(high|mid)-(\\d+).ts")
	listVideo := re.FindAllString(playlist, -1)
	lastItem := listVideo[len(listVideo)-1]

	indexRe := regexp.MustCompile("(vtv\\d+)(.*\\-)(\\d+)")
	bufferIndex := indexRe.FindAllStringSubmatch(lastItem, -1)[0]
	lastBuffer, _ := strconv.Atoi(bufferIndex[3])

	dataExtend1 := fmt.Sprintf("%s%s%d.ts", bufferIndex[1], bufferIndex[2], lastBuffer+1)
	dataExtend2 := fmt.Sprintf("%s%s%d.ts", bufferIndex[1], bufferIndex[2], lastBuffer+2)

	listVideo = append(listVideo, dataExtend1, dataExtend2)

	for _, video := range listVideo {
		bufferURL := strings.Replace(liveURL, channel+"-high.m3u8", video, -1)
		go func(url string, videoFile string) {
			bufferFile := filepath.Join(currentDir, "../caches/buffer", videoFile)
			_, err := os.Stat(bufferFile)

			if os.IsNotExist(err) {
				data := getStreamData(url)

				if len(data) != 0 {
					ioutil.WriteFile(bufferFile, data, 0644)
					log.Printf("Buffer: %s", videoFile)
				}
			}
		}(bufferURL, video)
	}
}

func removeCache(channel string) {
	currentDir := currentPath()
	cachedFile := filepath.Join(currentDir, "../caches", channel)
	_, err := os.Stat(cachedFile)

	if os.IsNotExist(err) == false {
		os.Remove(cachedFile)
	}
}

func removeBufferExpired() {
	currentDir := currentPath()
	bufferDir := filepath.Join(currentDir, "../caches/buffer")

	files, _ := ioutil.ReadDir(bufferDir)

	for _, f := range files {
		go func(f os.FileInfo) {
			bufferFilePath := filepath.Join(currentDir, "../caches/buffer", f.Name())
			fileInfo, _ := os.Stat(bufferFilePath)
			nowTime := time.Now()
			modTime := fileInfo.ModTime()
			expiredTime := nowTime.Unix() - 300

			if expiredTime < modTime.Unix() {
				os.Remove(bufferFilePath)
			}
		}(f)
	}
}

func currentPath() string {
	_, filename, _, _ := runtime.Caller(1)
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		log.Fatal(err)
	}

	return dir
}
