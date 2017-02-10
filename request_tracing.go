package httptrace

import (
	"io"
	"net/http"

	"context"

	"github.com/m4rw3r/uuid"
	"os"
)

type ctxRequestIDKeyType string // This avoids key collisions with clients

const (
	ctxRequestIDKey    ctxRequestIDKeyType = "request-id"
	headerRequestIDKey string              = "X-Request-ID"
	headerUniqueIDKey  string              = "X-Unique-ID"
)

var userAgent string

// init reads the 'SERVICE_NAME' environment variable so that if it is not set manually, we have something meaningful
func init() {
	userAgent = os.Getenv(`SERVICE_NAME`)
}

/* We need a getter for the RequestID in this package, because context.Context.Value is also considering the type when
fetching values, and since the request-id is in a const, it is tied to this package. */
// GetRequestID gets the RequestID from a context.context
func GetRequestID(ctx context.Context) string {
	if ctx.Value(ctxRequestIDKey) == nil {
		return ""
	}
	return ctx.Value(ctxRequestIDKey).(string)
}

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
						setRequestID(w, headerRequestIDKey, requestIDStr)
						setRequestID(w, headerUniqueIDKey, requestIDStr)
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

func setRequestID(w http.ResponseWriter, headerName string, headerValue string) {
	w.Header().Set(headerName, headerValue)
}

// HTTPClient an client implementing this interface can be used by the tracer HTTP client
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
	return DefaultTracingHTTPClient.Post(ctx, url, bodyType, body)
}

func Put(ctx context.Context, url string, bodyType string, body io.Reader) (*http.Response, error) {
	return DefaultTracingHTTPClient.Put(ctx, url, bodyType, body)
}
