package httptrace

import (
	"net/http"
	"golang.org/x/net/context"
)

type ContextHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)