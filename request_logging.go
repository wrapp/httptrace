package httptrace

import (
	"fmt"
	"net/http"
	"time"

	"reflect"
	"runtime"

	log "github.com/Sirupsen/logrus"
	gorillactx "github.com/gorilla/context"
	"golang.org/x/net/context"
)

func LoggingMiddleWare(h ContextHandlerFunc) ContextHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		fields := log.Fields{
			"endpoint": r.URL.Path,
			"method":   r.Method,
		}
		lw := &statusResponseWriter{w: w}
		defer func(begin time.Time) {
			fields["took"] = time.Since(begin).String()
			if requestID := ctx.Value(CtxRequestIDKey); requestID != nil {
				if requestIDStr, ok := requestID.(string); ok {
					fields[CtxRequestIDKey] = requestIDStr
				}
			}
			if e := recover(); e != nil {
				msg, stack := extractPanic(e)
				fields["error"] = msg
				fields["traceback"] = stack
				http.Error(w, fmt.Sprintf("%s \n%s", msg, stack), http.StatusInternalServerError)
			} else if lw.status != http.StatusOK {
				fields["error"] = http.StatusText(lw.status)
			}
			if _, found := fields["error"]; found {
				log.WithFields(fields).Error("request failed")
			} else {
				log.WithFields(fields).Info("request succesful")
			}
		}(time.Now())

		h(ctx, lw, r)

	}
}

const parameterLoggingKey = "httptrace"

func ParameterLoggingMiddleWare(h ContextHandlerFunc) ContextHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		fields := log.Fields{
			"endpoint": r.URL.Path,
			"method":   r.Method,
		}
		lw := &statusResponseWriter{w: w}
		gorillactx.Clear(r)
		ctx = context.WithValue(ctx, parameterLoggingKey, r)
		defer func(begin time.Time) {
			fields["took"] = time.Since(begin).String()
			if requestID := ctx.Value(CtxRequestIDKey); requestID != nil {
				if requestIDStr, ok := requestID.(string); ok {
					fields[CtxRequestIDKey] = requestIDStr
				}
			}
			if loggingValues, found := GetAllLoggingValues(ctx); found {
				for k, v := range loggingValues {
					fields[k.(string)] = v
				}
			}
			gorillactx.Clear(r)
			if e := recover(); e != nil {
				msg, stack := extractPanic(e)
				fields["error"] = msg
				fields["traceback"] = stack
				http.Error(w, fmt.Sprintf("%s \n%s", msg, stack), http.StatusInternalServerError)
			} else if lw.status != http.StatusOK {
				fields["error"] = http.StatusText(lw.status)
			}
			if _, found := fields["error"]; found {
				log.WithFields(fields).Error("request failed")
			} else {
				log.WithFields(fields).Info("request succesful")
			}
		}(time.Now())

		h(ctx, lw, r)

	}
}

func AddToLogging(ctx context.Context, key string, value interface{}) {
	req := ctx.Value(parameterLoggingKey).(*http.Request)
	gorillactx.Set(req, key, value)
}

func GetLoggingValue(ctx context.Context, key string) (interface{}, bool) {
	return gorillactx.GetOk(ctx.Value(parameterLoggingKey).(*http.Request), key)
}

func GetAllLoggingValues(ctx context.Context) (map[interface{}]interface{}, bool) {
	if ret, found := gorillactx.GetAllOk(ctx.Value(parameterLoggingKey).(*http.Request)); found {
		return ret, found
	}
	return nil, false
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
	msg = "unhandled panic: "
	var buf [4096]byte
	runtime.Stack(buf[:], true)
	stack = string(buf[:runtime.Stack(buf[:], false)])
	switch v := p.(type) {
	case string:
		msg += v
	default:
		msg += reflect.TypeOf(v).String()
	}
	return
}
