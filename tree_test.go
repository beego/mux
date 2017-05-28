package mux

import (
	"strings"
	"testing"
)

func TestRegexpSegment(t *testing.T) {

	items := map[string]struct {
		params   []string
		regStr   string
		optional bool
	}{
		"admin":                      {nil, "admin", false},
		":id":                        {[]string{":id"}, "", false},
		"?:id":                       {[]string{":id"}, "", true},
		":id:int":                    {[]string{":id"}, "([0-9]+)", false},
		":name:string":               {[]string{":name"}, `([\w]+)`, false},
		":id([0-9]+)":                {[]string{":id"}, `([0-9]+)`, false},
		":id([0-9]+)_:name":          {[]string{":id", ":name"}, `([0-9]+)_(.+)`, false},
		":id(.+)_cms.html":           {[]string{":id"}, `(.+)_cms.html`, false},
		"cms_:id(.+)_:page(.+).html": {[]string{":id", ":page"}, `cms_(.+)_(.+).html`, false},
		`:app(a|b|c)`:                {[]string{":app"}, `(a|b|c)`, false},
		`:app\((a|b|c)\)`:            {[]string{":app"}, `(.+)\((a|b|c)\)`, false},
	}

	for pattern, v := range items {
		w, r, o := regexpSegment(pattern)
		if o != v.optional || r.String() != v.regStr || strings.Join(w, ",") != strings.Join(v.params, ",") {
			t.Fatalf("%s should return %s,%q,%t got %s,%q,%t", pattern, v.params, v.regStr, v.optional, w, r.String(), o)
		}
	}
}

var routers = []struct {
	url          string
	requesturl   string
	params       map[string]string
	pathCLean    bool
	skipURLBuild bool
}{
	{"/", "/", nil, false, false},
	{"/customer/login", "/customer/login", nil, false, false},
	{"/:id", "/123", map[string]string{":id": "123"}, false, false},
	{"/customer/login", "/customer/login.json", map[string]string{":ext": "json"}, false, true},
	{"/topic/?:auth:int", "/topic/123", map[string]string{":auth": "123"}, false, false},
	{"/topic/?:auth:int", "/topic", nil, false, false},
	{"/abc/xyz/?:id", "/abc/xyz", nil, false, false},
	{"/topic/:id/?:auth", "/topic/1", map[string]string{":id": "1"}, false, false},
	{"/topic/:id/?:auth", "/topic/1/2", map[string]string{":id": "1", ":auth": "2"}, false, false},
	{"/topic/:id/?:auth:int", "/topic/1", map[string]string{":id": "1"}, false, false},
	{"/topic/:id/?:auth:int", "/topic/1/123", map[string]string{":id": "1", ":auth": "123"}, false, false},
	{"/*", "/http://customer/123/", map[string]string{":splat": "http://customer/123/"}, false, false},
	{"/*", "/customer/2009/12/11", map[string]string{":splat": "customer/2009/12/11"}, false, false},
	{"/aa/*/bb", "/aa/2009/bb", map[string]string{":splat": "2009"}, false, false},
	{"/cc/*/dd", "/cc/2009/11/dd", map[string]string{":splat": "2009/11"}, false, false},
	{"/cc/:id/*", "/cc/2009/11/dd", map[string]string{":id": "2009", ":splat": "11/dd"}, false, false},
	{"/ee/:year/*/ff", "/ee/2009/11/ff", map[string]string{":year": "2009", ":splat": "11"}, false, false},
	{
		"/thumbnail/:size/uploads/*",
		"/thumbnail/100x100/uploads/items/2014/04/20/dPRCdChkUd651t1Hvs18.jpg",
		map[string]string{":size": "100x100", ":splat": "items/2014/04/20/dPRCdChkUd651t1Hvs18.jpg"},
		false,
		false,
	},
	{
		"/dl/:width:int/:height:int/*.*",
		"/dl/48/48/05ac66d9bda00a3acf948c43e306fc9a.jpg",
		map[string]string{":width": "48", ":height": "48", ":ext": "jpg", ":path": "05ac66d9bda00a3acf948c43e306fc9a"},
		false,
		false,
	},
	{"/*.*", "/nice/api.json", map[string]string{":path": "nice/api", ":ext": "json"}, false, false},
	{"/:name/*.*", "/nice/api.json", map[string]string{":name": "nice", ":path": "api", ":ext": "json"}, false, false},
	{"/:name/test/*.*", "/nice/test/api.json", map[string]string{":name": "nice", ":path": "api", ":ext": "json"}, false, false},
	{"/v1/shop/:id:int", "/v1/shop/123", map[string]string{":id": "123"}, false, false},
	{"/v1/shop/:id\\((a|b|c)\\)", "/v1/shop/123(a)", map[string]string{":id": "123"}, false, true},
	{"/v1/shop/:id\\((a|b|c)\\)", "/v1/shop/123(b)", map[string]string{":id": "123"}, false, true},
	{"/v1/shop/:id\\((a|b|c)\\)", "/v1/shop/123(c)", map[string]string{":id": "123"}, false, true},
	{"/:year:int/:month:int/:id/:endid", "/1111/111/aaa/aaa", map[string]string{":year": "1111", ":month": "111", ":id": "aaa", ":endid": "aaa"}, false, false},
	{"/v1/shop/:id/:name", "/v1/shop/123/nike", map[string]string{":id": "123", ":name": "nike"}, false, false},
	{"/v1/shop/:id/account", "/v1/shop/123/account", map[string]string{":id": "123"}, false, false},
	{"/v1/shop/:name:string", "/v1/shop/nike", map[string]string{":name": "nike"}, false, false},
	{"/v1/shop/:id([0-9]+)", "/v1/shop//123", map[string]string{":id": "123"}, true, true},
	{"/v1/shop/:id([0-9]+)_:name", "/v1/shop/123_nike", map[string]string{":id": "123", ":name": "nike"}, false, false},
	{"/v1/shop/:id(.+)_cms.html", "/v1/shop/123_cms.html", map[string]string{":id": "123"}, false, false},
	{"/v1/shop/cms_:id(.+)_:page(.+).html", "/v1/shop/cms_123_1.html", map[string]string{":id": "123", ":page": "1"}, false, false},
	{"/v1/:v/cms/aaa_:id(.+)_:page(.+).html", "/v1/2/cms/aaa_123_1.html", map[string]string{":v": "2", ":id": "123", ":page": "1"}, false, false},
	{"/v1/:v/cms_:id(.+)_:page(.+).html", "/v1/2/cms_123_1.html", map[string]string{":v": "2", ":id": "123", ":page": "1"}, false, false},
	{"/v1/:v(.+)_cms/ttt_:id(.+)_:page(.+).html", "/v1/2_cms/ttt_123_1.html", map[string]string{":v": "2", ":id": "123", ":page": "1"}, false, false},
	{"/api/projects/:pid/members/?:mid", "/api/projects/1/members", map[string]string{":pid": "1"}, false, false},
	{"/api/projects/:pid/members/?:mid", "/api/projects/1/members/2", map[string]string{":pid": "1", ":mid": "2"}, false, false},
}

