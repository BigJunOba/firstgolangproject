package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	// "github.com/influxdata/influxdb1-client/v2"
)

// 使用接口做一个抽象，使读取模块实现接口来实现
type Reader interface {
	Read(rc chan []byte)
}

// 使用接口做一个抽象，使写入模块实现接口来实现
type Writer interface {
	Write(rc chan *Message)
}

type LogProcess struct {
	rc    chan []byte   // 读取模块 -> 解析模块
	wc    chan *Message // 解析模块 -> 写入模块
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

// 存储提取出来的监控数据
type Message struct {
	TimeLocal                    time.Time // 时间
	BytesSent                    int       // 流量
	Path, Method, Scheme, Status string    // 路径，方法等
	UpstreamTime, RequestTime    float64
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

func (w *WriteToInfluxDB) Write(wc chan *Message) {
	// 写入模块
	for v := range wc {
		fmt.Println(v)
	}
}

func (l *LogProcess) Process() {
	// 解析模块
	// 1.从Read Channel中读取每行日志数据
	// 2.正则提取所需的监控数据(path, status, method等)
	// 3.写入Write Channel
	/**
	172.0.0.12 - - [04/Mar/2018:13:49:52 +0000] http "GET /foo?query=t HTTP/1.0" 200 2133 "-" "KeepAliveClient" "-" 1.005 1.854
	*/

	// 编译正则表达式
	r := regexp.MustCompile(`([\d\.]+)\s+([^ \[]+)\s+([^ \[]+)\s+\[([^\]]+)\]\s+([a-z]+)\s+\"([^"]+)\"\s+(\d{3})\s+(\d+)\s+\"([^"]+)\"\s+\"(.*?)\"\s+\"([\d\.-]+)\"\s+([\d\.-]+)\s+([\d\.-]+)`)

	loc, _ := time.LoadLocation("Asia/Shanghai") // 定义上海时区，通常不会出错，所以忽略掉这个错误
	for v := range l.rc {
		ret := r.FindStringSubmatch(string(v)) // 返回正则表达式括号里面匹配到的内容，一共14个括号
		if len(ret) != 14 {
			log.Println("FindStringSubmatch fail:", string(v))
			continue
		}

		// 初始化结构体
		message := &Message{}
		t, err := time.ParseInLocation("02/Jan/2006:15:04:05 +0000", ret[4], loc)
		if err != nil {
			log.Println("ParseInLocation fail:", err.Error(), ret[4])
		}

		// 时间字段转换完毕
		message.TimeLocal = t

		// ByteSent在第8个位置，2133
		byteSent, _ := strconv.Atoi(ret[8]) // 将字符串转成int
		message.BytesSent = byteSent

		// Path: "GET /foo?query=t HTTP/1.0"
		reqSli := strings.Split(ret[6], " ") // 按照空格进行切割
		if len(reqSli) != 3 {
			log.Println("strings.Split fail:", ret[6])
			continue
		}
		message.Method = reqSli[0]

		u, err := url.Parse(reqSli[1])
		if err != nil {
			log.Println("url parse fail:", err)
			continue
		}
		message.Path = u.Path

		message.Scheme = ret[5]
		message.Status = ret[7]

		upstreamTime, _ := strconv.ParseFloat(ret[12], 64)
		requestTime, _ := strconv.ParseFloat(ret[13], 64)
		message.UpstreamTime = upstreamTime
		message.RequestTime = requestTime

		l.wc <- message
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
		rc:    make(chan []byte),   // 初始化 channel
		wc:    make(chan *Message), // 初始化 channel
		read:  r,
		write: w,
	}

	go lp.read.Read(lp.rc) // 实际上应该写成这样：(*lp).ReadFromFile() 但是 golang 已经优化了
	go lp.Process()
	go lp.write.Write(lp.wc)

	time.Sleep(30 * time.Second) // 为了不让创建完协程就退出，使其等待一秒钟保证三个协程执行完毕  打印：MESSAGE
}
