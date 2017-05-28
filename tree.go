// Package mux implements a high performance and powerful trie based url path router for Go.
package mux

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Version holds the current mux version
const Version = "0.0.1"

var (
	allowSuffixExt = []string{".json", ".xml", ".html"}
	// *.*  :path  :ext
	extWildRegexp = regexp.MustCompile(`([^.]+).(.+)`)
	// *    :splat
	wildRegexp = regexp.MustCompile(`(.+)`)
	// :string
	wordRegexp = regexp.MustCompile(`^\w+$`)
	// param name only allowed alphabet numbers and _
	paramRegexp = regexp.MustCompile(`^:\w+$`)
	// optional param
	optionalParamRegexp = regexp.MustCompile(`^\?:\w+$`)

	defaultOptions = Options{
		CaseSensitive:  true,
		PathClean:      true,
		StrictSlash:    true,
		UseEncodedPath: true,
	}
)

// Options describes options for Trie.
type Options struct {
	// CaseSensitive when matching URL path.
	CaseSensitive bool

	// PathClean defines the path cleaning behavior for new routes. The default value is false.
	// Users should be careful about which routes are not cleaned
	// When true, the path will be cleaned, if the route path is "/path//to", it will return "/path/to"
	// When false, if the route path is "/path//to", it will remain with the double slash
	PathClean bool

	// 	StrictSlash defines the trailing slash behavior for new routes.
	// The initial value is false.
	// When true, if the route path is "/path/", accessing "/path" will
	//    redirect to the former and vice versa. In other words,
	//    your application will always see the path as specified in the route.
	// When false, if the route path is "/path", accessing "/path/" will
	//    not match this route and vice versa.
	StrictSlash bool

	// UseEncodedPath tells the router to match the encoded original path to the routes.
	// For eg. "/path/foo%2Fbar/to" will match the path "/path/{var}/to".
	// This behavior has the drawback of needing to match routes against r.RequestURI instead of r.URL.Path.
	// Any modifications (such as http.StripPrefix) to r.URL.Path will not affect routing when this flag is
	// on and thus may induce unintended behavior.
	// If not called, the router will match the unencoded path to the routes.
	// For eg. "/path/foo%2Fbar/to" will match the path "/path/foo/bar/to"
	UseEncodedPath bool
}

// NewTrie returns a trie
//
//  trie := New()
//  // disable CaseSensitive, PathClean and StrictSlash
//  trie := New(Options{})
//
func NewTrie(args ...Options) *Trie {
	opts := defaultOptions
	if len(args) > 0 {
		opts = args[0]
	}

	return &Trie{
		caseSensitive:  opts.CaseSensitive,
		pathClean:      opts.PathClean,
		strictSlash:    opts.StrictSlash,
		useEncodedPath: opts.UseEncodedPath,
		root: &Node{
			parent:   nil,
			children: make(map[string]*Node),
			handlers: make(map[string]interface{}),
		},
	}
}

// Trie represents a trie that defining patterns and matching URL.
type Trie struct {
	caseSensitive  bool
	pathClean      bool
	strictSlash    bool
	useEncodedPath bool
	root           *Node
}

// Parse will parse the pattern and returns the endpoint node for the pattern.
//
//  trie := New()
//  node1 := trie.Parse("/a")
//  node2 := trie.Parse("/a/b")
//  node3 := trie.Parse("/a/b")
//  // node2.parent == node1
//  // node2 == node3
func (t *Trie) Parse(pattern string) *Node {
	if strings.Contains(pattern, "//") {
		panic(fmt.Errorf(`multi-slash exist: "%s"`, pattern))
	}
	_pattern := strings.TrimPrefix(pattern, "/")
	if !t.caseSensitive {
		_pattern = strings.ToLower(_pattern)
	}
	node := parsePattern(t.root, strings.Split(_pattern, "/"))
	if node.pattern == "" {
		node.pattern = pattern
	}
	return node
}

