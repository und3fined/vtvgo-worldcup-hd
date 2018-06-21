package main // import "world-cup"

import (
	"flag"
	"fmt"
	"log"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"world-cup/request"
)

var (
	addr     = flag.String("addr", ":8080", "TCP address to listen to")
	compress = flag.Bool("compress", false, "Whether to enable transparent response compression")
)

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
	fmt.Fprint(ctx, "Welcome!\n")
}

func getHello(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "hello, %s!\n", ctx.UserValue("name"))
}

func getLiveChannel(ctx *fasthttp.RequestCtx) {
	channel := fmt.Sprintf("%s", ctx.UserValue("channel"))
	content := request.FetchChannel(channel)

	ctx.SetContentType("text/plain charset=UTF-8")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody([]byte(content))
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

}
