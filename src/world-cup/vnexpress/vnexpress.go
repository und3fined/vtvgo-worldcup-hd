package vnexpress

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	// "os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const matchURL = "https://thethao.vnexpress.net/tin-tuc/worldcup-2018/lich-thi-dau-wordcup-2018"

// Match struct
type Match struct {
	Time             int    `json:"time"`
	TeamOne          string `json:"teamOne"`
	TeamTwo          string `json:"teamTwo"`
	FlagOne          string `json:"flagOne"`
	FlagTwo          string `json:"flagTwo"`
	Channel          string `json:"channel"`
	ChannelThumbnail string `json:"thumbnail"`
}

// GetMatch func
func GetMatch() string {
	var contentText string
	currentDir := currentPath()
	// fileCachedPath := filepath.Join(currentDir, "../caches/vnexpress.html")
	matchesPath := filepath.Join(currentDir, "../caches/matches.json")
	// _, err := os.Stat(fileCachedPath)

	// if os.IsNotExist(err) == false {
	// 	data, _ := ioutil.ReadFile(fileCachedPath)
	// 	contentText = string(data)
	// } else {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", matchURL, nil)

	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:61.0) Gecko/20100101 Firefox/61.0")
	req.Header.Add("Referer", "https://thethao.vnexpress.net")
	req.Header.Add("Origin", "https://thethao.vnexpress.net")

	resp, err := client.Do(req)
	if err != nil {
		panic("Can't get channel data from VTV")
	}

	defer resp.Body.Close()

	// convert response Body to string
	contentByte, _ := ioutil.ReadAll(resp.Body)
	// ioutil.WriteFile(fileCachedPath, contentByte, 0644)

	contentText = string(contentByte)
	// }

	re, _ := regexp.Compile("(<li id=\"(\\d+)\")|(<img src=\"(.*)\" alt=\".+\">.+<h3>)(.*)(</h3>)|(<span></span>\t<h3>)(.*)(</h3>)|(.* src=\"(.*)\" class=\"icon_tvlive\")")
	parseContent := re.FindAllStringSubmatch(contentText, -1)
	var activeIndex = 0
	var matches []Match
	var match = Match{}

	for i := 0; i < len(parseContent); i++ {
		index := float64(i)
		item := math.Ceil(index / 4)
		if math.Mod(index, 4) == 0 {
			if i != 0 {
				matches = append(matches, match)
				match = Match{}
			}

			activeIndex = 0
			item++
		}

		if activeIndex == 0 {
			time, _ := strconv.Atoi(parseContent[i][2])
			match.Time = time
		}

		if activeIndex == 1 {
			if len(parseContent[i][5]) != 0 {
				teamOne := strings.Replace(parseContent[i][5], "<a>", "", -1)
				teamOne = strings.Replace(teamOne, "</a>", "", -1)

				match.TeamOne = teamOne
				match.FlagOne = parseContent[i][4]
			} else {
				teamOne := strings.Replace(parseContent[i][8], "<a>", "", -1)
				teamOne = strings.Replace(teamOne, "</a>", "", -1)
				match.TeamOne = teamOne
			}
		}

		if activeIndex == 2 {
			if len(parseContent[i][5]) != 0 {
				teamTwo := strings.Replace(parseContent[i][5], "<a>", "", -1)
				teamTwo = strings.Replace(teamTwo, "</a>", "", -1)
				match.TeamTwo = teamTwo
				match.FlagTwo = parseContent[i][4]
			} else {
				teamTwo := strings.Replace(parseContent[i][8], "<a>", "", -1)
				teamTwo = strings.Replace(teamTwo, "</a>", "", -1)
				match.TeamTwo = teamTwo
			}
		}

		if activeIndex == 3 {
			reChannel, _ := regexp.Compile("VTV-\\d")
			channelParse := reChannel.FindStringSubmatch(parseContent[i][11])

			match.Channel = strings.Replace(channelParse[0], "VTV-", "vtv", -1)
			match.ChannelThumbnail = parseContent[i][11]
		}

		activeIndex++
	}

	matchesJSON, _ := json.MarshalIndent(matches, "", "\t")
	ioutil.WriteFile(matchesPath, matchesJSON, 0644)

	return ""
}

func currentPath() string {
	_, filename, _, _ := runtime.Caller(1)
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		log.Fatal(err)
	}

	return dir
}