func TestRouters(t *testing.T) {
	for _, r := range routers {
		tr := NewTrie(Options{PathClean: r.pathCLean, CaseSensitive: true})
		tr.Parse(r.url).Handle("GET", "astaxie")
		tr.Parse(r.url).Handle("POST", "asta")
		m, err := tr.Match(r.requesturl)
		if err != nil {
			t.Fatalf("rule:%s URL:%s err:%s", r.url, r.requesturl, err)
		}
		if m == nil || m.Node == nil {
			t.Fatalf("rule:%s; URL:%s; expect:%v; Get:%#v", r.url, r.requesturl, r.params, m)
		}
		if m.Node.GetHandler("GET") == nil || m.Node.GetHandler("GET").(string) != "astaxie" {
			t.Fatalf("rule:%s; URL:%s; expect:%v; Get:%#v", r.url, r.requesturl, r.params, m)
		}
		if m.Node.GetHandler("POST") == nil || m.Node.GetHandler("POST").(string) != "asta" {
			t.Fatalf("rule:%s; URL:%s; expect:%v; Get:%#v", r.url, r.requesturl, r.params, m)
		}
		if r.params != nil {
			for k, v := range r.params {
				if vv, ok := m.Params[k]; !ok {
					t.Fatal(r.url + "    " + r.requesturl + " get param empty: " + k)
				} else if vv != v {
					t.Fatal("The Rule: " + r.url + "\nThe RequestURL:" + r.requesturl + "\nThe Key is " +
						k + ", The Value should be: " + v + ", but get: " + vv)
				}
			}
		}
	}
}

func TestBuildURL(t *testing.T) {
	for _, r := range routers {
		if r.skipURLBuild {
			continue
		}
		tr := NewTrie(Options{PathClean: r.pathCLean, CaseSensitive: true})
		n := tr.Parse(r.url).Name("Cool")
		if tn := n.GetName("Cool"); tn == nil {
			t.Fatalf("Can't Get Named node")
		} else {
			u, err := tn.BuildURL(mapToSlice(r.params)...)
			if err != nil {
				t.Fatalf("rule:%s; expect:%v; Get err:%#v", r.url, r.requesturl, err)
			}
			if u.String() != r.requesturl {
				t.Fatalf("rule:%s; expect:%v; Get :%#v", r.url, r.requesturl, u.String())
			}
		}
	}
}

func TestUnmatched(t *testing.T) {
	var unrouters = []struct {
		url        string
		requesturl string
		params     map[string]string
		pathCLean  bool
	}{
		{"/topic/:id:int", "/topic/aaa", nil, false},
		{"/abc/?:id:int", "/abc/zzz", nil, false},
	}
	for _, r := range unrouters {
		tr := NewTrie(Options{PathClean: r.pathCLean, CaseSensitive: true})
		tr.Parse(r.url).Handle("GET", "astaxie")
		m, err := tr.Match(r.requesturl)
		if err != nil {
			t.Fatalf("rule:%s URL:%s err:%s", r.url, r.requesturl, err)
		}
		if m.Node != nil {
			t.Fatalf("rule:%s URL:%s matched: %v", r.url, r.requesturl, m.Node)
		}
	}
}

func mapToSlice(m map[string]string) (s []string) {
	for k, v := range m {
		s = append(s, k, v)
	}
	return s
}
