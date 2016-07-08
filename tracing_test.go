package httptrace

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/m4rw3r/uuid"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type TracingSuite struct {
	suite.Suite
}

func (s *TracingSuite) SetupTest() {
}

func (s *TracingSuite) SetupSuite() {
}

func TestTracingSuite(t *testing.T) {
	suite.Run(t, new(TracingSuite))
}

func (s *TracingSuite) TestMiddlewareCreatesNewRequestIDIfNotPresent() {
	req, _ := http.NewRequest("GET", "/endpoint", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	TracingMiddleware(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			requestID := ctx.Value(CtxRequestIDKey)
			s.NotEmpty(requestID)
			_, ok := requestID.(string)
			s.True(ok)
		})(resp, req)
	s.NotEqual("", resp.Header().Get(HeaderRequestIDKey))
}

func (s *TracingSuite) TestMiddlewarePreservesRequestIDIfPresent() {
	req, _ := http.NewRequest("GET", "/endpoint", bytes.NewReader(nil))
	uuidValue, _ := uuid.V4()
	req.Header.Set(HeaderRequestIDKey, uuidValue.String())
	resp := httptest.NewRecorder()
	TracingMiddleware(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			requestID := ctx.Value(CtxRequestIDKey)
			s.NotEmpty(requestID)
			requestIDStr, ok := requestID.(string)
			s.True(ok)
			s.Equal(uuidValue.String(), requestIDStr)
		})(resp, req)
	s.Equal(uuidValue.String(), resp.Header().Get(HeaderRequestIDKey))
}

func (s *TracingSuite) TestParameterLoggingMiddleware() {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	defer log.SetOutput(os.Stdout)
	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/endpoint", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	resp.WriteHeader(http.StatusOK)
	ParameterLoggingMiddleWare(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			AddToLogging(ctx, "some-key", "some-value")
		})(ctx, resp, req)
	s.True(strings.Contains(buf.String(), "some-key=some-value"))
}

func (s *TracingSuite) TestClientSetsHeaders() {
	SetGlobalUserAgent("TestUserAgent")
	uuidValue, _ := uuid.V4()
	ctx := context.WithValue(context.Background(), CtxRequestIDKey, uuidValue.String())
	handler := func(w http.ResponseWriter, r *http.Request) {
		s.Equal("TestUserAgent", r.Header.Get("User-Agent"))
		s.Equal(uuidValue.String(), r.Header.Get(HeaderRequestIDKey))
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handler))
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := Get(ctx, server.URL) // httptrace client sets headers in the outgoing request
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
}
