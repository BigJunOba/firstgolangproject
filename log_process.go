package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// 使用接口做一个抽象，使读取模块实现接口来实现
type Reader interface {
	Read(rc chan []byte)
}

// 使用接口做一个抽象，使写入模块实现接口来实现
type Writer interface {
	Write(rc chan string)
}

type LogProcess struct {
	rc    chan []byte // 读取模块 -> 解析模块
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
func (r *ReadFromFile) Read(rc chan []byte) {
	// 读取模块
	// 1.打开文件
	f, err := os.Open(r.path)
	if err != nil {
		panic(fmt.Sprintf("open file error:%s", err.Error()))
	}
	// 2.从文件末尾开始逐行读取文件内容
	f.Seek(0, 2) // 将文件的字符指针移动到末尾
	rd := bufio.NewReader(f)
	// 循环读取文件内容
	for {
		line, err := rd.ReadBytes('\n') // 读取文件内容，直到遇到了换行符为止
		if err == io.EOF {              // 读取到文件末尾就sleep500毫秒
			time.Sleep(500 * time.Millisecond)
			continue
		} else if err != nil {
			panic(fmt.Sprintf("ReadBytes error:%s", err.Error()))
		}
		// 3.将读取到的文件内容写入到rc里面
		rc <- line[:len(line)-1]
	}
}

func (w *WriteToInfluxDB) Write(wc chan string) {
	// 写入模块
	for v := range wc {
		fmt.Println(v)
	}
}

func (l *LogProcess) Process() {
	// 解析模块
	for v := range l.rc {
		l.wc <- strings.ToUpper(string(v))
	}
}

func main() {
	r := &ReadFromFile{
		path: "./access.log", // 当前路径下
	}

	w := &WriteToInfluxDB{
		influxDBDsn: "username&password",
	}

	lp := &LogProcess{
		rc:    make(chan []byte), // 初始化 channel
		wc:    make(chan string), // 初始化 channel
		read:  r,
		write: w,
	}

	go lp.read.Read(lp.rc) // 实际上应该写成这样：(*lp).ReadFromFile() 但是 golang 已经优化了
	go lp.Process()
	go lp.write.Write(lp.wc)

	time.Sleep(30 * time.Second) // 为了不让创建完协程就退出，使其等待一秒钟保证三个协程执行完毕  打印：MESSAGE
}