// Match try to match path. It will returns a Matched instance that
// includes	*Node, Params when matching success, otherwise a nil.
//
//  matched, err := trie.Match("/a/b")
//
func (t *Trie) Match(path string) (*Matched, error) {
	if path == "" || path[0] != '/' {
		return nil, fmt.Errorf(`path is not start with "/": "%s"`, path)
	}
	if t.pathClean {
		path = pathClean(path)
	}
	if !t.caseSensitive {
		path = strings.ToLower(path)
	}

	start := 1
	end := len(path)
	matched := new(Matched)
	parent := t.root
	for i := 1; i <= end; i++ {
		if i < end && path[i] != '/' {
			continue
		}
		segment := path[start:i]
		node := matchNode(parent, segment, path[i:])
		if node == nil {
			// TrailingSlashRedirect: /abc/efg/ -> /abc/efg
			if parent.endpoint && i == end && segment == "" {
				matched.Path = path[:end-1]
			}
			// match suffixext match
			if i == end {
				for _, ext := range allowSuffixExt {
					if strings.HasSuffix(segment, ext) {
						node = matchNode(parent, strings.TrimSuffix(segment, ext), path[i:])
						if node != nil {
							if matched.Params == nil {
								matched.Params = make(map[string]string)
							}
							matched.Params[":ext"] = ext[1:]
							goto ParentNode
						}
					}
				}
			}
			return matched, nil
		}
	ParentNode:
		parent = node
		if len(parent.name) > 0 {
			if matched.Params == nil {
				matched.Params = make(map[string]string)
			}
			if parent.wildcard {
				// match *
				if len(parent.name) == 1 {
					segs := strings.Split(path[start:end], "/")
					starValue := []string{}
					for {
						if len(segs) > 0 {
							starValue = append(starValue, segs[0])
						}
						if len(segs) == 1 {
							break
						} else {
							segs = segs[1:]
						}
						n := matchNode(parent, segs[0], strings.Join(segs, "/"))
						if n != nil {
							matched.Params[parent.name[0]] = strings.Join(starValue, "/")
							parent = n
							i = i + 1 + len(segs[0])
							start = start + len(strings.Join(starValue, "/"))
							goto END
						} else {
							i = i + 1 + len(segs[0])
						}
					}
					matched.Params[parent.name[0]] = strings.Join(starValue, "/")
				} else {
					// match *.*
					values := parent.regex.FindStringSubmatch(path[start:end])
					if len(values) != len(parent.name)+1 {
						return nil, fmt.Errorf("%s: Find wrong match %v, need names %v", path, values, parent.name)
					}
					for i, name := range parent.name {
						matched.Params[name] = values[i+1]
					}
				}
				break
			} else if parent.regex == nil { // :name
				matched.Params[parent.name[0]] = segment
			} else {
				values := parent.regex.FindStringSubmatch(segment)
				for i, name := range parent.name {
					matched.Params[name] = values[i+1]
				}
			}
		}
		start = i + 1
	END:
	}

	switch {
	case parent.endpoint:
		matched.Node = parent
	case parent.getChild("") != nil:
		// TrailingSlashRedirect: /abc/efg -> /abc/efg/
		matched.Path = path + "/"
	case len(parent.optionChildren) > 0:
		for _, child := range parent.optionChildren {
			matched.Node = child
			break
		}
	}

	return matched, nil
}

// Matched is a result returned by Trie.Match.
type Matched struct {
	// Either a Node pointer when matched or nil
	Node *Node

	// Either a map contained matched values or empty map.
	Params map[string]string

	// Matched path to access
	// If Node is nil then redirect to this PATH
	Path string
}

// Node represents a node on defined patterns that can be matched.
type Node struct {
	name, allow                  []string
	pattern, segment             string
	endpoint, wildcard, optional bool
	parent                       *Node
	segChildren                  []*Node
	optionChildren               []*Node
	varyChildren                 []*Node
	children                     map[string]*Node
	handlers                     map[string]interface{}
	regex                        *regexp.Regexp
	namedRoutes                  map[string]*Node
}

func (n *Node) getSegments() string {
	segments := n.segment
	if n.parent != nil {
		segments = n.parent.getSegments() + "/" + segments
	}
	return segments
}

// getChild return the static key
func (n *Node) getChild(key string) *Node {
	if strings.Contains(key, "::") {
		key = strings.Replace(key, "::", ":", -1)
	}
	if v, ok := n.children[key]; ok {
		return v
	}
	for _, c := range n.segChildren {
		if c.segment == key {
			return c
		}
	}
	for _, c := range n.optionChildren {
		if c.segment == key {
			return c
		}
	}
	for _, c := range n.varyChildren {
		if c.segment == key {
			return c
		}
	}
	return nil
}

// Name sets the name for the route, used to build URLs.
func (n *Node) Name(name string) *Node {
	if n.getRootNode().namedRoutes == nil {
		n.getRootNode().namedRoutes = map[string]*Node{name: n}
	} else {
		if _, ok := n.getRootNode().namedRoutes[name]; ok {
			panic(fmt.Errorf("mux: route already has name %q, can't set", name))
		}
		n.getRootNode().namedRoutes[name] = n
	}
	return n
}

// GetName returns the name for the route, if any.
func (n *Node) GetName(name string) *Node {
	if n.getRootNode().namedRoutes != nil {
		return n.getRootNode().namedRoutes[name]
	}
	return nil
}

