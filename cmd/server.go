package main

import (
	"bufio"
	"fmt"
	"github.com/wendev/hsdb/parser"
	"os"
	"strings"
)

// 数据库系统的服务端
// 建立服务端监听，循环接入客户端，在每一个单独的协程中为每一个具体的客户端提供服务
func main() {
	reader := bufio.NewReader(os.Stdin)
	sql, _ := reader.ReadString('\n')
	sql = strings.Replace(sql, "\n", "", -1)
	parsedSql, err := parser.Parse(sql)
	if err != nil {
		fmt.Println(err)
	} else {
		err = parser.Handle(parsedSql)
	}
	fmt.Println(parsedSql)
}
