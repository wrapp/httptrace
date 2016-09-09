package httptrace

import (
	"io"
	"net/http"

	"context"

	"github.com/m4rw3r/uuid"
)

type ctxRequestIDKeyType string // This avoids key collisions with clients

const (
	ctxRequestIDKey    ctxRequestIDKeyType = "request-id"
	headerRequestIDKey string              = "X-Request-ID"
)

var userAgent string

func SetGlobalUserAgent(ua string) {
	userAgent = ua
}

func TracingMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requestWithContext := r.WithContext(context.WithValue(r.Context(), ctxRequestIDKey, getRequestID(r)))
			defer func() {
				if requestID := requestWithContext.Context().Value(ctxRequestIDKey); requestID != nil {
					if requestIDStr, ok := requestID.(string); ok {
						w.Header().Set(headerRequestIDKey, requestIDStr)
					}
				}
			}()

			h(w, requestWithContext)

		},
	)
}

func getRequestID(req *http.Request) string {
	requestID := req.Header.Get(headerRequestIDKey)
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
	if requestID := ctx.Value(ctxRequestIDKey); requestID != nil {
		if requestIDStr, ok := requestID.(string); ok {
			req.Header.Set(headerRequestIDKey, requestIDStr)
		}
	}
	return c.HTTPClient.Do(req)
}

func (c *TracingHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

func (c *TracingHTTPClient) Post(ctx context.Context, url string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", bodyType)
	return c.Do(ctx, req)
}

func (c *TracingHTTPClient) Put(ctx context.Context, url string, bodyType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPut, url, body)
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
	return DefaultTracingHTTPClient.Put(ctx, url, bodyType, body)
}

func Put(ctx context.Context, url string, bodyType string, body io.Reader) (*http.Response, error) {
	return DefaultTracingHTTPClient.Put(ctx, url, bodyType, body)
}