// getRootNode will return the Node whose parent is nil
func (n *Node) getRootNode() *Node {
	if n.parent == nil {
		return n
	}
	return n.parent.getRootNode()
}

// BuildURL will builds a URL for the pattern.
func (n *Node) BuildURL(pairs ...string) (*url.URL, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("pairs expect even number key/val, but get %d", len(pairs))
	}
	params := make(map[string]string)
	var key string
	for k, v := range pairs {
		if k%2 == 0 {
			key = v
		} else {
			params[key] = v
		}
	}
	path, err := buildPath(strings.Split(n.pattern, "/"), params)
	if err != nil {
		return nil, err
	}
	return &url.URL{
		Path: path,
	}, nil
}

func buildPath(segments []string, params map[string]string) (string, error) {
	var results []string
	for {
		if len(segments) == 0 {
			break
		}
		segment := segments[0]
		segments = segments[1:]
		if strings.Contains(segment, "::") {
			results = append(results, strings.Replace(segment, "::", ":", -1))
		} else if segment == "*" {
			v, ok := params[":splat"]
			if !ok {
				return "", fmt.Errorf("* need to map to :splat, but the pairs doesn't exist the key :splat")
			}
			results = append(results, v)
		} else if segment == "*.*" {
			if p, ok := params[":path"]; !ok {
				return "", fmt.Errorf("*.* need to map to :path, but the pairs doesn't exist the key :path")
			} else if e, ok := params[":ext"]; !ok {
				return "", fmt.Errorf("*.* need to map to :ext, but the pairs doesn't exist the key :ext")
			} else {
				results = append(results, p+"."+e)
			}
		} else if optionalParamRegexp.MatchString(segment) {
			v, ok := params[segment[1:]]
			if !ok {
				continue
			}
			results = append(results, v)
		} else if paramRegexp.MatchString(segment) {
			v, ok := params[segment]
			if !ok {
				return "", fmt.Errorf("the pairs doesn't exist the key %s", segment)
			}
			results = append(results, v)
		} else if strings.ContainsAny(segment, ":") {
			names, regex, optional := regexpSegment(segment)
			rules := regex.String()
			for _, name := range names {
				if v, ok := params[name]; !ok {
					if optional {
						continue
					} else {
						return "", fmt.Errorf("the pairs doesn't exist the key %s for %s", name, segment)
					}
				} else {
					start := strings.IndexRune(rules, '(')
					end := strings.IndexByte(rules, ')')
					rules = rules[:start] + v + rules[end+1:]
				}
			}
			if rules != regex.String() {
				results = append(results, rules)
			}
		} else {
			results = append(results, segment)
		}
	}
	return strings.Join(results, "/"), nil
}

// Handle is used to mount a handler with a method name to the node.
//
//  t := New()
//  node := t.Define("/a/b")
//  node.Handle("GET", handler1)
//  node.Handle("POST", handler1)
//
func (n *Node) Handle(method string, handler interface{}) {
	if n.GetHandler(method) != nil {
		panic(fmt.Errorf(`"%s" already defined`, n.getSegments()))
	}
	n.handlers[method] = handler
	n.allow = append(n.allow, method)
}

// GetHandler ...
// GetHandler returns handler by method that defined on the node
//
//  trie := New()
//  trie.Parse("/api").Handle("GET", func handler1() {})
//  trie.Parse("/api").Handle("PUT", func handler2() {})
//
//  trie.Match("/api").Node.GetHandler("GET").(func()) == handler1
//  trie.Match("/api").Node.GetHandler("PUT").(func()) == handler2
//
func (n *Node) GetHandler(method string) interface{} {
	return n.handlers[method]
}

// GetAllow returns allow methods defined on the node
//
//  trie := New()
//  trie.Parse("/").Handle("GET", handler1)
//  trie.Parse("/").Handle("PUT", handler2)
//
//  // trie.Match("/").Node.GetAllow() == []string{"GET", "PUT"}
//
func (n *Node) GetAllow() []string {
	return n.allow
}

// parsePattern support multi pattern
func parsePattern(parent *Node, segments []string) *Node {
	segment := segments[0]
	segments = segments[1:]
	child := parseSegment(parent, segment)
	if len(segments) == 0 {
		child.endpoint = true
		return child
	}
	return parsePattern(child, segments)
}

func matchNode(parent *Node, segment, path string) (child *Node) {
	if child = parent.getChild(segment); child != nil {
		return
	}
	for _, child = range parent.segChildren {
		if len(path) > 0 && len(child.children) == 0 &&
			len(child.varyChildren) == 0 && len(child.segChildren) == 0 && len(child.optionChildren) == 0 {
			continue
		}
		if child.regex != nil && !child.regex.MatchString(segment) {
			continue
		}
		return child
	}
	for _, child = range parent.varyChildren {
		if child.regex != nil && !child.regex.MatchString(segment) {
			continue
		}
		return
	}
	return nil
}

