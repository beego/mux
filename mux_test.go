package mux

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Request(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func TestMux(t *testing.T) {
	t.Run("Mux.Empty method", func(t *testing.T) {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("Should be panic")
			}
		}()
		mux := New()
		mux.Handler("", "/:type", http.NotFoundHandler())
	})
	t.Run("Mux.Handler", func(t *testing.T) {
		assert := assert.New(t)

		mux := New()
		mux.Handler("GET", "/:type", http.NotFoundHandler())

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := http.Get(ts.URL + "/users")
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		res.Body.Close()
	})

	t.Run("Mux.HandlerFunc", func(t *testing.T) {
		assert := assert.New(t)

		mux := New()
		mux.Handle("GET", "/:type", http.NotFound)

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := http.Get(ts.URL + "/users")
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		res.Body.Close()
	})

	t.Run("router with http.Method", func(t *testing.T) {
		assert := assert.New(t)

		handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(req.Method))
		}

		mux := New()
		mux.Get("/", handler)
		mux.Head("/", handler)
		mux.Post("/", handler)
		mux.Put("/", handler)
		mux.Patch("/", handler)
		mux.Delete("/", handler)
		mux.Options("/", handler)

		assert.Panics(func() {
			mux.Get("", handler)
		})

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := http.Get(ts.URL)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ := ioutil.ReadAll(res.Body)
		assert.Equal("GET", string(body))
		res.Body.Close()

		res, err = http.Head(ts.URL)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ = ioutil.ReadAll(res.Body)
		assert.Equal("", string(body))
		res.Body.Close()

		res, err = http.Post(ts.URL, "", nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ = ioutil.ReadAll(res.Body)
		assert.Equal("POST", string(body))
		res.Body.Close()

		res, err = Request("PUT", ts.URL, nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ = ioutil.ReadAll(res.Body)
		assert.Equal("PUT", string(body))
		res.Body.Close()

		res, err = Request("PATCH", ts.URL, nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ = ioutil.ReadAll(res.Body)
		assert.Equal("PATCH", string(body))
		res.Body.Close()

		res, err = Request("DELETE", ts.URL, nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ = ioutil.ReadAll(res.Body)
		assert.Equal("DELETE", string(body))
		res.Body.Close()

		res, err = Request("OPTIONS", ts.URL, nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ = ioutil.ReadAll(res.Body)
		assert.Equal("OPTIONS", string(body))
		res.Body.Close()
	})

	t.Run("automatic handle `OPTIONS` method", func(t *testing.T) {
		assert := assert.New(t)

		handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(req.Method))
		}

		mux := New()
		mux.Get("/", handler)
		mux.Head("/", handler)
		mux.Post("/", handler)
		mux.Put("/", handler)

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := Request("OPTIONS", ts.URL, nil)
		assert.Nil(err)
		assert.Equal(204, res.StatusCode)
		assert.Equal("GET, HEAD, POST, PUT", res.Header.Get("Access-Control-Allow-Methods"))
		res.Body.Close()
	})

	t.Run("router with 404", func(t *testing.T) {
		assert := assert.New(t)

		mux := New()
		mux.Get("/abc", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(204)
		})

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := Request("GET", ts.URL, nil)
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		assert.Equal("nosniff", res.Header.Get("X-Content-Type-Options"))
		assert.Equal("text/plain; charset=utf-8", res.Header.Get("Content-Type"))
		body, _ := ioutil.ReadAll(res.Body)
		assert.Equal(`"/" not implemented`+"\n", string(body))
		res.Body.Close()
	})

	t.Run("router with named pattern", func(t *testing.T) {
		assert := assert.New(t)

		mux := New()
		mux.Get("/api/:type/:ID", func(w http.ResponseWriter, r *http.Request) {
			params := Params(r)
			w.WriteHeader(200)
			w.Write([]byte(params[":type"] + params[":ID"]))
		})

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := Request("GET", ts.URL+"/api/user/123", nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ := ioutil.ReadAll(res.Body)
		assert.Equal("user123", string(body))
		res.Body.Close()
	})

	t.Run("router with double colon pattern", func(t *testing.T) {
		assert := assert.New(t)

		mux := New()
		mux.Get("/api/::/:ID", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(Param(r, ":ID")))
		})

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := Request("GET", ts.URL+"/api/:/123", nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ := ioutil.ReadAll(res.Body)
		assert.Equal("123", string(body))
		res.Body.Close()
	})

	t.Run("router with wildcard pattern", func(t *testing.T) {
		assert := assert.New(t)

		mux := New()
		mux.Get("/api/*", func(w http.ResponseWriter, r *http.Request) {
			params := Params(r)
			w.WriteHeader(200)
			w.Write([]byte(params[":splat"]))
		})

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := Request("GET", ts.URL+"/api/user/123", nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ := ioutil.ReadAll(res.Body)
		assert.Equal("user/123", string(body))
		res.Body.Close()
	})

	t.Run("router with regexp pattern", func(t *testing.T) {
		assert := assert.New(t)

		mux := New()
		mux.Get(`/api/:type/:ID(^\d+$)`, func(w http.ResponseWriter, r *http.Request) {
			params := Params(r)
			w.WriteHeader(200)
			w.Write([]byte(params[":type"] + params[":ID"]))
		})

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := Request("GET", ts.URL+"/api/user/abc", nil)
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		res.Body.Close()

		res, err = Request("GET", ts.URL+"/api/user/123", nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ := ioutil.ReadAll(res.Body)
		assert.Equal("user123", string(body))
		res.Body.Close()
	})

	t.Run("router with DefaultHandler", func(t *testing.T) {
		assert := assert.New(t)

		mux := New()
		mux.Get("/api", func(w http.ResponseWriter, r *http.Request) {
			Params(r)
			Param(r, ":id")
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		})
		mux.DefaultHandler(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte(req.Method + " " + req.URL.Path))
		})

		ts := httptest.NewServer(mux)
		defer ts.Close()

		res, err := Request("GET", ts.URL+"/api", nil)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		body, _ := ioutil.ReadAll(res.Body)
		assert.Equal("OK", string(body))
		res.Body.Close()

		res, err = Request("PUT", ts.URL+"/api", nil)
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		body, _ = ioutil.ReadAll(res.Body)
		assert.Equal("PUT /api", string(body))
		res.Body.Close()

		res, err = Request("GET", ts.URL+"/api/user/abc", nil)
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		body, _ = ioutil.ReadAll(res.Body)
		assert.Equal("GET /api/user/abc", string(body))
		res.Body.Close()
	})

}
