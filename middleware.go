package httplog

import (
	"net/http"
)

// WithLogging provides HTTP logging capabilities to an http.Handler.
// It is used as http.Handler middleware. Requires the http.Handler and a
// pre-configured Logger.
// Returns a new handler that wraps the supplied handler and provides logging
// functionality.
// Example:
//
//   type customHandler {}
//
//   func (h *customHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//     w.WriteHeader(200)
// 	   // ...your handler logic goes here...
//   }
//
//   func main() {
// 	   // Configure a logger
// 	   l := httplog.New(os.Stdout)
//
// 	   // And use the middleware
// 	   http.Handle("/logs", httplog.WithLogging(&customHandler{}, l))
// 	   http.ListenAndServe(":8080", nil)
//   }
//
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
