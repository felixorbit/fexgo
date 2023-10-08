package fexgo

import (
	"log"
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 只允许 Path 中存在一个 *，后面的不再解析
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)
	for _, item := range vs {
		if item == "" {
			continue
		}
		parts = append(parts, item)
		if item[0] == '*' {
			break
		}
	}
	return parts
}

// 动态路由：一条路由规则可以匹配某一类型而非某一条固定的路由
// 哈希表只能存储静态路由
// 为了支持动态路由，使用前缀树结构保存路由信息
func (r *router) addRoute(method, pattern string, handler HandlerFunc) {
	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parsePattern(pattern), 0)
	log.Printf("router: add route [%s] %s", method, pattern)

	key := method + "-" + pattern
	r.handlers[key] = handler
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	root, ok := r.roots[method]
	if !ok {
		log.Println("router: method not found")
		return nil, nil
	}

	log.Printf("router: get route for [%s] %s\n", method, path)
	searchParts := parsePattern(path)
	n := root.search(searchParts, 0)
	if n == nil {
		return nil, nil
	}
	// 除了匹配节点，还要获取 Path 中的 :var 参数和 *suffix 路径
	params := make(map[string]string)
	parts := parsePattern(n.pattern)
	for index, part := range parts {
		if part[0] == ':' {
			params[part[1:]] = searchParts[index]
		}
		if part[0] == '*' && len(part) > 1 {
			params[part[1:]] = strings.Join(searchParts[index:], "/")
			break
		}
	}
	return n, params
}

// 路由的主要逻辑：根据 URL Path 查找到对应的 Handler
func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern // 必须用匹配节点的 pattern 作为 key
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		c.handlers = append(c.handlers, func(ctx *Context) {
			c.String(http.StatusNotFound, "404 Not Found %s\n", c.Path)
		})
	}
	c.Next()
}
