package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/felixorbit/fexgo"
)

func onlyForV2() fexgo.HandlerFunc {
	return func(ctx *fexgo.Context) {
		t := time.Now()
		ctx.Next()
		log.Printf("[%d] %s in %v for group v2", ctx.StatusCode, ctx.Req.RequestURI, time.Since(t))
	}
}

type student struct {
	Name string
	Age  int8
}

func FormatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

func main() {
	r := fexgo.NewEngine()

	r.Use(fexgo.Logger(), fexgo.Recovery())

	// /hello?name=xxx
	r.GET("/hello", func(c *fexgo.Context) {
		c.String(http.StatusOK, "hello %s, you are at %s\n", c.Query("name"), c.Path)
	})
	// 获取参数
	r.GET("/hello/:name", func(c *fexgo.Context) {
		c.String(http.StatusOK, "hello %s, you are at %s\n", c.Param("name"), c.Path)
	})
	r.GET("/files/*filepath", func(c *fexgo.Context) {
		c.JSON(http.StatusOK, fexgo.H{"filepath": c.Param("filepath")})
	})
	r.POST("/login", func(c *fexgo.Context) {
		c.JSON(http.StatusOK, fexgo.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})

	// 分组
	v1 := r.Group("/v1")
	v1.GET("/hello", func(c *fexgo.Context) {
		c.String(http.StatusOK, "hello %s, you are at %s\n", c.Query("name"), c.Path)
	})
	v2 := r.Group("/v2")
	v2.Use(onlyForV2())
	v2.GET("/hello", func(c *fexgo.Context) {
		c.String(http.StatusOK, "hello %s, you are at %s\n", c.Query("name"), c.Path)
	})

	// 静态文件服务
	r.Static("/assets", "./example/static")

	// 服务端渲染模板
	r.SetFuncMap(template.FuncMap{
		"FormatAsDate": FormatAsDate,
	})
	r.LoadHTMLGlob("example/templates/*")
	// 1
	r.GET("/", func(c *fexgo.Context) {
		c.HTML(http.StatusOK, "css.tmpl", nil)
	})
	// 2 列表渲染
	stu1 := &student{Name: "Tom", Age: 10}
	stu2 := &student{Name: "Bob", Age: 12}
	r.GET("/students", func(c *fexgo.Context) {
		c.HTML(http.StatusOK, "arr.tmpl", fexgo.H{
			"title":  "fex",
			"stuArr": [2]*student{stu1, stu2},
		})
	})
	// 3 渲染函数
	r.GET("/date", func(c *fexgo.Context) {
		c.HTML(http.StatusOK, "custom_func.tmpl", fexgo.H{
			"title": "fex",
			"now":   time.Date(2019, 8, 17, 0, 0, 0, 0, time.UTC),
		})
	})

	// 错误恢复
	r.GET("/panic", func(c *fexgo.Context) {
		names := []string{"test"}
		c.String(http.StatusOK, names[100])
	})

	r.Run(":9999")
}
