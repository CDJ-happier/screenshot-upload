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
	"path"
	"path/filepath"
	"syscall"
	"time"
)

var port = 8080

func main() {
	server := StartServer(port)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	GracefulShutdown(server)
	time.Sleep(2 * time.Second)
}

// StartServer 启动服务器
func StartServer(port int) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.POST("/api/v1/upload", FilesController)
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	fmt.Printf("Server started on port %d\n", port)
	return server
}

// GracefulShutdown 优雅地关闭服务器
func GracefulShutdown(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to gracefully shutdown server: %v", err)
	}

	fmt.Println("Server gracefully shut down")
}

func FilesController(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		log.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dir := filepath.Dir(exe)
	//filename := fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
	uploads := filepath.Join(dir, "uploads")
	err = os.MkdirAll(uploads, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	fullPath := path.Join("uploads", file.Filename)
	fileErr := c.SaveUploadedFile(file, filepath.Join(dir, fullPath))
	if fileErr != nil {
		log.Fatal(fileErr)
	}
	c.JSON(http.StatusOK, gin.H{"url": "/" + fullPath})
}
