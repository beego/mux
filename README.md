# mux   

[![Build Status](https://travis-ci.org/beego/mux.svg?branch=master)](https://travis-ci.org/beego/mux)
[![Coverage Status](https://coveralls.io/repos/github/beego/mux/badge.svg?branch=master)](https://coveralls.io/github/beego/mux?branch=master)
[![GoDoc](https://godoc.org/github.com/beego/mux?status.svg)](http://godoc.org/github.com/beego/mux)

A high performance and powerful trie based url path router for Go.

This router supports fixed and regex rules in routing pattern, and matches request method. It's optimized by trie structure for faster matching and large scale rules.

requirement: **Go 1.7+**

## Feature

*todo*

## Usage

This is a simple example for `mux`. Read [godoc](https://godoc.org/github.com/beego/mux) to get full api documentation.

There is a basic example:

```go
package main

import (
    	"fmt"
	"log"
	"net/http"

	"github.com/beego/mux"
)

func main() {
	mx := mux.New()
	mx.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, beego mux"))
	})
	mx.Get("/abc/:id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello, abc page %s", mux.Param(r,":id"))
	})

	log.Fatal(http.ListenAndServe("127.0.0.1:9999", mx))
}
```

Register route mapping as `http.HandleFunc` via http method name:

```go
mx.Get("/get", getHandleFunc)
mx.Post("/post", postHandleFunc)
mx.Put("/put", putHandleFunc)
mx.Delete("/delete", deleteHandleFunc)
mx.Head("/head", headHandleFunc)
mx.Options("/options", optionsHandleFunc)
mx.Patch("/patch", patchHandleFunc)
```

Or use raw api to add route with http method name:

```go
mx.Handle("GET", "/abc", abcHandleFunc)
```

Register `http.Handle`.

```go
mx2 := mux.New()
mx.Get("/ttt", getHandleFunc)
mx.Handler("GET","/abc", mx2) // /abc/ttt -> getHandleFunc
```

#### default handler

Register default handle to resolve missing matches. If can not find matched pattern, `mux` runs default handler if set.

```go
mx.Get("/abc",abcHandleFunc)
mx.DefaultHandler(defaultHandleFunc)
// abc -> abcHandleFunc
// ^abc -> defaultHandleFunc
```

-----

## Routing

The routing pattern can set as fixed pattern as most simple way. When using fixed pattern, it supports to parse `json`, `xml` and `html` extension to match pattern.

```
Pattern: /abc/xyz

/abc/xyz        matched
/abc/xyz.html   matched 
/abc/xyz.json   matched 
/abc/xyz.xml    matched 
/abc/xzz        no matched
```

But in common cases, you need parameters to match differenct segments in path.

### Named parameters

As you see, `:id` is a **named parameter**. The matched parameters can read one via `mux.Param` method from `*http.Request` by parameter's name.

```go
// r is *http.Request
fmt.Println(mux.Param(r,":id"))
```

Or read all parameters by `mux.Params`.

```go
// r is *http.Request
fmt.Println(mux.Params(r))
// e.g. map[:id:1 :name:beego]
```

A named parameter only can match single segment of path with extension.

```
Pattern: /abc/:id

/abc/           no match
/abc/123        matched     (:id is 123)
/abc/xyz        matched     (:id is xyz)
/abc/123/xyz    no matched
/abc/123.html   matched     (:id is 123.html)
```

### Wildcard parameters

If you need to match several segments in path, use `*` and `*.*` named **wildcard parameters**.

`*` matches all segments between previous and next segment node in pattern. The matched segement parts are stored in params with key `:splat`.

```
Pattern: /abc/*/xyz

/abc/xyz                no match
/abc/123/xyz            matched     (:splat is 123)
/abc/12/34/xyz          matched     (:splat is 12/34)  
```

`*.*` has familar behaviour with `*`, but matched results are two parts, `:path` as path segment and `:ext` as extension suffix.

```
Pattern : /abc/*.*

/abc/xyz.json           matched     (:path is xyz, :ext is json)
/abc/123/xyz.html       matched     (:path is 123/xyz, :ext is html)
```

### Regexp parameters

`mux` supports a regular expression as a paramter , named **regexp paramaters**. You can set a regexp into pattern with a name placeholder.

```
Pattern : /abc/:id([0-9]+)

/abc/123                matched     (:id is 123)
/abc/xyz                no matched
```

You can set value type for one named paramater to simplify some common regexp rules. Now support `:int` ([0-9]+) and `:string` ([\w]+).

```
Pattern: /abc/:id:int

/abc/123        matched (:id is 123)
/abc/xyz        no match
```

Regexp paramters can match several parts in one segment in path.

```
Pattern: /abc/article_:id:int

/abc/123            no matched
/abc/article_123    matched     (:id is 123)
/abc/article_xyz    no matched
```

#### Optional parameters

If the parameter can be not found in pattern when matching url, use `?` to declare this situation. `?` support named and regexp parameters.

```
Pattern: /abc/xyz/?:id

/abc/xyz/               matched     (:id is empty)
/abc/xyz/123            matched     (:id is 123)
```

```
Pattern: /abc/xyz/?:id:int

/abc/xyz/               matched     (:id is empty)
/abc/xyz/123            matched     (:id is 123)
/abc/xyz/aaa            no matched
```

#### Complex patterns

The fixed segements, named parameters and regexp patterns can be used in one rule together.

```
Pattern: /article/:id/comment_:page:int

/article/12/comment_2       matched     (:id is 12, :page is 2)
/article/abc/comment_3      matched     (:id is abc, :page is 3)
/article/abc/comment_xyz    no match
```

```
Pattern: /data/:year/*/list

/data/2012/11/12/list       matched     (:year is 2012, :splat is 11/12)
/data/2014/12/list          matched     (:year is 2014, :splat is 12)
```

```
Pattern: /pic/:width:int/:height:int/*.*

/pic/20/20/aaaaaa.jpg      matched     (:width is 20, :height is 20, :path is aaaaaa, :ext is jpg)
```

#### pattern matching order

Static pattern > parameters' pattern > regexp pattern.

```
URL : /abc/99

pattern: /abc/99            matched
pattern: /abc/:id           no match
pattern: /abc/:id:int       no match

URL : /abc/123

pattern: /abc/99            no match
pattern: /abc/:id           matched    (:id is 123)
pattern: /abc/:id:int       no match
```

If register confusing patterns, it matches first one in adding order. For example, in regexp patterns:

```go
mx := mux.New()

mx.Get("/abc/?:id:int", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "abc, int params %v", mux.Params(r))
})
mx.Get("/abc/?:name:string", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "abc, string params %v", mux.Params(r))
})

mx.Get("/xyz/?:name:string", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "xyz, string params %v", mux.Params(r))
})
mx.Get("/xyz/?:id:int", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "xyz, int params %v", mux.Params(r))
})
```

When using this `mx` to match urls, it shows result:

```
URL				Pattern
/abc		->		/abc/?:id:int			(first one)
/abc/123	->		/abc/?:id:int
/abc/zzz	->		/abc/?:name:string

/xyz		->		/xyz/?:name:string		(first one)
/xyz/123	->		/xyz/?:name:string		(123 is treated as string "123")
/xyz/zzz	->		/xyz/?:name:string
```
