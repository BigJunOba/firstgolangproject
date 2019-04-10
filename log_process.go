package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/influxdata/influxdb1-client/v2"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// 系统监控状态
type SystemInfo struct {
	HandleLine   int     `json:"handleLine"`   // 总处理日志行数
	Tps          float64 `json:"tps"`          // 系统吞出量
	ReadChanLen  int     `json:"readChanLen"`  // read channel 长度
	WriteChanLen int     `json:"writeChanLen"` // write channel 长度
	RunTime      string  `json:"runTime"`      // 运行总时间
	ErrNum       int     `json:"errNum"`       // 错误数
}

const (
	TypeHandleLine = 0
	TypeErrNum     = 1
)

var TypeMonitorChan = make(chan int, 200)

type Monitor struct {
	startTime time.Time
	data      SystemInfo
	tpsSli    []int
}

func (m *Monitor) start(lp *LogProcess) {

	// 起一个协程来处理处理行数和错误数
	go func() {
		for n := range TypeMonitorChan {
			switch n {
			case TypeErrNum:
				m.data.ErrNum += 1
			case TypeHandleLine:
				m.data.HandleLine += 1
			}
		}
	}()

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for {
			<-ticker.C
			m.tpsSli = append(m.tpsSli, m.data.HandleLine)
			if len(m.tpsSli) > 2 {
				m.tpsSli = m.tpsSli[1:]
			}
		}
	}()

	http.HandleFunc("/monitor", func(writer http.ResponseWriter, request *http.Request) {
		m.data.RunTime = time.Now().Sub(m.startTime).String()
		m.data.ReadChanLen = len(lp.rc)
		m.data.WriteChanLen = len(lp.wc)

		if len(m.tpsSli) >= 2 {
			m.data.Tps = float64(m.tpsSli[1]-m.tpsSli[0]) / 5
		}

		ret, _ := json.MarshalIndent(m.data, "", "\t")

		io.WriteString(writer, string(ret))
	})
	http.ListenAndServe(":9193", nil)
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

		TypeMonitorChan <- TypeHandleLine

		// 3.将读取到的文件内容写入到rc里面
		rc <- line[:len(line)-1]
	}
}

func (w *WriteToInfluxDB) Write(wc chan *Message) {
	// 写入模块

	for v := range wc {
		// 解析 influxDBDsn: "http://127.0.0.1:8086@user@password@dbname%s"
		infSli := strings.Split(w.influxDBDsn, "@")

		// Make client
		c, err := client.NewHTTPClient(client.HTTPConfig{
			Addr:     infSli[0],
			Username: infSli[1],
			Password: infSli[2],
		})
		if err != nil {
			fmt.Println("Error creating InfluxDB Client: ", err.Error())
		}

		// Create a new point batch
		bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  infSli[3],
			Precision: infSli[4],
		})

		// Create a point and add to batch
		// Tags: Path, Method, Scheme, Status
		tags := map[string]string{"Path": v.Path, "Method": v.Method, "Scheme": v.Scheme, "Status": v.Status}
		fields := map[string]interface{}{
			"UpstremTime": v.UpstreamTime,
			"RequestTime": v.RequestTime,
			"BytesSent":   v.BytesSent,
		}

		pt, err := client.NewPoint("nginx_log", tags, fields, v.TimeLocal)
		if err != nil {
			fmt.Println("Error: ", err.Error())
		}
		bp.AddPoint(pt)

		// Write the batch
		if err := c.Write(bp); err != nil {
			log.Fatal(err)
		}

		log.Println("write success!")
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
			TypeMonitorChan <- TypeErrNum
			log.Println("FindStringSubmatch fail:", string(v))
			continue
		}

		// 初始化结构体
		message := &Message{}
		t, err := time.ParseInLocation("02/Jan/2006:15:04:05 +0000", ret[4], loc)
		if err != nil {
			TypeMonitorChan <- TypeErrNum
			log.Println("ParseInLocation fail:", err.Error(), ret[4])
			continue
		}

		// 时间字段转换完毕
		message.TimeLocal = t

		// ByteSent在第8个位置，2133
		byteSent, _ := strconv.Atoi(ret[8]) // 将字符串转成int
		message.BytesSent = byteSent

		// Path: "GET /foo?query=t HTTP/1.0"
		reqSli := strings.Split(ret[6], " ") // 按照空格进行切割
		if len(reqSli) != 3 {
			TypeMonitorChan <- TypeErrNum
			log.Println("strings.Split fail:", ret[6])
			continue
		}
		message.Method = reqSli[0]

		u, err := url.Parse(reqSli[1])
		if err != nil {
			TypeMonitorChan <- TypeErrNum
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

	var path, influxDBDsn string

	// 动态传入参数，例如在命令行运行时敲出下面的参数
	// go run log_process.go -path ./access.log -influxDBDsn http://127.0.0.1:8086@user@password@dbname%s
	flag.StringVar(&path, "path", "./access.log", "read file path")
	flag.StringVar(&influxDBDsn, "influxDBDsn", "http://127.0.0.1:8086@junoba@bjtungirc@log_process@s",
		"influxdb data source")
	flag.Parse()

	r := &ReadFromFile{
		path: path, // 当前路径下
	}

	w := &WriteToInfluxDB{
		influxDBDsn: influxDBDsn,
	}

	lp := &LogProcess{
		rc:    make(chan []byte, 200),   // 初始化 channel,给channel添加缓存
		wc:    make(chan *Message, 200), // 初始化 channel,给channel添加缓存
		read:  r,
		write: w,
	}

	go lp.read.Read(lp.rc) // 实际上应该写成这样：(*lp).ReadFromFile() 但是 golang 已经优化了
	for i := 0; i < 2; i++ {
		go lp.Process()
	}
	for i := 0; i < 4; i++ {
		go lp.write.Write(lp.wc)
	}
	// 解析模块慢于读取模块，写入模块慢于解析模块，因此可以多开几个协程去并发执行相对较慢的模块

	m := &Monitor{
		startTime: time.Now(),
		data:      SystemInfo{},
	}
	m.start(lp)
}
