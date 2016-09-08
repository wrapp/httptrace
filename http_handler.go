package httptrace

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

// This middleware is defined for convenience to avoid the long chain of
// middleware invocations on each endpoint. It wraps a http.HandlerFunc
// adding both, request-tracing and profiling capabilities.
// This function can be used to enhance single endpoints in a http-server
// application.
func TracingHandlerFunc(h http.HandlerFunc) http.HandlerFunc {
	return TracingMiddleware(LoggingMiddleWare(h))
}

// This middleware wraps a http.Handler adding request-tracing and
// profiling capabilities.
// This function can be used to enhance a http router adding extra
// capabilities to all its enpoints.
func Trace(handler http.Handler) http.Handler {
	return TracingHandlerFunc(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				handler.ServeHTTP(w, r)
			},
		),
	)
}

// ListenAndServe wraps the http.ListenAndServe function adding request-
// tracing and profiling capabilities to the provided http.Handler.
// This function can be used as a replacement for http.ListenAndServe.
// When using this function, neither of the middlewares defined above are
// necessary.
func ListenAndServe(addr string, handler http.Handler) error {
	log.Info(fmt.Sprintf("Starting service on %s", addr))
	return http.ListenAndServe(addr, Trace(handler))
}
