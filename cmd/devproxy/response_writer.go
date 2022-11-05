package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type (
	// context for holding response details
	responseData struct {
		status int
		size   int
	}

	// custom http.ResponseWriter implementation
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	// write response using original http.ResponseWriter
	size, err := w.ResponseWriter.Write(b)
	// capture size
	w.responseData.size += size
	return size, err
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	// write status code using original http.ResponseWriter
	w.ResponseWriter.WriteHeader(statusCode)
	// capture status code
	w.responseData.status = statusCode
}

func (w *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijack, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%v does not support hijack", w.ResponseWriter)
	}
	return hijack.Hijack()
}
