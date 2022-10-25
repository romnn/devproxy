package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

type (
	// key for the request metadata
	requestMetadataKey struct{}

	// context for holding request and response details
	requestMetadata struct {
		status int
		size   int
		method string
		url    *url.URL
	}

	// key for the formatter context
	fmtProxyTargetKey struct{}

	// context for formatting the proxy target
	fmtProxyTarget struct {
		*proxyTarget
		color uint
		pad   int
	}

	// custom formatter
	proxyFormatter struct {
		log.TextFormatter
	}
)

var ansi16ColorPalette = []uint{31, 32, 33, 34, 35, 36, 37}

func (target *fmtProxyTarget) URL() string {
	return pad(target.proxyTarget.url.String(), target.pad)
}

func (f *proxyFormatter) Format(entry *log.Entry) ([]byte, error) {
	var levelColor uint
	switch entry.Level {
	case log.DebugLevel, log.TraceLevel:
		levelColor = 30
	case log.WarnLevel:
		levelColor = 33
	case log.ErrorLevel, log.FatalLevel, log.PanicLevel:
		levelColor = 31
	case log.InfoLevel:
		levelColor = 36
	default:
		levelColor = 37
	}

	var target fmtProxyTarget
	if entry.Context != nil {
		key := fmtProxyTargetKey{}
		if t, ok := entry.Context.Value(key).(fmtProxyTarget); ok {
			target = t
		}
	}
	ts := entry.Time.Format(f.TimestampFormat)
	level := strings.ToUpper(entry.Level.String())

	var buf bytes.Buffer

	// log level and timestamp first
	fmt.Fprintf(&buf, "\x1b[%d;1m%s[%s]\x1b[0m", levelColor, level, ts)
	fmt.Fprintf(&buf, " ")

	// log message
	fmt.Fprintf(&buf, "\x1b[%d;1m%-60s\x1b[0m", target.color, entry.Message)

	// log structured fields
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	if !f.DisableSorting {
		sort.Strings(keys)
	}
	for _, k := range keys {
		v := entry.Data[k]
		fmt.Fprintf(&buf, " \x1b[%d;1m%s\x1b[0m=%v", target.color, k, v)
	}
	fmt.Fprintf(&buf, "\n")

	return buf.Bytes(), nil
}

// WithLogging wraps a http.Handler for a proxy target to log requests
func WithLogging(target fmtProxyTarget, h http.Handler) http.Handler {
	loggingFn := func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		// inject custom ResponseWriter
		lrw := loggingResponseWriter{
			ResponseWriter: rw,
			responseData:   responseData,
		}
		h.ServeHTTP(&lrw, req)

		duration := time.Since(start)

		// add context with metadata for logging
		ctx := context.WithValue(
			context.Background(),
			fmtProxyTargetKey{},
			target,
		)
		requestCtx := context.WithValue(
			ctx,
			requestMetadataKey{},
			requestMetadata{
				status: responseData.status,
				size:   responseData.size,
				method: req.Method,
				url:    req.URL,
			},
		)
		msg := fmt.Sprintf("%s %s @ %s", req.Method, target.URL(), req.URL)
		log.WithContext(requestCtx).WithFields(log.Fields{
			"status":   responseData.status,
			"duration": pad(roundDuration(duration).String(), 8),
			"size":     pad(humanize.Bytes(uint64(responseData.size)), 6),
		}).Info(msg)
	}
	return http.HandlerFunc(loggingFn)
}
