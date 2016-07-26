package httptrace

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func ListenAndServe(addr string, handler http.Handler) error {
	log.Info(fmt.Sprintf("Starting service on %s", addr))
	return http.ListenAndServe(addr, Recover(handler))
}

// Recover is a middleware that recovers a handler from an error and logs the traceback
func Recover(handler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					msg, stack := extractPanic(rec)
					log.WithFields(log.Fields{
						"endpoint":  r.RequestURI,
						"traceback": stack,
						"panic":     msg,
					}).Error("request failed")
					http.Error(w, fmt.Sprintf("%s \n%s", msg, stack), http.StatusInternalServerError)
				}
			}()
			handler.ServeHTTP(w, r)
		})
}
