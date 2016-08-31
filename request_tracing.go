package httptrace

import (
	"io"
	"net/http"

	"context"

	"github.com/m4rw3r/uuid"
)

const (
	CtxRequestIDKey    string = "request-id"
	HeaderRequestIDKey string = "X-Request-ID"
)

var userAgent string

func SetGlobalUserAgent(ua string) {
	userAgent = ua
}

func TracingMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requestWithContext := r.WithContext(context.WithValue(r.Context(), CtxRequestIDKey, getRequestID(r)))
			defer func() {
				if requestID := requestWithContext.Context().Value(CtxRequestIDKey); requestID != nil {
					if requestIDStr, ok := requestID.(string); ok {
						w.Header().Set(HeaderRequestIDKey, requestIDStr)
					}
				}
			}()

			h(w, requestWithContext)

		},
	)
}

func getRequestID(req *http.Request) string {
	requestID := req.Header.Get(HeaderRequestIDKey)
	if requestID == "" {
		uuidValue, _ := uuid.V4()
		requestID = uuidValue.String()
	}
	return requestID
}

// Any HTTP client implementing this interface can be used by the tracer HTTP client
type HTTPClient interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

type TracingHTTPClient struct {
	HTTPClient
}

func (c *TracingHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
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

func (c *TracingHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

func (c *TracingHTTPClient) Post(ctx context.Context, url string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(ctx, req)
}

var DefaultTracingHTTPClient = TracingHTTPClient{http.DefaultClient}

func Get(ctx context.Context, url string) (*http.Response, error) {
	return DefaultTracingHTTPClient.Get(ctx, url)
}

func Post(ctx context.Context, url string, bodyType string, body io.Reader) (*http.Response, error) {
	return DefaultTracingHTTPClient.Post(ctx, url, bodyType, body)
}
