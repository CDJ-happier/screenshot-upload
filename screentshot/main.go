package main

import (
	"bytes"
	"fmt"
	"github.com/daspoet/gowinkey"
	"github.com/kbinani/screenshot"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

var serverIp = "127.0.0.1:8080"

var uploadUrl = "/api/v1/upload"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./xxx.exe ip:port")
		fmt.Println("Example: ./xxx.exe 127.0.0.1:8080")
	} else {
		serverIp = os.Args[1]
	}
	endpoint := fmt.Sprintf("http://%s%s", serverIp, uploadUrl)
	screenFileChan := make(chan string, 10)
	go MonitorKeyboard(screenFileChan)
	for file := range screenFileChan {
		UploadFile(endpoint, file)
	}
}

func UploadFile(endpoint, filePath string) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	// 创建一个带有文件的表单数据
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 将文件添加到表单数据中
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		fmt.Println("Failed to create form file:", err)
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		fmt.Println("Failed to copy file:", err)
		return
	}

	// 不要忘记关闭multipart writer
	err = writer.Close()
	if err != nil {
		fmt.Println("Failed to close multipart writer:", err)
		return
	}

	// 设置请求头
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		fmt.Println("Failed to create request:", err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send request:", err)
		return
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status code:", resp.StatusCode)
		return
	}

	fmt.Printf("File: %s uploaded successfully.", filePath)
}

func MonitorKeyboard(screenFiles chan string) {
	// 监听ESC和F2
	keys := []gowinkey.VirtualKey{
		gowinkey.VK_ESCAPE,
		gowinkey.VK_F2,
	}
	events, stopFunc := gowinkey.Listen(gowinkey.Selective(keys...))
	fmt.Println("start monitor keyboard")
	fmt.Println("press ESC to stop monitor keyboard")
	fmt.Println("press F2 to capture full screen")
	for e := range events {
		fmt.Printf("event: %v\n", e)
		switch e.VirtualKey {
		case gowinkey.VK_ESCAPE:
			if e.State == gowinkey.KeyUp {
				stopFunc()
				close(screenFiles)
			}
		case gowinkey.VK_F2:
			if e.State == gowinkey.KeyUp {
				fileName := GetFileNameByTime()
				err := CaptureFullScreen(fileName)
				if err != nil {
					fmt.Printf("capture full screen error: %v\n", err)
				}
				fmt.Println("capture full screen success")
				screenFiles <- fileName
			}
		}
	}
}

func CaptureFullScreen(fileName string) error {
	bounds := screenshot.GetDisplayBounds(0) // 只有1个display
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		panic(err)
	}
	file, _ := os.Create(fileName)
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("close file error: %v\n", err)
		}
	}(file)
	err = png.Encode(file, img)
	if err != nil {
		fmt.Printf("encode png to file error: %v\n", err)
		return err
	}
	return nil
}

func GetFileNameByTime() string {
	return fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
}
