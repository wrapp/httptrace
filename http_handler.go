package httptrace

import (
	"fmt"
	"net/http"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/tylerb/graceful"
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

// In addition to tracing capabilities this function also provides graceful
// shutdown of the server. When a SIGTERM signal is received by the server
// it will stop accepting new connections but it will keep it running until
// all the existing connections have been closed. It will kill the server
// after a default timeout of 25 seconds if there are still any pending
// connections.

func ListenAndServe(addr string, handler http.Handler) error {
	log.Info(fmt.Sprintf("Starting service on %s", addr))
	srv := &graceful.Server{
		Timeout: 25 * time.Second,
		LogFunc: func(format string, args ...interface{}) {
			log.Info(format, args)
		},
		Server: &http.Server{
			Addr:    addr,
			Handler: Trace(handler),
		},
	}
	return srv.ListenAndServe()
}
