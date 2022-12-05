package main

import (
	bytes2 "bytes"
	"code.sajari.com/docconv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Params struct {
	Hl string `json:"hl"`
	Q  string `json:"q"`
}

const (
	URL       = "https://scholar.google.com/scholar?"
	userAgent = "Mozilla/5.0"
)

func getRef(filename string) (res string) {
	pdf2str, err := docconv.ConvertPath(filename)
	if err != nil {
		log.Fatal(err)
	}
	pdfBody := pdf2str.Body
	refIdx := strings.Index(pdfBody, "REFERENCES")
	res = pdfBody[refIdx:]
	return res
}

func searchAtGoogle(content string) (result string) {
	params := Params{
		Hl: "zh-CN",
		Q:  content,
	}
	paramsJson, err := json.Marshal(params)
	if err != nil {
		log.Fatal(err)
	}
	reader := bytes2.NewReader(paramsJson)
	// 创建请求
	//req, _ := http.NewRequest("GET", URL, nil)

	req, err := http.NewRequest("POST", URL, reader)
	if err != nil {
		log.Fatal(err)
	}
	// 请求头中添加指定user agent
	req.Header.Add("User-Agent", userAgent)

	client := &http.Client{}
	// 发起请求并返回结果
	netRes, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	// 获取响应体
	body := netRes.Body
	defer body.Close()

	// 创建文件，保存响应内容
	file, _ := os.Create("search_res.txt")
	defer file.Close()

	// 创建一个multiWriter 会同时写入标准输出和os文件
	dest := io.MultiWriter(os.Stdout, file)

	// 将响应内容复制到multiWriter每个目标，返回总的字节数
	bytes, _ := io.Copy(dest, body)

	// 打印总的字节数
	fmt.Println("total bytes: ", bytes)

	return "hello"
}

func main() {
	content := "security"
	searchAtGoogle(content)
}
