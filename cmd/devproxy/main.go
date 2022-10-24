package main

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Version is set during build
var Version = ""

// Rev is set during build
var Rev = ""

var ansi16ColorPalette = []uint{31, 32, 33, 34, 35, 36, 37}

type colorKey struct{}

func stringToUint64(s string) uint64 {
	hashed := sha1.Sum([]byte(s))
	return binary.BigEndian.Uint64(hashed[:])
}

func stringToColorCode(s string, codes []uint) uint {
	i := stringToUint64(s)
	idx := i % uint64(len(codes))
	return codes[idx]
}

func run(cliCtx *cli.Context) error {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	addr := fmt.Sprintf(":%d", cliCtx.Uint("port"))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	proxyRouter := http.NewServeMux()
	args := append([]string{cliCtx.Args().First()}, cliCtx.Args().Tail()...)
	for _, target := range args {
		split := strings.Split(target, "@")

		// split must contain a route path component and the url to be proxied
		if len(split) != 2 {
			log.Warnf("invalid target %q ignored", target)
			continue
		}

		path := split[0]
		targetUrl, err := url.Parse(split[1])
		if err != nil {
			log.Warnf("invalid url %q ignored", split[1])
		}

		addCORS := func(res *http.Response) error {
			res.Header.Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE")
			res.Header.Set("Access-Control-Allow-Credentials", "true")
			res.Header.Set("Access-Control-Allow-Origin", "*")
			return nil
		}

		reverse := httputil.NewSingleHostReverseProxy(targetUrl)
		reverse.ModifyResponse = addCORS

		colorCode := stringToColorCode(targetUrl.String(), ansi16ColorPalette)
		ctx := context.WithValue(context.Background(), colorKey{}, colorCode)

		proxyRouter.Handle(path, WithLogging(ctx, http.StripPrefix(path, reverse)))
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
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
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
