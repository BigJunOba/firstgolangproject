package main

import (
	"fmt"
	"regexp"
)

func main() {
	//	r := regexp.MustCompile(`([\d\.]+)\s
	//+([^ \[]+)\s
	//+([^ \[]+)\s
	//+\[([^\]]+)\]\s
	//+([a-z]+)\s
	//+\"([^"]+)\"\s
	//+(\d{3})\s
	//+(\d+)\s
	//+\"([^"]+)\"\s
	//+\"(.*?)\"\s
	//+\"([\d\.-]+)\"\s
	//+([\d\.-]+)\s
	//+([d\.-]+)`)
	//	text := `172.0.0.12 - - [04/Mar/2018:13:49:52 +0000] http GET /foo?query=t HTTP/1.0 200 2133 - KeepAliveClient - 1.005 1.854`
	//	ret := r.FindStringSubmatch(text)
	//	fmt.Println(ret)

	//r := regexp.MustCompile(`([\d\.]+)\s +([^ \[]+)`)
	//r := regexp.MustCompile(`([\d\.]+)\s + ([^ \[]+)\s`)
	//r := regexp.MustCompile(`([\d\.]+)\s`)
	r := regexp.MustCompile(`([\d\.]+)\s+([^ \[]+)\s+([^ \[]+)\s+\[([^\]]+)\]\s+([a-z]+)\s+\"([^"]+)\"\s+(\d{3})\s+(\d+)\s+\"([^"]+)\"\s+\"(.*?)\"\s+\"([\d\.-]+)\"\s+([\d\.-]+)\s+([\d\.-]+)`)
	//text := `172.0.0.12 - - [04/Mar/2018:13:49:52 +0000] http "GET /foo?query=t HTTP/1.0" 200 2133 "-" "KeepAliveClient" "-" 1.005 1.854 `
	text := `172.0.0.12 - - [10/Apr/2019:11:16:54 +0000] http "GET /qux HTTP/1.0" 200 1014 "-" "KeepAliveClient" "-" 0.479 0.479 `
	ret := r.FindStringSubmatch(text)
	for k, v := range ret {
		fmt.Println(k, v)
	}
	fmt.Println(ret)
}
