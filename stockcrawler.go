package main

import (
	"fmt"
	"github.com/go-errors/errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type Stock struct {
	Market string
	Code   string
	Name   string
}

type StockInfo struct {
	Name         string
	Code         string
	Price        float64
	PreviousOpen float64
	Open         float64
	Bid          float64
	Ask          float64
	Volumn       int
}

func extractContent(s string) (map[string]string, error) {
	start := strings.Index(s, "=")
	if start == -1 {
		return nil, errors.New("malformed input string: " + s)
	} else {
		end := strings.LastIndex(s, `"`)
		if end == -1 {
			end = len(s)
		}
		result := make(map[string]string)
		result[s[:start]] = s[start+2 : end]
		return result, nil
	}
}

func GetStockList(q string) ([]Stock, error) {
	// http get query
	resp, err := http.Get("http://smartbox.gtimg.cn/s3/?t=all&q=" + q)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// read response
	defer resp.Body.Close()
	body_byte, _ := ioutil.ReadAll(resp.Body)

	// extract kv pair
	body, err := extractContent(string(body_byte))
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// extract info string
	stock_str, ok := body["v_hint"]
	if !ok {
		return nil, errors.New("malfored stock list query result: " + fmt.Sprintf("%+v", body))
	}

	// parse info
	stock_str_lst := strings.Split(stock_str, "^")
	result := make([]Stock, len(stock_str_lst))
	for i, stock_str := range stock_str_lst {
		info_str_lst := strings.Split(stock_str, "~")
		if len(info_str_lst) > 3 {
			result[i].Market = info_str_lst[0]
			result[i].Code = info_str_lst[1]
			result[i].Name, _ = strconv.Unquote(`"` + info_str_lst[2] + `"`)
		} else {
			return nil, errors.New("malfored stock list query result: " + fmt.Sprintf("%+v", body))
		}
	}
	return result, nil
}

func GetStockInfo(q Stock) (StockInfo, error) {
	// http get query
	resp, err := http.Get("http://qt.gtimg.cn/?q=" + q.Market + q.Code)
	if err != nil {
		return StockInfo{}, errors.Wrap(err, 0)
	}

	// read response
	defer resp.Body.Close()
	body_byte, _ := ioutil.ReadAll(resp.Body)

	// extract kv pair
	body, err := extractContent(string(body_byte))
	if err != nil {
		return StockInfo{}, errors.Wrap(err, 0)
	}

	for key, v := range body {
		if key == "pv_none_match" {
			return StockInfo{}, errors.New("no matched stock: " + q.Name)
		} else {
			stock_str := strings.Split(v, "~")
			if len(stock_str) > 20 {
				result := StockInfo{}
				result.Name = q.Name
				result.Code = stock_str[2]
				result.Price, _ = strconv.ParseFloat(stock_str[3], 64)
				result.PreviousOpen, _ = strconv.ParseFloat(stock_str[4], 64)
				result.Open, _ = strconv.ParseFloat(stock_str[5], 64)
				result.Bid, _ = strconv.ParseFloat(stock_str[9], 64)
				result.Ask, _ = strconv.ParseFloat(stock_str[19], 64)
				result.Volumn, _ = strconv.Atoi(stock_str[6])
				return result, nil
			} else {
				return StockInfo{}, errors.New("malformed stock info query result: " + v)
			}
		}
	}
	return StockInfo{}, errors.New("empty stock info query result.")
}
