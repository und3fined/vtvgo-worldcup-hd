package main // import "world-cup"

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"world-cup/request"
)

var (
	addr     = flag.String("addr", ":8080", "TCP address to listen to")
	compress = flag.Bool("compress", false, "Whether to enable transparent response compression")
)

var activeChannel = []string{"vtv2", "vtv3", "vtv6"}

func main() {
	flag.Parse()

	router := fasthttprouter.New()
	router.GET("/", getIndex)
	router.GET("/hello/:name", getHello)
	router.GET("/watch/:channel", getWatchChannel)
	router.GET("/live/:channel", getLiveChannel)
	router.GET("/stream/:channel/:file", getStreamFile)

	handler := router.Handler

	if *compress {
		handler = fasthttp.CompressHandler(handler)
	}

	log.Printf("Server running with `%s`", *addr)

	if err := fasthttp.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func getIndex(ctx *fasthttp.RequestCtx) {
	ctx.Redirect("/watch/vtv6", fasthttp.StatusMovedPermanently)
}

func getHello(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "hello, %s!\n", ctx.UserValue("name"))
}

func getLiveChannel(ctx *fasthttp.RequestCtx) {
	channel := fmt.Sprintf("%s", ctx.UserValue("channel"))

	ctx.SetContentType("text/plain charset=UTF-8")
	ctx.SetStatusCode(fasthttp.StatusOK)

	if request.IndexOf(channel, activeChannel) == -1 {
		ctx.SetBody([]byte("Channel not supported"))
	} else {
		content := request.FetchChannel(channel)
		ctx.SetBody([]byte(content))
	}
}

func getStreamFile(ctx *fasthttp.RequestCtx) {
	channel := fmt.Sprintf("%s", ctx.UserValue("channel"))
	file := fmt.Sprintf("%s", ctx.UserValue("file"))

	content := request.StreamData(channel, file)

	ctx.SetContentType("video/mp2t")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(content)
}

func getWatchChannel(ctx *fasthttp.RequestCtx) {
	channel := fmt.Sprintf("%s", ctx.UserValue("channel"))

	ctx.SetStatusCode(fasthttp.StatusOK)

	if request.IndexOf(channel, activeChannel) == -1 {
		ctx.SetContentType("text/plain charset=UTF-8")
		ctx.SetBody([]byte("Channel not supported"))
	} else {
		currentDir := currentPath()
		templateFile := filepath.Join(currentDir, "./template.html")
		data, _ := ioutil.ReadFile(templateFile)

		content := strings.Replace(string(data), "__STREAM_URL__", "/live/"+channel, -1)
		content = strings.Replace(content, "__CHANNEL__", strings.ToUpper(channel), -1)

		ctx.SetContentType("text/html charset=UTF-8")
		ctx.SetBody([]byte(content))
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
