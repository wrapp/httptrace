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

var debug bool = false

func SetDebug(d bool) {
	debug = d
}

// unexported key types
type requestKeyType string
type loggingKeyType string

const requestKey requestKeyType = "httptrace"

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
				if requestID := ctx.Value(CtxRequestIDKey); requestID != nil {
					if requestIDStr, ok := requestID.(string); ok {
						fields[CtxRequestIDKey] = requestIDStr
					}
				}
				for k, v := range getAllLoggingValues(ctx) {
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

func AddToLogging(ctx context.Context, key string, value interface{}) {
	req := ctx.Value(requestKey).(*http.Request)
	gorillactx.Set(req, loggingKeyType(key), value)
}

func GetLoggingValue(ctx context.Context, key string) (interface{}, bool) {
	return gorillactx.GetOk(ctx.Value(requestKey).(*http.Request), loggingKeyType(key))
}

func getAllLoggingValues(ctx context.Context) map[loggingKeyType]interface{} {
	ret := make(map[loggingKeyType]interface{})
	if reqData, found := gorillactx.GetAllOk(ctx.Value(requestKey).(*http.Request)); found {
		for k, v := range reqData {
			if loggingKey, ok := k.(loggingKeyType); ok {
				ret[loggingKey] = v
			}
		}
		return ret
	}
	return nil
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
	var buf [4096]byte
	runtime.Stack(buf[:], true)
	stack = string(buf[:runtime.Stack(buf[:], false)])
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
