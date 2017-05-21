package mux

import (
	"net/http"
)

type key int

const (
	// RouteParamsID represent the key to store matched route params
	RouteParamsID key = iota
)

// Params return the router params
func Params(r *http.Request) map[string]string {
	v := r.Context().Value(RouteParamsID)
	if v == nil {
		return map[string]string{}
	}
	if v, ok := v.(map[string]string); ok {
		return v
	}
	return map[string]string{}
}

// Param return the router param based on the key
func Param(r *http.Request, key string) string {
	p := Params(r)
	if v, ok := p[key]; ok {
		return v
	}
	return ""
}
