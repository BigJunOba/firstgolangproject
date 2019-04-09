package main

import (
	"fmt"
	"strings"
	"time"
)

type LogProcess struct {
	rc          chan string // 读取模块 -> 解析模块
	wc          chan string // 解析模块 -> 写入模块
	path        string      // 读取文件的路径
	influxDBDsn string      // influx data source
}

func (l *LogProcess) ReadFromFile() {
	// 读取模块
	line := "message"
	l.rc <- line
}

func (l *LogProcess) Process() {
	// 解析模块
	data := <-l.rc
	l.wc <- strings.ToUpper(data)
}

func (l *LogProcess) WriteToInfluxDB() {
	// 写入模块
	fmt.Println(<-l.wc)
}

func main() {
	lp := &LogProcess{
		rc:          make(chan string), // 初始化 channel
		wc:          make(chan string), // 初始化 channel
		path:        "/temp/access.log",
		influxDBDsn: "username&password",
	}

	go lp.ReadFromFile() // 实际上应该写成这样：(*lp).ReadFromFile() 但是 golang 已经优化了
	go lp.Process()
	go lp.WriteToInfluxDB()

	time.Sleep(1 * time.Second) // 为了不让创建完协程就退出，使其等待一秒钟保证三个协程执行完毕  打印：MESSAGE
}
