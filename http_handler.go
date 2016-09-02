package httptrace

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func TracingHandlerFunc(h http.HandlerFunc) http.HandlerFunc {
	return TracingMiddleware(LoggingMiddleWare(h))
}

func ListenAndServe(addr string, handler http.Handler) error {
	log.Info(fmt.Sprintf("Starting service on %s", addr))
	return http.ListenAndServe(addr, Trace(handler))
}

func Trace(handler http.Handler) http.Handler {
	return TracingHandlerFunc(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				handler.ServeHTTP(w, r)
			},
		),
	)
}
