package fexgo

import (
	"html/template"
	"net/http"
	"path"
	"strings"
)

type HandlerFunc func(*Context)

// Engine 协调整个框架的资源。
// router 作为核心，不直接对外暴露接口，而是对外提供 Engine 模型
type Engine struct {
	*RouterGroup // 支持 RouterGroup 的所有接口
	router       *router
	groups       []*RouterGroup
	funcMap      *template.FuncMap
	templates    *template.Template
}

func NewEngine() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

func Default() *Engine {
	engine := NewEngine()
	engine.Use(Logger(), Recovery())
	return engine
}

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = &funcMap
}

// LoadHTMLGlob 将所有模板加载到内存
// 注意需要先设置 FuncMap
func (e *Engine) LoadHTMLGlob(path string) {
	e.templates = template.Must(template.New("").Funcs(*e.funcMap).ParseGlob(path))
}

// ServeHTTP 处理请求时，将 Writer 和 Req 封装为 Context 实例
// 1. 根据 URL 解析分组中间件
// 2. 由 Router 实现具体路由
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := newContext(w, r)
	c.engine = e
	for _, group := range e.groups {
		if strings.HasPrefix(r.URL.Path, group.prefix) {
			c.handlers = append(c.handlers, group.middlewares...)
		}
	}

	e.router.handle(c)
}

func (e *Engine) Run(addr string) error {
	return http.ListenAndServe(addr, e)
}

// RouterGroup 保存 Path 前缀，以前缀划分分组，支持嵌套
type RouterGroup struct {
	prefix      string // 分组完整前缀
	parent      *RouterGroup
	engine      *Engine // 支持分组之上（带分组前缀）的路由注册等任务
	middlewares []HandlerFunc
}

// Group 创建分组，支持嵌套创建
func (rg *RouterGroup) Group(name string) *RouterGroup {
	engine := rg.engine
	group := &RouterGroup{
		prefix: rg.prefix + name,
		parent: rg,
		engine: engine,
	}
	engine.groups = append(engine.groups, group)
	return group
}

func (rg *RouterGroup) addRoute(method, pattern string, handler HandlerFunc) {
	fullPattern := rg.prefix + pattern
	rg.engine.router.addRoute(method, fullPattern, handler)
}

func (rg *RouterGroup) GET(pattern string, handler HandlerFunc) {
	rg.addRoute("GET", pattern, handler)
}

func (rg *RouterGroup) POST(pattern string, handler HandlerFunc) {
	rg.addRoute("POST", pattern, handler)
}

func (rg *RouterGroup) Use(middlewares ...HandlerFunc) {
	rg.middlewares = append(rg.middlewares, middlewares...)
}

func (rg *RouterGroup) createStaticHandler(relativeDir string, fs http.FileSystem) HandlerFunc {
	absoluteDir := path.Join(rg.prefix, relativeDir)
	fileServer := http.StripPrefix(absoluteDir, http.FileServer(fs))
	return func(ctx *Context) {
		file := ctx.Param("filepath")
		if _, err := fs.Open(file); err != nil {
			ctx.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(ctx.Writer, ctx.Req)
	}
}

// Static 绑定静态资源目录。使用 http.FileServer 完成静态文件服务
func (rg *RouterGroup) Static(relativeDir, rootDir string) {
	handler := rg.createStaticHandler(relativeDir, http.Dir(rootDir))
	pathPattern := path.Join(relativeDir, "*filepath")
	rg.GET(pathPattern, handler)
}
