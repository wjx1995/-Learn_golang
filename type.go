package main

import (
	"encoding/json"
	"fmt"
)

type A struct {
	Id   int64
	Name string
	Json []byte
}

type C struct {
	Enable bool
}


func main() {
	ret := []A{A{Name: "system_configuration", Enable: true}, A{Name: "system", Enable: false}}

    // 第一步吧数据去除:
    x := map[string]*C
    // 反解byte 为接口体方便 更新 
    json.Unmarshal(a.Json, &x)
    // 反解byte 为接口体方便 更新 
    if t,ok := x["xxx"] ; ok {
        t.Enable = false
    }
    // 数据转换更新数据库表:
    bts, err := json.Marshal(x)
    a.Json = bts
	for _, v := range ret {
		tmp1 := make(map[string]interface{})
		tmp2 := make(map[string]interface{})
		tmp2["enable"] = v.Enable
		tmp1[v.Name] = tmp2
		a = append(a, tmp1)
	}

	fmt.Println(a)
	x, err := json.Marshal(a)
	fmt.Println(string(x), err)
}
