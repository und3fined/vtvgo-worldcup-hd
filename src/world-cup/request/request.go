package request

import (
	"bytes"
	"fmt"
	// "io"
	"crypto/tls"
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

var downloadCaching []string

// FetchChannel - fetch latest channel from vtvgo.vn
func FetchChannel(channel string) string {
	log.Printf("--- Fetch %s: start", channel)
	go removeBufferExpired()

	liveURL := getLiveURL(channel)
	var totalDuration float64

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

	durationRe, _ := regexp.Compile("(TARGETDURATION|EXTINF)\\:([0-9]\\.?([0-9]+)?)")
	parseDuration := durationRe.FindAllStringSubmatch(textContent, -1)

	for i := 1; i < len(parseDuration); i++ {
		durtation, _ := strconv.ParseFloat(parseDuration[i][2], 64)
		totalDuration += durtation
	}

	log.Printf("--- Duration target: %ss", parseDuration[0][2])
	log.Printf("--- Total duration: %.3fs", totalDuration)

	re, _ := regexp.Compile("(vtv\\d+)(.*\\-)(\\d+)(.*)")
	parseContent := re.FindAllStringSubmatch(textContent, -1)
	bufferURL := strings.Replace(liveURL, channel+"-high.m3u8", parseContent[0][0], -1)

	go bufferData(bufferURL, len(parseContent), 0)

	textContent = strings.Replace(textContent, ",\n", ",\n/stream/"+channel+"/", -1)

	log.Printf("--- Fetch %s: done", channel)
	return textContent
}

// StreamData - get live content from channel url
func StreamData(channel string, fileStream string) []byte {
	currentDir := currentPath()

	// load buffer
	channelFile := filepath.Join(currentDir, "../caches", channel)
	bufferFile := filepath.Join(currentDir, "../caches/buffer", fileStream)
	data, err := ioutil.ReadFile(bufferFile)

	if err != nil {
		// log.Printf("%s not found!", fileStream)

		channelURL, _ := ioutil.ReadFile(channelFile)
		re, _ := regexp.Compile("(.*)(vtv.*)")
		parseURL := re.FindAllStringSubmatch(string(channelURL), -1)[0]
		streamURL := fmt.Sprintf("%s%s", parseURL[1], fileStream)

		getStreamData(streamURL)
		// go bufferData(streamURL, 3, 1)

		time.Sleep(300 * time.Millisecond)
		return StreamData(channel, fileStream)
	}

	return data
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

	var channelURL = "https://vtvgo.vn/worldcup2018/index.php"

	if channel == "vtv3" {
		channelURL = "https://vtvgo.vn/worldcup2018/vtv3.php"
	}

	if channel == "vtv2" {
		channelURL = "https://vtvgo.vn/xem-truc-tuyen-kenh-vtv2-2.html"
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
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

	// parse HTML
	liveURLRegex, _ := regexp.Compile("(https?\\:(.*).m3u8)")
	liveURLMatch := liveURLRegex.FindStringSubmatch(htmlStr)

	// define playlistURL
	liveURL := liveURLMatch[0]
	liveURL = strings.Replace(liveURL, "mid.m3u8", "high.m3u8", -1)

	cachedContent := []byte(liveURL)
	ioutil.WriteFile(cachedFile, cachedContent, 0644)
	log.Printf("Save channel cache: %s", channel)

	return liveURL
}

func getStreamData(streamURL string) error {
	currentDir := currentPath()
	re := regexp.MustCompile("(.*)(vtv\\d+)(.*\\-)(\\d+)(.*)")
	parseURL := re.FindAllStringSubmatch(streamURL, -1)[0]
	videoFile := fmt.Sprintf("%s%s%s%s", parseURL[2], parseURL[3], parseURL[4], parseURL[5])
	bufferFilePath := filepath.Join(currentDir, "../caches/buffer", videoFile)

	if IndexOf(videoFile, downloadCaching) != -1 {
		return nil
	}

	downloadCaching = append(downloadCaching, videoFile)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", streamURL, nil)

	// add header
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.87 Safari/537.36")
	req.Header.Add("Referer", "http://vtvgo.vn/")
	req.Header.Add("Origin", "http://vtvgo.vn")

	resp, err := client.Do(req)
	if err != nil {
		panic("Can't get stream data")
	}

	defer resp.Body.Close()

	// Read the content
	var bodyBytes []byte
	if resp.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(resp.Body)
	}

	// Restore the io.ReadCloser to its original state
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	ioutil.WriteFile(bufferFilePath, bodyBytes, 0644)

	cacheIndex := IndexOf(videoFile, downloadCaching)

	if cacheIndex != -1 {
		downloadCaching = append(downloadCaching[:cacheIndex], downloadCaching[cacheIndex+1:]...)
	}

	// log.Printf("%s downloaded!", videoFile)

	return nil
}

func bufferData(bufferURL string, entends int, start int) error {
	ch := make(chan string)
	buffCount := 0

	var listBuffer []string

	re := regexp.MustCompile("(.*)(vtv\\d+)(.*\\-)(\\d+)(.*)")
	parseURL := re.FindAllStringSubmatch(bufferURL, -1)[0]
	bufferIndex, _ := strconv.Atoi(parseURL[4])

	for i := start; i < entends; i++ {
		extends := fmt.Sprintf("%s%s%s%d%s", parseURL[1], parseURL[2], parseURL[3], bufferIndex+i, parseURL[5])
		listBuffer = append(listBuffer, extends)
	}

	currentDir := currentPath()

	for _, url := range listBuffer {
		parseURL := re.FindAllStringSubmatch(url, -1)[0]
		video := fmt.Sprintf("%s%s%s%s", parseURL[2], parseURL[3], parseURL[4], parseURL[5])

		go func(url string, videoFile string, ch chan string) {
			bufferFile := filepath.Join(currentDir, "../caches/buffer", videoFile)
			_, err := os.Stat(bufferFile)

			if os.IsNotExist(err) {
				buffCount++
				getStreamData(url)
				buffCount--

				ch <- parseURL[4]
			}
		}(url, video, ch)
	}

	waitStreamData(parseURL[4], &buffCount, ch)
	return nil
}

func waitStreamData(videoID string, buffCount *int, end chan string) {
	for {
		select {
		case <-end:
			// log.Printf("Waiting %d", *buffCount)

			if *buffCount <= 1 {
				return
			}
		}

		time.Sleep(100 * time.Millisecond)
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
			if f.Name() != ".gitkeep" {
				bufferFilePath := filepath.Join(currentDir, "../caches/buffer", f.Name())
				fileInfo, _ := os.Stat(bufferFilePath)
				nowTime := time.Now()
				modTime := fileInfo.ModTime()
				expiredTime := nowTime.Unix() - 120

				if modTime.Unix() < expiredTime {
					os.Remove(bufferFilePath)
				}
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

// IndexOf func util
func IndexOf(word string, data []string) int {
	for k, v := range data {
		if word == v {
			return k
		}
	}
	return -1
}
