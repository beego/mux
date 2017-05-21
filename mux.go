package mux

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Mux is a tire base HTTP request router which can be used to
// dispatch requests to different handler functions.
type Mux struct {
	trie           *Trie
	defaultHandler http.HandlerFunc
}

// New returns a Mux instance.
func New(opts ...Options) *Mux {
	return &Mux{trie: NewTrie(opts...)}
}

// Get registers a new GET route for a path with matching handler in the Mux.
func (m *Mux) Get(pattern string, handler http.HandlerFunc) {
	m.Handle(http.MethodGet, pattern, handler)
}

// Head registers a new HEAD route for a path with matching handler in the Mux.
func (m *Mux) Head(pattern string, handler http.HandlerFunc) {
	m.Handle(http.MethodHead, pattern, handler)
}

// Post registers a new POST route for a path with matching handler in the Mux.
func (m *Mux) Post(pattern string, handler http.HandlerFunc) {
	m.Handle(http.MethodPost, pattern, handler)
}

// Put registers a new PUT route for a path with matching handler in the Mux.
func (m *Mux) Put(pattern string, handler http.HandlerFunc) {
	m.Handle(http.MethodPut, pattern, handler)
}

// Patch registers a new PATCH route for a path with matching handler in the Mux.
func (m *Mux) Patch(pattern string, handler http.HandlerFunc) {
	m.Handle(http.MethodPatch, pattern, handler)
}

// Delete registers a new DELETE route for a path with matching handler in the Mux.
func (m *Mux) Delete(pattern string, handler http.HandlerFunc) {
	m.Handle(http.MethodDelete, pattern, handler)
}

// Options registers a new OPTIONS route for a path with matching handler in the Mux.
func (m *Mux) Options(pattern string, handler http.HandlerFunc) {
	m.Handle(http.MethodOptions, pattern, handler)
}

// DefaultHandler registers a new handler in the Mux
// that will run if there is no other handler matching.
func (m *Mux) DefaultHandler(handler http.HandlerFunc) {
	m.defaultHandler = handler
}

// Handle registers a new handler with method and path in the Mux.
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
func (m *Mux) Handle(method, pattern string, handler http.HandlerFunc) {
	if method == "" {
		panic(fmt.Errorf("invalid method"))
	}
	m.trie.Parse(pattern).Handle(strings.ToUpper(method), handler)
}

// Handler is an adapter which allows the usage of an http.Handler as a
// request handle.
func (m *Mux) Handler(method, path string, handler http.Handler) {
	m.Handle(method, path, func(w http.ResponseWriter, req *http.Request) {
		handler.ServeHTTP(w, req)
	})
}

// ServeHTTP implemented http.Handler interface
func (m *Mux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var handler http.HandlerFunc
	path := req.URL.Path
	method := req.Method
	match, err := m.trie.Match(path)
	if err != nil {
		http.Error(w, fmt.Sprintf(`"Access %s: %s"`, path, err), http.StatusNotImplemented)
		return
	}
	if match.Node == nil {
		// Redirect for slash url
		// Router /a/b   Access PATH /a/b/ Redirect to /a/b
		// Router /a/b/  Access PATH /a/b  Redirect to /a/b/
		if match.Path != "" {
			req.URL.Path = match.Path
			code := http.StatusMovedPermanently
			if method != "GET" {
				code = http.StatusTemporaryRedirect
			}
			http.Redirect(w, req, req.URL.String(), code)
			return
		}
		if m.defaultHandler == nil {
			http.Error(w, fmt.Sprintf(`"%s" not implemented`, path), http.StatusNotFound)
			return
		}
		handler = m.defaultHandler
	} else {
		var ok bool
		if handler, ok = match.Node.GetHandler(method).(http.HandlerFunc); !ok {
			// OPTIONS preflight
			if method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(match.Node.GetAllow(), ", "))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if m.defaultHandler == nil {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(match.Node.GetAllow(), ", "))
				http.Error(w, fmt.Sprintf(`"%s" not allowed in "%s"`, method, path), 405)
				return
			}
			handler = m.defaultHandler
		}
	}
	if match.Params != nil {
		ctx := context.WithValue(req.Context(), routeParamsID, match.Params)
		req = req.WithContext(ctx)
	}
	handler(w, req)
}
