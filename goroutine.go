package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
)

func process(w http.ResponseWriter, _ *http.Request) {
	x := rand.Int63n(500)
	fmt.Println("sleep:", x+500)
	time.Sleep(time.Duration(x+500) * time.Millisecond)
}

func main() {
	handler := http.HandlerFunc(process)
	limitedHandler := NewLimiter(handler, 1, 100, time.Second)
	log.Fatal(http.ListenAndServe(":8080", limitedHandler))
}

type counter struct {
	count   uint64
	creatAt int64
	sync.Mutex
}

type limiter struct {
	sync.Mutex
	requests map[string]*counter
	limit    uint64
	next     http.Handler
	tasks    chan task
}

func NewLimiter(h http.Handler, limit uint64, qsize int, timeout time.Duration) *limiter {
	x := &limiter{
		requests: make(map[string]*counter),
		limit:    limit,
		next:     h,
		tasks:    make(chan task, qsize),
	}

	// 循环处理队列
	go func() {
		for task := range x.tasks {
			task.l.ServeHTTP(task.w, task.r)
		}
	}()

	// 超时清除数据
	go func() {
		for {
			// 没秒统计一次
			t := <-time.After(time.Second)
			now := t.Unix()
			expired := now - int64(timeout.Seconds())
			x.Lock()
			for k, v := range x.requests {
				if expired >= v.creatAt {
					delete(x.requests, k)
				}
			}
			x.Unlock()
		}
	}()
	return x
}

func newCounter() *counter {
	return &counter{creatAt: time.Now().Unix()}
}

type task struct {
	l *limiter
	w http.ResponseWriter
	r *http.Request
}

// 类型责任链传递处理
func (l *limiter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println(remoteIP)

	var m map[string]string
	m = make(map[string]string)
	ret, _ := GetIpDetail(remoteIP)
	m["Ip"] = ret.Data.Ip
	m["Cityi"] = ret.Data.City
	m["Country"] = ret.Data.Country
	fmt.Println(m)
	fmt.Println(ret.Data.Ip, ret.Data.City, ret.Data.Country)

	l.Lock()
	c := l.requests[remoteIP]
	if c == nil {
		c = newCounter()
		l.requests[remoteIP] = c
	}
	l.Unlock()
	fromTask := (r.Header.Get("limit") != "")

	c.Lock()
	limitExceeded := c.count > l.limit
	if !fromTask {
		c.count++
	}
	c.Unlock()
	if limitExceeded {
		// 防止死锁任务列表中的任务可以丢弃
		if fromTask {
			select {
			case l.tasks <- task{l: l, w: w, r: r}:
			default: // 最老的数据回丢弃
			}
		} else {
			l.tasks <- task{l: l, w: w, r: r}
		}
		r.Header.Add("limit", fmt.Sprint(c.count))
		fmt.Println("limit ", l.limit, " count ", c.count)
		return
	}
	l.next.ServeHTTP(w, r)
}

var empty IpInfo

type IpInfo struct {
	Data struct {
		Ip      string `json:"ip"`
		Country string `json:"country"`
		City    string `json:"city"`
	} `json:"data"`
}

func GetIpDetail(ip string) (IpInfo, error) {
	resp, err := http.Get(fmt.Sprintf("http://ip.taobao.com/service/getIpInfo.php?ip=%s", ip))
	if err != nil {
		glog.Errorf("userip:%q is not IP:port", ip)
		return empty, err
	}

	defer resp.Body.Close()
	bts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return empty, err
	}

	var ret IpInfo
	json.Unmarshal(bts, &ret)
	return ret, err
}
