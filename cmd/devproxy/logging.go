package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
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
	}

	// key for the formatter context
	fmtProxyTargetKey struct{}

	// formatter context for the proxy target
	fmtProxyTarget struct {
		proxyTarget
		color uint
		pad   uint
	}
)

var ansi16ColorPalette = []uint{31, 32, 33, 34, 35, 36, 37}

func stringToUint64(s string) uint64 {
	hashed := sha1.Sum([]byte(s))
	return binary.BigEndian.Uint64(hashed[:])
}

func stringToColorCode(s string, codes []uint) uint {
	i := stringToUint64(s)
	idx := i % uint64(len(codes))
	return codes[idx]
}

func pad(s string, length int) string {
	if len(s) >= length {
		return s
	}
  output := strings.Repeat(" ", length-len(s)) + s
	return output
}

func (f *myFormatter) Format(entry *log.Entry) ([]byte, error) {
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
	// name := pad(target.proxyTarget.url, target.pad)
	// fmt.Fprintf(buf, "\x1b[%d;1m%s:[%s] ", name)

	// log level and timestamp first
	fmt.Fprintf(&buf, "\x1b[%d;1m%s[%s]\x1b[0m", levelColor, level, ts)
	fmt.Fprintf(&buf, " ")

	// log message
	fmt.Fprintf(&buf, "\x1b[%d;1m%s\x1b[0m", target.color, entry.Message)
	fmt.Fprintf(&buf, "\n")

	return buf.Bytes(), nil
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
