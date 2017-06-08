package httptrace

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"encoding/json"

	"fmt"

	"context"

	gorillactx "github.com/gorilla/context"
	"github.com/m4rw3r/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
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
		func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Context().Value(ctxRequestIDKey)
			s.NotEmpty(requestID)
			_, ok := requestID.(string)
			s.True(ok)
		})(resp, req)
	s.NotEqual("", resp.Header().Get(headerRequestIDKey))
}

func (s *TracingSuite) TestMiddlewarePreservesRequestIDIfPresent() {
	req, _ := http.NewRequest("GET", "/endpoint", bytes.NewReader(nil))
	uuidValue, _ := uuid.V4()
	req.Header.Set(headerRequestIDKey, uuidValue.String())
	resp := httptest.NewRecorder()
	TracingMiddleware(
		func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Context().Value(ctxRequestIDKey)
			s.NotEmpty(requestID)
			requestIDStr, ok := requestID.(string)
			s.True(ok)
			s.Equal(uuidValue.String(), requestIDStr)
		})(resp, req)
	s.Equal(uuidValue.String(), resp.Header().Get(headerRequestIDKey))
}

func (s *TracingSuite) TestParameterLoggingMiddlewareDebug() {
	SetDebug(true)
	defer SetDebug(false)
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	req, _ := http.NewRequest("GET", "/endpoint", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	LoggingMiddleWare(
		func(w http.ResponseWriter, r *http.Request) {
			AddToLogging(r.Context(), "some-key", "some-value")
			// add non-logging related stuff to gorilla context
			gorillactx.Set(r, "unrelated-stuff", 42)
			w.WriteHeader(http.StatusOK)
		})(resp, req)
	fmt.Println(buf)
	log.SetOutput(os.Stdout)
	res := map[string]interface{}{}
	err := json.Unmarshal(buf.Bytes()[5:len(buf.Bytes())-1], &res)
	s.NoError(err)
	value, found := res["some-key"]
	s.True(found)
	s.Equal("some-value", value.(string))
	_, found = res["unrelated-stuff"]
	s.False(found)
}

func (s *TracingSuite) TestParameterLoggingMiddlewareNoLoggedValues() {
	req, _ := http.NewRequest("GET", "/endpoint", bytes.NewReader(nil))
	resp := httptest.NewRecorder()
	LoggingMiddleWare(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			values := extractAllLoggingValues(r.Context())
			s.Equal(map[loggingKeyType]interface{}(nil), values)
		})(resp, req)
}

func (s *TracingSuite) TestClientSetsHeaders() {
	SetGlobalUserAgent("TestUserAgent")
	uuidValue, _ := uuid.V4()
	ctx := context.WithValue(context.Background(), ctxRequestIDKey, uuidValue.String())
	handler := func(w http.ResponseWriter, r *http.Request) {
		s.Equal("TestUserAgent", r.Header.Get("User-Agent"))
		s.Equal(uuidValue.String(), r.Header.Get(headerRequestIDKey))
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handler))
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := Get(ctx, server.URL) // httptrace client sets headers in the outgoing request
	s.NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *TracingSuite) TestRecover() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("Oh, no!")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoint", handler)
	server := httptest.NewServer(Recover(mux))
	defer server.Close()

	buf := new(bytes.Buffer)

	log.SetOutput(buf)
	http.Get(server.URL + "/endpoint")
	log.SetOutput(os.Stdout)

	var output map[string]interface{}
	json.Unmarshal(buf.Bytes()[6:], &output)
	s.Contains(output, "panic")
	s.Contains(output, "traceback")
	s.Contains(output, "endpoint")
	s.Equal("Unhandled panic: Oh, no!", output["panic"].(string))
}

func (s *TracingSuite) TestTrace() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		AddToLogging(r.Context(), "MyKey", "MyValue")
		panic("Oh, no!")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoint", handler)
	server := httptest.NewServer(Trace(mux))
	defer server.Close()

	buf := new(bytes.Buffer)

	log.SetOutput(buf)
	req, _ := http.NewRequest("GET", server.URL+"/endpoint", nil)
	req.Header.Set(headerRequestIDKey, "Request-ID-Value")
	http.DefaultClient.Do(req)
	log.SetOutput(os.Stdout)

	// Should not log anything
	s.Equal(0, len(buf.Bytes()))
}
