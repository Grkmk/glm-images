package handlers

import (
	"compress/gzip"
	"net/http"
	"strings"
)

type GzipHandler struct{}

func GZipResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {
			wrappedResponse := NewWrappedResponseWriter(responseWriter)
			wrappedResponse.Header().Set("Content-Encoding", "gzip")

			next.ServeHTTP(wrappedResponse.responseWriter, request) // the rw here might be wrong (ep.12)
			defer wrappedResponse.Flush()

			return
		}

		next.ServeHTTP(responseWriter, request)
	})
}

type WrappedResponseWriter struct {
	responseWriter http.ResponseWriter
	gzipWriter     *gzip.Writer
}

func NewWrappedResponseWriter(responseWriter http.ResponseWriter) *WrappedResponseWriter {
	gzipWriter := gzip.NewWriter(responseWriter)

	return &WrappedResponseWriter{responseWriter, gzipWriter}
}

func (wrappedResponse *WrappedResponseWriter) Header() http.Header {
	return wrappedResponse.responseWriter.Header()
}

func (wrappedResponse *WrappedResponseWriter) Write(data []byte) (int, error) {
	return wrappedResponse.gzipWriter.Write(data)
}

func (wrappedResponse *WrappedResponseWriter) WriteHandler(statusCode int) {
	wrappedResponse.responseWriter.WriteHeader(statusCode)
}

func (wrappedResponse *WrappedResponseWriter) Flush() {
	wrappedResponse.gzipWriter.Flush()
	wrappedResponse.gzipWriter.Close()
}
