package httptrace

import (
	"io"
	"net/http"

	"github.com/m4rw3r/uuid"
	"golang.org/x/net/context"
)

const (
	CtxRequestIDKey    string = "request-id"
	HeaderRequestIDKey string = "X-Request-ID"
)

var userAgent string

func SetGlobalUserAgent(ua string) {
	userAgent = ua
}

func TracingMiddleware(h ContextHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := newContextWithRequestID(r)
		defer func() {
			if requestID := ctx.Value(CtxRequestIDKey); requestID != nil {
				if requestIDStr, ok := requestID.(string); ok {
					w.Header().Set(HeaderRequestIDKey, requestIDStr)
				}
			}
		}()
		h(ctx, w, r)
	}
}

func newContextWithRequestID(req *http.Request) context.Context {
	requestID := req.Header.Get(HeaderRequestIDKey)
	if requestID == "" {
		uuidValue, _ := uuid.V4()
		requestID = uuidValue.String()
	}
	return context.WithValue(context.Background(), CtxRequestIDKey, requestID)
}

// Any HTTP client implementing this interface can be used by the tracer HTTP client
type HTTPClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

type TracingHTTPClient struct {
	HTTPClient
}

func (c *TracingHTTPClient) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if requestID := ctx.Value(CtxRequestIDKey); requestID != nil {
		if requestIDStr, ok := requestID.(string); ok {
			req.Header.Set(HeaderRequestIDKey, requestIDStr)
		}
	}
	return c.HTTPClient.Do(req)
}

func (c *TracingHTTPClient) Get(ctx context.Context, url string) (resp *http.Response, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

func (c *TracingHTTPClient) Post(ctx context.Context, url string, bodyType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(ctx, req)
}

var DefaultTracingHTTPClient = TracingHTTPClient{http.DefaultClient}

func Get(ctx context.Context, url string) (resp *http.Response, err error) {
	return DefaultTracingHTTPClient.Get(ctx, url)
}

func Post(ctx context.Context, url string, bodyType string, body io.Reader) (resp *http.Response, err error) {
	return DefaultTracingHTTPClient.Post(ctx, url, bodyType, body)
}
