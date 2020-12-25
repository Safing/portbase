package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/safing/portbase/log"
)

var (
	// gorilla mux
	mainMux = mux.NewRouter()

	// middlewares
	middlewareHandler = &mwHandler{
		final: mainMux,
		handlers: []Middleware{
			ModuleWorker,
			LogTracer,
			RequestLogger,
			authMiddleware,
		},
	}

	// main server and lock
	server      = &http.Server{}
	handlerLock sync.RWMutex
)

// RegisterHandler registers a handler with the API endoint.
func RegisterHandler(path string, handler http.Handler) *mux.Route {
	handlerLock.Lock()
	defer handlerLock.Unlock()
	return mainMux.Handle(path, handler)
}

// RegisterHandleFunc registers a handle function with the API endoint.
func RegisterHandleFunc(path string, handleFunc func(http.ResponseWriter, *http.Request)) *mux.Route {
	handlerLock.Lock()
	defer handlerLock.Unlock()
	return mainMux.HandleFunc(path, handleFunc)
}

// RegisterMiddleware registers a middle function with the API endoint.
func RegisterMiddleware(middleware Middleware) {
	handlerLock.Lock()
	defer handlerLock.Unlock()
	middlewareHandler.handlers = append(middlewareHandler.handlers, middleware)
}

// Serve starts serving the API endpoint.
func Serve() {
	// configure server
	server.Addr = listenAddressConfig()
	server.Handler = &mainHandler{
		mux: mainMux,
	}

	// start serving
	log.Infof("api: starting to listen on %s", server.Addr)
	backoffDuration := 10 * time.Second
	for {
		// always returns an error
		err := module.RunWorker("http endpoint", func(ctx context.Context) error {
			return server.ListenAndServe()
		})
		// return on shutdown error
		if err == http.ErrServerClosed {
			return
		}
		// log error and restart
		log.Errorf("api: http endpoint failed: %s - restarting in %s", err, backoffDuration)
		time.Sleep(backoffDuration)
	}
}

type mainHandler struct {
	mux *mux.Router
}

func (mh *mainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = module.RunWorker("http request", func(_ context.Context) error {
		return mh.handle(w, r)
	})
}

func (mh *mainHandler) handle(w http.ResponseWriter, r *http.Request) error {
	// Setup context trace logging.
	ctx, tracer := log.AddTracer(r.Context())
	lrw := NewLoggingResponseWriter(w, r)
	// Add request context.
	apiRequest := &Request{
		Request: r,
	}
	ctx = context.WithValue(ctx, requestContextKey, apiRequest)
	// Add context back to request.
	r = r.WithContext(ctx)

	tracer.Tracef("api request: %s ___ %s", r.RemoteAddr, r.RequestURI)
	defer func() {
		// Log request status.
		if lrw.Status != 0 {
			// If lrw.Status is 0, the request may have been hijacked.
			tracer.Debugf("api request: %s %d %s", lrw.Request.RemoteAddr, lrw.Status, lrw.Request.RequestURI)
		}
		tracer.Submit()
	}()

	// Get handler for request.
	// Gorilla does not support handling this on our own very well.
	// See github.com/gorilla/mux.ServeHTTP for reference.
	var match mux.RouteMatch
	var handler http.Handler
	if mh.mux.Match(r, &match) {
		handler = match.Handler
		apiRequest.Route = match.Route
		apiRequest.URLVars = match.Vars
	}

	// Be sure that URLVars always is a map.
	if apiRequest.URLVars == nil {
		apiRequest.URLVars = make(map[string]string)
	}

	// Check authentication.
	token := authenticateRequest(lrw, r, handler)
	if token == nil {
		// Authenticator already replied.
		return nil
	}
	apiRequest.AuthToken = &AuthToken{
		Read:  token.Read,
		Write: token.Write,
	}

	// Handle request.
	switch {
	case handler != nil:
		handler.ServeHTTP(lrw, r)
	case match.MatchErr == mux.ErrMethodMismatch:
		http.Error(lrw, "Method not allowed.", http.StatusMethodNotAllowed)
	default: // handler == nil or other error
		http.Error(lrw, "Not found.", http.StatusNotFound)
	}

	return nil
}
