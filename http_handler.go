package httptrace

import "net/http"

func NewHandlerFunc(h ContextHandlerFunc) http.HandlerFunc {
	return TracingMiddleware(ParameterLoggingMiddleWare(h))
}
