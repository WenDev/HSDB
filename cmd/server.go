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
	fmt.Println("HSDB: A Simple DBMS")
	fmt.Println("====================")

	for {
		fmt.Printf("->")
		sql, _ := reader.ReadString('\n')
		sql = strings.Replace(sql, "\n", "", -1)
		if sql == "" {
			continue
		}
		s := strings.Split(sql, " ")
		if strings.ToUpper(s[0]) == "HELP" {
			err := parser.HandleHelp(sql)
			if err != nil {
				fmt.Println(err)
			}
			continue
		}
		parsedSql, err := parser.Parse(sql)
		if err != nil {
			fmt.Println(err)
		} else {
			result, rows, err := parser.Handle(parsedSql)
			if err != nil {
				fmt.Println(err)
			}
			if parsedSql.Type == parser.Select {
				fmt.Println("Result: ")
				for _, record := range result {
					fmt.Printf("%-10s|", record.Field.Name)
					for _, data := range record.Data {
						fmt.Printf("%-10s\t|", data)
					}
					fmt.Println()
				}
				fmt.Printf("\n")
			} else {
				fmt.Printf("OK, %d rows changed\n", rows)
			}
		}
	}
}
