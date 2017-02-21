package httptrace

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"time"

	"context"

	log "github.com/Sirupsen/logrus"
)

// TracingHandlerFunc is middleware defined for convenience to avoid the long chain of
// middleware invocations on each endpoint. It wraps a http.HandlerFunc
// adding both, request-tracing and profiling capabilities.
// This function can be used to enhance single endpoints in a http-server
// application.
func TracingHandlerFunc(h http.HandlerFunc) http.HandlerFunc {
	return TracingMiddleware(LoggingMiddleWare(h))
}

// Trace is middleware that wraps a http.Handler adding request-tracing and
// profiling capabilities.
// This function can be used to enhance a http router adding extra
// capabilities to all its endpoints.
func Trace(handler http.Handler) http.Handler {
	if handler == nil {
		handler = http.DefaultServeMux
	}
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
// When using this function, neither of the middleware defined above are
// necessary.

// In addition to tracing capabilities this function also provides graceful
// shutdown of the server. When a SIGTERM or SIGINT signal is received by the
// server it will stop accepting new connections but it will keep it running
// until all the existing connections have been closed. It will kill the server
// after a default timeout of 25 seconds if there are still any pending
// connections.

func ListenAndServe(addr string, handler http.Handler) error {
	log.Info(fmt.Sprintf("Starting service on %s", addr))

	// channel for SIGINT signals
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	srv := &http.Server{
		Addr:    addr,
		Handler: Trace(handler),
	}

	var err error
	go func() {
		if err = srv.ListenAndServe(); err != nil {
			log.Error("Server errror: ", err)
			close(stopChan)
		}
	}()

	<-stopChan // wait for SIGINT
	log.Info("Shutting down server gracefully...")

	ctx, _ := context.WithTimeout(context.Background(), 25*time.Second)
	srv.Shutdown(ctx)
	log.Info("Server gracefully stopped.")
	return err
}