// segment support multi type
// ?:id
// :name
// :name(reg)
// :name:int
// :name:string
// *.*
// *
// cms_:id([0-9]+).html
func parseSegment(parent *Node, segment string) *Node {
	if node := parent.getChild(segment); node != nil {
		return node
	}
	node := &Node{
		segment:  segment,
		parent:   parent,
		children: make(map[string]*Node),
		handlers: make(map[string]interface{}),
	}
	// route "/a/" match the last segment empty
	if segment == "" {
		parent.children[segment] = node
		// segment contain any :: will clean up to static segment
		// "::name" convert to ":name"
		// "cms::name::hello" convert to "cms:name:hello"
	} else if strings.Contains(segment, "::") {
		parent.children[strings.Replace(segment, "::", ":", -1)] = node
	} else if segment == "*" {
		node.wildcard = true
		node.regex = wildRegexp
		node.name = []string{":splat"}
		parent.varyChildren = append(parent.varyChildren, node)
	} else if segment == "*.*" {
		node.wildcard = true
		node.regex = extWildRegexp
		node.name = []string{":path", ":ext"}
		parent.varyChildren = append(parent.varyChildren, node)
	} else if optionalParamRegexp.MatchString(segment) {
		node.optional = true
		node.name = []string{segment[1:]}
		parent.optionChildren = append(parent.optionChildren, node)
		parent.segChildren = append(parent.segChildren, node)
	} else if paramRegexp.MatchString(segment) {
		node.name = []string{segment}
		parent.segChildren = append(parent.segChildren, node)
	} else if strings.ContainsAny(segment, ":") {
		node.name, node.regex, node.optional = regexpSegment(segment)
		if node.optional {
			parent.optionChildren = append(parent.optionChildren, node)
		}
		parent.varyChildren = append(parent.varyChildren, node)
	} else {
		parent.children[segment] = node
	}
	return node
}

func regexpSegment(seg string) (params []string, r *regexp.Regexp, optional bool) {
	var (
		expr     []rune
		start    bool
		startexp bool
		param    []rune
		skipnum  int
		err      error
	)
	for i, v := range seg {
		if skipnum > 0 {
			skipnum--
			continue
		}
		// if start is true then it means it's param now
		if start {
			if v == ':' {
				if len(seg) >= i+4 {
					if seg[i+1:i+4] == "int" {
						intRule := "([0-9]+)"
						if optional {
							intRule = "([0-9]+)"
						}
						expr = append(expr, []rune(intRule)...)
						params = append(params, ":"+string(param))
						start = false
						startexp = false
						skipnum = 3
						param = make([]rune, 0)
						continue
					}
				}
				if len(seg) >= i+7 {
					if seg[i+1:i+7] == "string" {
						stringRule := `([\w]+)`
						if optional {
							stringRule = `([\w]*)`
						}
						expr = append(expr, []rune(stringRule)...)
						params = append(params, ":"+string(param))
						start = false
						startexp = false
						skipnum = 6
						param = make([]rune, 0)
						continue
					}
				}
			}
			if wordRegexp.MatchString(string(v)) {
				param = append(param, v)
				continue
			}
			// param name scan finish
			if len(param) > 0 {
				params = append(params, ":"+string(param))
				param = make([]rune, 0)
				start = false
			}
		}
		if startexp {
			if v != ')' {
				expr = append(expr, v)
				continue
			}
		}
		if v == ':' {
			param = make([]rune, 0)
			start = true
		} else if v == '(' {
			startexp = true
			start = false
			expr = append(expr, '(')
		} else if v == ')' {
			startexp = false
			expr = append(expr, ')')
			param = make([]rune, 0)
		} else if v == '?' && len(seg)-1 > i && seg[i+1] == ':' {
			optional = true
		} else {
			if len(expr) == 0 && len(params) > 0 {
				expr = []rune("(.+)")
			}
			expr = append(expr, v)
		}
	}
	if len(param) > 0 {
		params = append(params, ":"+string(param))
		if len(expr) > 0 {
			expr = append(expr, []rune("(.+)")...)
		}
	}
	r, err = regexp.Compile(string(expr))
	if err != nil {
		panic(fmt.Errorf(`Wrong regexp format: "%s"`, string(expr)))
	}
	// valid all params
	for _, p := range params {
		if !paramRegexp.MatchString(p) {
			panic(fmt.Errorf(`Wrong param format: "%s"`, p))
		}
	}
	return
}

func pathClean(path string) string {
	if !strings.Contains(path, "//") {
		return path
	}
	return strings.Replace(path, `//`, "/", -1)
}
