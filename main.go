package main

import (
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"github.com/go-errors/errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Wechat xml struct
type WxMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string
	FromUserName string
	CreateTime   int
	MsgType      string
	Content      string
	PicUrl       string
	MediaId      string
	Format       string
	Recognition  string
	ThunbMediaId string
	Location_X   float32
	Location_Y   float32
	Scale        int
	Label        string
	MsgId        int64
}

type WxMsgText struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDataNode
	FromUserName CDataNode
	CreateTime   int
	MsgType      CDataNode
	Content      CDataNode
}

type CDataNode struct {
	Val string `xml:",cdata"`
}

func checkParam(v url.Values, criterion map[string]string) error {
	for key, _ := range criterion {
		// check existence of each key
		if _, ok := v[key]; !ok {
			return errors.New(fmt.Sprintf("key '%s' does not exists", key))
		}
	}
	return nil
}

// generate reply text message
func replyText(from string, to string, content string) string {
	msg := WxMsgText{
		ToUserName:   CDataNode{Val: to},
		FromUserName: CDataNode{Val: from},
		CreateTime:   int(time.Now().Unix()),
		MsgType:      CDataNode{Val: "text"},
		Content:      CDataNode{Val: content},
	}
	output, err := xml.Marshal(msg)
	if err != nil {
		log.Println("[ERROR]", err)
		return ""
	} else {
		return string(output)
	}
}

// handle wechat http request
func webWx(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Println("[DEBUG]", "Form:", r.Form)

	// token verification
	if r.Method == "GET" {
		if err := checkParam(r.Form, map[string]string{
			"signature": "",
			"timestamp": "",
			"nonce":     "",
			"echostr":   "",
		}); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Error: ", err)
			log.Print("[Error]", err)
			return
		}

		signature := strings.Join(r.Form["signature"], "")
		timestamp := strings.Join(r.Form["timestamp"], "")
		nonce := strings.Join(r.Form["nonce"], "")
		echostr := strings.Join(r.Form["echostr"], "")
		token := "glymestock123alarm"

		list := []string{token, timestamp, nonce}
		sort.Strings(list)

		fmt.Println(list)
		sha1 := sha1.New()
		for _, s := range list {
			io.WriteString(sha1, s)
		}
		hashcode := fmt.Sprintf("%x", sha1.Sum(nil))

		//log.Println("[DEBUG]", "Signature:", hashcode, signature)

		// return echostr if token is verified
		if hashcode == signature {
			fmt.Fprint(w, echostr)
		} else {
			fmt.Fprint(w, "")
		}
	} else if r.Method == "POST" {
		// wechat message passive reply
		body_byte, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		//log.Println("[DEBUG]", "log body:", bodyStr)

		var body WxMsg
		xml.Unmarshal(body_byte, &body)

		var replyMsg string

		stocks, err := GetStockList(body.Content)
		if err != nil {
			replyMsg = replyText(body.ToUserName, body.FromUserName, "没有找到相关的股票")
			log.Println("[ERROR]", err.(*errors.Error).ErrorStack())
		} else {
			stock_info, err := GetStockInfo(stocks[0])
			if err != nil {
				replyMsg = replyText(body.ToUserName, body.FromUserName,
					fmt.Sprintf("%s(%s) 获取信息出错", stock_info.Name, stock_info.Code))
				log.Println("[ERROR]", err.(*errors.Error).ErrorStack())
			} else {
				replyMsg = replyText(body.ToUserName, body.FromUserName,
					fmt.Sprintf("%s(%s) 现价: %.3f", stock_info.Name, stock_info.Code, stock_info.Price))
			}
		}
		fmt.Fprint(w, replyMsg)
	}
}

func main() {
	http.HandleFunc("/wx", webWx)
	var err = http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal("ListenAndServe error:", err)
	} else {
		log.Println("[DEBUG]", "Serving at :80")
	}
}
