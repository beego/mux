package mux

import (
	"fmt"
	"log"
	"net/http"
	"testing"
)

func TestMain(t *testing.T) {
	mx := New()
	mx.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, beego mux"))
	})
	mx.Get("/abc/:id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello, abc page %s", Param(r, ":id"))
	})

	mx.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello")
	})

	AddSuffixExt(".hahahahah")
	fmt.Println(GetSuffixExts())
	AddSuffixExt(".xixixixixi")
	fmt.Println(GetSuffixExts())
	RemoveSuffixExt(".hahahahah")
	fmt.Println(GetSuffixExts())

	log.Fatal(http.ListenAndServe("127.0.0.1:9999", mx))
}
