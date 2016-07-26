package httptrace

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"

	log "github.com/Sirupsen/logrus"
)

// Recover is a middleware that recovers a handler from an error and logs the traceback
func Recover(handler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					var msg = "Unhandled panic: "
					var buf [4096]byte
					runtime.Stack(buf[:], true)
					stack := buf[:runtime.Stack(buf[:], false)]
					switch v := rec.(type) {
					case string:
						msg += v
					default:
						msg += reflect.TypeOf(v).String()
					}
					log.WithFields(log.Fields{
						"traceback": string(stack),
					}).Error(msg)
					http.Error(w, fmt.Sprintf("%s \n%s", msg, stack), http.StatusInternalServerError)
				}
			}()
			handler.ServeHTTP(w, r)
		})
}
