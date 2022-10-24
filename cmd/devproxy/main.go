package main

import (
	"context"
	"fmt"
	// "math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Version is set during build
var Version = ""

// Rev is set during build
var Rev = ""

// proxy target
type proxyTarget struct {
	pathPrefix string
	url        *url.URL
}

func run(cliCtx *cli.Context) error {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	addr := fmt.Sprintf(":%d", cliCtx.Uint("port"))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	var targets []proxyTarget
	args := append([]string{cliCtx.Args().First()}, cliCtx.Args().Tail()...)
	for _, target := range args {
		split := strings.Split(target, "@")

		// split must contain a prefix path and a target url
		if len(split) != 2 {
			log.Warnf("invalid target %q ignored", target)
			continue
		}

		// path prefix must not be empty
		prefix := split[0]
		if prefix == "" {
			log.Warnf("empty prefix %q ignored", prefix)
			continue
		}

		// target url must be a valid url
		targetURL, err := url.Parse(split[1])
		if err != nil {
			log.Warnf("invalid url %q ignored", split[1])
			continue
		}
		targets = append(targets, proxyTarget{
			pathPrefix: prefix,
			url:        targetURL,
		})
	}

	// check for at least one valid target to proxy
	if len(targets) == 0 {
		return fmt.Errorf("no valid targets to proxy")
	}

	longestUrl := ""
	taken := make(map[uint]bool)
	colormap := make(map[*url.URL]uint)

	// compute colors and padding
	for _, target := range targets {
		if len(target.url.String()) > len(longestUrl) {
			longestUrl = target.url.String()
		}
		color := stringToColorCode(target.url.String(), ansi16ColorPalette)
		if _, ok := taken[color]; !ok {
			taken[color] = true
			colormap[target.url] = color
		}
	}

	// create a random color palette
	pal := ansi16ColorPalette[:]
	// rand.Seed(time.Now().UnixNano())
	// rand.Shuffle(len(pal), func(i, j int) {
	// 	pal[i], pal[j] = pal[j], pal[i]
	// })

	// randomly assign colors to unassigned
	i := 0
	for _, target := range targets {
		if _, assigned := colormap[target.url]; !assigned {
			colormap[target.url] = pal[i]
			i += 1
			i = i % len(pal)
		}
	}

	proxyRouter := mux.NewRouter()
	for _, target := range targets {
		addCORS := func(res *http.Response) error {
			res.Header.Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE")
			res.Header.Set("Access-Control-Allow-Credentials", "true")
			res.Header.Set("Access-Control-Allow-Origin", "*")
			return nil
		}

		reverse := httputil.NewSingleHostReverseProxy(target.url)
		reverse.ModifyResponse = addCORS

		metadata := fmtProxyTarget{
			proxyTarget: target,
			color:       colormap[target.url],
			pad:         uint(len(longestUrl)),
		}
		ctx := context.WithValue(
			context.Background(),
			fmtProxyTargetKey{},
			metadata,
		)

		handler := WithLogging(ctx, http.StripPrefix(target.pathPrefix, reverse))
		proxyRouter.PathPrefix(target.pathPrefix).Handler(handler)
	}

	log.SetFormatter(&myFormatter{log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	}})

	server := &http.Server{Handler: proxyRouter}
	go func() {
		<-shutdownChan
		log.Warnf("shutdown ...")
		server.Shutdown(context.Background())
	}()

	log.Infof("listening on: %v", listener.Addr())
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}

func main() {
	app := &cli.App{
		Name:     "devproxy",
		Compiled: time.Now(),
		Version:  fmt.Sprintf("%s (%s)", Version, Rev),
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "romnn",
				Email: "contact@romnn.com",
			},
		},
		Usage: "todo",
		Commands: []*cli.Command{
			&cli.Command{
				Name:        "start",
				Aliases:     []string{"run"},
				Usage:       "start the proxy",
				Description: "start the proxy",
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:    "port",
						Value:   8080,
						Aliases: []string{"proxy-port"},
						EnvVars: []string{"PORT", "PROXY_PORT"},
						Usage:   "proxy port",
					},
				},
				Action: run,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
