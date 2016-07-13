package httptrace

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"encoding/json"

	log "github.com/Sirupsen/logrus"
	"github.com/m4rw3r/uuid"
	"github.com/stretchr/testify/suite"
	_ "github.com/wrapp/wrapplog"
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
	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/endpoint", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	ParameterLoggingMiddleWare(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			AddToLogging(ctx, "some-key", "some-value")
			w.WriteHeader(http.StatusOK)
		})(ctx, resp, req)
	log.SetOutput(os.Stdout)
	res := map[string]interface{}{}
	err := json.Unmarshal(buf.Bytes()[5:len(buf.Bytes())-1], &res)
	s.NoError(err)
	value, found := res["some-key"]
	s.True(found)
	s.Equal("some-value", value.(string))
}

func (s *TracingSuite) TestParameterLoggingMiddlewareNoLoggedValues() {
	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/endpoint", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	ParameterLoggingMiddleWare(
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			values, found := GetAllLoggingValues(ctx)
			s.False(found)
			s.Equal(map[interface{}]interface{}(nil), values)
		})(ctx, resp, req)
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
