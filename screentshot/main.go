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
	"os/signal"
	"syscall"
	"time"
)

var serverIp = "127.0.0.1:8080"

var uploadUrl = "/api/v1/upload"
var endpoint = serverIp + uploadUrl

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./xxx.exe ip:port")
		fmt.Println("Example: ./xxx.exe 127.0.0.1:8080")
	} else {
		serverIp = os.Args[1]
	}
	endpoint = fmt.Sprintf("http://%s%s", serverIp, uploadUrl)
	screenFileChan := make(chan string, 10)
	stopChan := make(chan struct{})

	// monitor keyboard for event F12 and ESC
	// capture full screen and send it to screenFileChan when F2 is pressed
	// exit when receive ESC or os.Interrupt signal
	// ESC: ESC pressed -> stop monitor keyboard, close screenFileChan, close stopChan
	// os.Interrupt or os.Kill: close stopChan by main, then this goroutine will exit too.
	go MonitorKeyboard(screenFileChan, stopChan)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// upload screenshot file to server, only exit when screenFileChan is closed.
	// the screenshot goroutine will close screenFileChan when ESC is pressed or receive
	// os.Interrupt signal or os.Kill signal.
	uploadDone := make(chan struct{})
	go func() {
		for file := range screenFileChan {
			UploadFile(endpoint, file)
		}
		uploadDone <- struct{}{}
	}()

	select {
	case <-signalChan:
		// receive os.Interrupt or os.Kill signal -> close stopChan -> MonitorKeyboard close screenFileChan and exit
		// -> upload goroutine will exit too, cause screenFileChan closed.
		close(stopChan)
		fmt.Println("exit cause receive os.Interrupt or os.Kill signal")
	case <-stopChan:
		// ESC is pressed -> MonitorKeyboard close screenFileChan, stopChan, and exit
		// -> upload goroutine will exit, cause screenFileChan closed -> main goroutine will exit too by select stopChan.
		fmt.Println("exit cause receive ESC event")
	}
	<-uploadDone
	fmt.Println("upload all screenshot file done")
}

func MonitorKeyboard(screenFiles chan string, stopChan chan struct{}) {
	// 监听ESC和F2
	keys := []gowinkey.VirtualKey{
		gowinkey.VK_ESCAPE,
		gowinkey.VK_F2,
	}
	events, stopFunc := gowinkey.Listen(gowinkey.Selective(keys...))
	defer close(screenFiles)
	defer stopFunc()
	fmt.Println("start monitor keyboard&mouse")
	fmt.Println("press ESC to stop monitor keyboard")
	fmt.Println("press F2 to capture full screen")
	// note that cannot use following code, because stopChan branch will never be executed.
	// once enter the default branch, it would block on events channel.
	//for {
	//	select {
	//	case <-stopChan:
	//		// ...
	//	default:
	//		for e := range events {
	//			// ...
	//		}
	//	}
	//}
	for {
		select {
		case <-stopChan:
			// no longer send file to screenFiles, so close it to stop the upload goroutine.
			fmt.Println("stop monitor keyboard&mouse")
			return
		case e, ok := <-events:
			if !ok {
				fmt.Println("events channel closed")
				return
			}
			switch e.VirtualKey {
			case gowinkey.VK_ESCAPE:
				if e.State == gowinkey.KeyUp {
					fmt.Println("stop monitor keyboard&mouse")
					close(stopChan)
					return
				}
			case gowinkey.VK_F2:
				if e.State == gowinkey.KeyUp {
					fileName := GetFileNameByTime()
					err := CaptureFullScreen(fileName)
					if err != nil {
						fmt.Printf("capture full screen error: %v\n", err)
					}
					fmt.Printf("capture full screen success: %s\n", fileName)
					screenFiles <- fileName
				}
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

	fmt.Printf("File: %s uploaded to %s successfully.\n", filePath, endpoint)
}

func GetFileNameByTime() string {
	return fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
}
