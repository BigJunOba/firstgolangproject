package main

import (
	"fmt"
	"strings"
	"time"
)

// 使用接口做一个抽象，使读取模块实现接口来实现
type Reader interface {
	Read(rc chan string)
}

// 使用接口做一个抽象，使写入模块实现接口来实现
type Writer interface {
	Write(rc chan string)
}

type LogProcess struct {
	rc    chan string // 读取模块 -> 解析模块
	wc    chan string // 解析模块 -> 写入模块
	read  Reader
	write Writer
}

// 定义读取文件结构体
type ReadFromFile struct {
	path string // 读取文件的路径
}

// 定义写入文件结构体
type WriteToInfluxDB struct {
	influxDBDsn string // influx data source
}

// ReadFromFile 结构体实现了 Reader 接口
func (r *ReadFromFile) Read(rc chan string) {
	// 读取模块
	line := "message"
	rc <- line
}

func (w *WriteToInfluxDB) Write(wc chan string) {
	// 写入模块
	fmt.Println(<-wc)
}

func (l *LogProcess) Process() {
	// 解析模块
	data := <-l.rc
	l.wc <- strings.ToUpper(data)
}

func main() {
	r := &ReadFromFile{
		path: "/temp/access.log",
	}

	w := &WriteToInfluxDB{
		influxDBDsn: "username&password",
	}

	lp := &LogProcess{
		rc:    make(chan string), // 初始化 channel
		wc:    make(chan string), // 初始化 channel
		read:  r,
		write: w,
	}

	go lp.read.Read(lp.rc) // 实际上应该写成这样：(*lp).ReadFromFile() 但是 golang 已经优化了
	go lp.Process()
	go lp.write.Write(lp.wc)

	time.Sleep(1 * time.Second) // 为了不让创建完协程就退出，使其等待一秒钟保证三个协程执行完毕  打印：MESSAGE
}
