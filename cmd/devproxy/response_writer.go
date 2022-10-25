package main

import (
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
