主要功能：截图并自动上传到server。

所使用到的三方包：

* `github.com/kbinani/screenshot`：截屏包
* `github.com/daspoet/gowinkey`：全局keyboard及mouse事件监听
* 以及gin：搭建server

使用场景：在电脑A上按下指定键（如F12）后截屏，并将截图文件上传到电脑B上指定文件夹。

目录说明：

```shell
easy-upload
	fileserver
		main.go：电脑B上的文件服务器程序
	screenshot
		main.go：电脑A上的监听及截图程序
```

原理说明：

* `fileserver/main.go`中通过`gin`启动一个服务器，并监听在8080端口，有一个route为`/api/v1/upload`，向这个endpoint使用POST上传文件后会经过handler处理，并将文件保存到该程序所在目录的upload目录下。
* `screenshot/main.go`中通过两个三方包监听keyboard上的F12并截屏，通过http将该截图文件上传到电脑B。

TODO，做一些扩展。