package httplog

import (
	"net/http"
)

func WithLogging(next http.Handler, l Logger) http.Handler {
	return &loggingHandler{next: next, logger: l}
}

type loggingHandler struct {
	logger Logger
	next   http.Handler
}

func (h loggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.SetRequestInfo(r)
	lrw := &loggingResponseWriter{ResponseWriter: w}
	h.next.ServeHTTP(lrw, r)
	h.logger.SetStatus(lrw.Status)
	defer h.logger.Log()
}

type loggingResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (lrw *loggingResponseWriter) WriteHeader(status int) {
	lrw.Status = status
	lrw.ResponseWriter.WriteHeader(status)
}
