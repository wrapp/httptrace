package httptrace

import (
	"fmt"
	"net/http"
	"time"

	"runtime"

	"context"

	log "github.com/Sirupsen/logrus"
	gorillactx "github.com/gorilla/context"
)

// If debug is set to false, request-logging will only occur in case of error
var debug bool = false

func SetDebug(d bool) {
	debug = d
}

// non-exported key types
type requestKeyType string // This avoids collisions with other gorilla context keys
type loggingKeyType string // This allows discriminating httptrace-logging keys from others

const requestKey requestKeyType = "httptrace"

// LoggingMiddleware adds logging and profiling capabilities to http.HandlerFunc.
// When used, a number of useful request-local parameters will be logged once the request
// finishes being processed. Any panic or errors will be captured and logged as well, along with
// any other key-value pairs that may have been "added to logging" while serving a specific
// request.
// To add key-value pairs so that they are logged once the request has been processed, use the
// function AddToLogging retrieving the context.Context from the request
func LoggingMiddleWare(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fields := log.Fields{
				"endpoint": r.URL.Path,
				"method":   r.Method,
			}
			lw := &statusResponseWriter{w: w}
			ctx := context.WithValue(r.Context(), requestKey, r)
			requestWithContext := r.WithContext(ctx)
			defer func(begin time.Time) {
				fields["took"] = time.Since(begin).String()
				if requestID := ctx.Value(ctxRequestIDKey); requestID != nil {
					if requestIDStr, ok := requestID.(string); ok {
						fields[string(ctxRequestIDKey)] = requestIDStr
					}
				}
				for k, v := range extractAllLoggingValues(ctx) {
					fields[string(k)] = v
				}
				gorillactx.Clear(r)
				if e := recover(); e != nil {
					msg, stack := extractPanic(e)
					fields["panic"] = msg
					fields["traceback"] = stack
					http.Error(lw, fmt.Sprintf("%s \n%s", msg, stack), http.StatusInternalServerError)
				}
				fields["status_text"] = http.StatusText(lw.status)
				switch {
				case lw.status < 300:
					if debug {
						log.WithFields(fields).Info("request successful")
					}
				case lw.status >= 300 && lw.status < 400:
					log.WithFields(fields).Warn("additional action required")
				case lw.status >= 400 && lw.status < 500:
					log.WithFields(fields).Warn("request failed")
				default:
					log.WithFields(fields).Error("request failed")
				}
			}(time.Now())

			h(lw, requestWithContext)

		},
	)
}

// This function is used to add key-value pairs to logging
func AddToLogging(ctx context.Context, key string, value interface{}) {
	if iReq := ctx.Value(requestKey); iReq != nil {
		if req, ok := iReq.(*http.Request); ok {
			gorillactx.Set(req, loggingKeyType(key), value)
		}
	}
}

func GetLoggingValue(ctx context.Context, key string) (interface{}, bool) {
	iReq := ctx.Value(requestKey)
	if iReq == nil {
		return nil, false
	}
	req, ok := iReq.(*http.Request)
	if !ok {
		return nil, false
	}
	return gorillactx.GetOk(req, loggingKeyType(key))
}

func extractAllLoggingValues(ctx context.Context) map[loggingKeyType]interface{} {
	iReq := ctx.Value(requestKey)
	if iReq == nil {
		return nil
	}

	req, ok := iReq.(*http.Request)
	if !ok {
		return nil
	}

	reqData, found := gorillactx.GetAllOk(req)
	if !found {
		return nil
	}

	ret := make(map[loggingKeyType]interface{})
	for k, v := range reqData {
		if loggingKey, ok := k.(loggingKeyType); ok {
			ret[loggingKey] = v
			gorillactx.Delete(req, loggingKey)
		}
	}
	return ret
}

type statusResponseWriter struct {
	w      http.ResponseWriter
	status int
}

func (l *statusResponseWriter) Flush() {
	if wf, ok := l.w.(http.Flusher); ok {
		wf.Flush()
	}
}

func (l *statusResponseWriter) Header() http.Header { return l.w.Header() }

func (l *statusResponseWriter) Write(b []byte) (int, error) {
	if l.status == 0 {
		// The status will be StatusOK if WriteHeader has not been called yet
		l.status = http.StatusOK
	}
	size, err := l.w.Write(b)
	return size, err
}

func (l *statusResponseWriter) WriteHeader(status int) {
	l.w.WriteHeader(status)
	l.status = status
}

func extractPanic(p interface{}) (msg string, stack string) {
	msg = "Unhandled panic: "
	var buf []byte
	runtime.Stack(buf, true)
	stack = string(buf)
	switch p := p.(type) {
	case string:
		msg += p
	case error:
		msg += p.Error()
	case fmt.Stringer:
		msg += p.String()
	default:
		msg += fmt.Sprintf("%T: %+v\n", p, p)
	}
	return
}
