package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

type (
	// struct for holding response details
	responseData struct {
		status int
		size   int
	}

	// our http.ResponseWriter implementation
	loggingResponseWriter struct {
		// compose original http.ResponseWriter
		http.ResponseWriter
		responseData *responseData
	}

	// our formatter
	myFormatter struct {
		log.TextFormatter
		// colorMap map[string]log.TextFormatter
	}
)

func (f *myFormatter) Format(entry *log.Entry) ([]byte, error) {
	color := uint(37) // white
	if entry.Context != nil {
		if c, ok := entry.Context.Value(colorKey{}).(uint); ok {
			color = c
		}
	}
	ts := entry.Time.Format(f.TimestampFormat)
	level := strings.ToUpper(entry.Level.String())
	msg := fmt.Sprintf("\x1b[%d;1m%s[%s] %s\x1b[0m\n", color, level, ts, entry.Message)
	return []byte(msg), nil
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// write response using original http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	// capture size
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// write status code using original http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	// capture status code
	r.responseData.status = statusCode
}

func WithLogging(ctx context.Context, h http.Handler) http.Handler {
	loggingFn := func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lrw := loggingResponseWriter{
			ResponseWriter: rw,
			responseData:   responseData,
		}
		// inject our implementation of http.ResponseWriter
		h.ServeHTTP(&lrw, req)

		duration := time.Since(start)

		log.WithContext(ctx).WithFields(log.Fields{
			// "timestamp": time.Now().Format("2006-01-02 15:04:05"),
			// "uri":       req.RequestURI,
			// "method":    req.Method,
			"status":   responseData.status,
			"duration": duration,
			"size":     humanize.Bytes(uint64(responseData.size)),
		}).Info(fmt.Sprintf("%s %s", req.Method, req.URL))
	}
	return http.HandlerFunc(loggingFn)
}
