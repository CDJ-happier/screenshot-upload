package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var port = "8080"

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ./xxx.exe port")
		fmt.Println("Example: ./xxx.exe 8080(default to use)")
		fmt.Println()
	} else {
		port = os.Args[1]
	}
	server := StartServer(port)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	GracefulShutdown(server)
	time.Sleep(1 * time.Second)
}

// StartServer 启动服务器
func StartServer(port string) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.GET("/ping", PongHandler)
	router.POST("/api/v1/upload", FileHandler)
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
	fmt.Printf("Server started on port %s\n", port)
	PrintRoutes(router)
	return server
}

// GracefulShutdown 优雅地关闭服务器
func GracefulShutdown(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to gracefully shutdown server: %v", err)
	}

	fmt.Println("Server gracefully shut down ...")
}

// PongHandler 处理 /ping 请求
func PongHandler(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}

// FileHandler 处理文件上传, 保存到 uploads 目录下
func FileHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		log.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dir := filepath.Dir(exe)
	uploadDir := filepath.Join(dir, "uploads")
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	fileErr := c.SaveUploadedFile(file, filepath.Join(uploadDir, file.Filename))
	if fileErr != nil {
		log.Fatal(fileErr)
	}
	c.JSON(http.StatusOK, gin.H{"url": "/uploads" + file.Filename})
}

// PrintRoutes 打印所有路由
func PrintRoutes(r *gin.Engine) {
	routes := r.Routes()
	for _, route := range routes {
		fmt.Printf("%-6s %-15s %s\n", route.Method, route.Path, route.Handler)
	}
}
