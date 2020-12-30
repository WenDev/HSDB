package parser

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"unsafe"
)

// 处理帮助命令
func HandleHelp(help string) (err error) {
	s := strings.Split(help, " ")
	switch strings.ToUpper(s[1]) {
	case "DATABASE":
		err = handleHelpDataBase()
		if err != nil {
			return err
		} else {
			return nil
		}
	case "TABLE":
		err = handleHelpTable(help)
		if err != nil {
			return err
		} else {
			return nil
		}
	case "VIEW":
		err = handleHelpView(help)
		if err != nil {
			return err
		} else {
			return nil
		}
	case "INDEX":
		err = handleHelpIndex(help)
		if err != nil {
			return err
		} else {
			return nil
		}
	default:
		return fmt.Errorf("at HELP: unknown identifier %s", s[1])
	}
}

// help database命令的处理器
func handleHelpDataBase() (err error) {
	tables, indexes, views, err := getFilesForHelpDataBase()
	if err != nil {
		return err
	}
	// 表
	fmt.Println("Tables: ")
	for _, table := range tables {
		fmt.Print("- ")
		fmt.Println(strings.TrimSuffix(table, ".json"))
	}
	// 视图
	fmt.Println("Views: ")
	for _, view := range views {
		fmt.Print("- ")
		fmt.Println(strings.TrimSuffix(view, ".txt"))
	}
	// 索引
	fmt.Println("Indexes: ")
	for _, index := range indexes {
		indexInfo := strings.Split(index, "_")
		fmt.Printf("- Index: %s, Table: %s, Field: %s, Type: %s\n", indexInfo[0], indexInfo[1], strings.TrimSuffix(indexInfo[4], ".json"), indexInfo[3])
	}
	return nil
}

// help table命令的处理器
func handleHelpTable(help string) (err error) {
	s := strings.Split(help, " ")
	fileName, err := getFileByName(s[2] + ".json")
	path := "./file/"
	if err != nil {
		return err
	}
	// 不存在这个名称的表文件，说明该表不存在
	if fileName == "" {
		return fmt.Errorf("at HELP: unknown table name %s", s[2])
	}
	// 读表文件内容
	bytes, err := ioutil.ReadFile(path + fileName)
	if err != nil {
		return err
	}
	// 把表文件转换为结构体
	table := &TableJson{}
	err = json.Unmarshal(bytes, table)
	if err != nil {
		return err
	}
	fmt.Println("ColumnName\t|DataType\t|DataLength\t|NotNull\t|Unique\t|PrimaryKey\t|ForeignKey\t|ForeignKeyReferenceTable\t|ForeignKeyReferenceColumn\t")
	// 处理帮助命令
	for _, field := range table.Fields {
		fmt.Printf("%-10s\t|%-10s\t|%-10d\t|%-10s\t|%-10s\t|%-10s\t|%-10s\t|%-10s\t|%-10s\t\n",
			field.Name, DataTypeString[field.DataType], field.DataLength, strconv.FormatBool(field.NotNull), strconv.FormatBool(field.Unique),
			strconv.FormatBool(field.PrimaryKey), strconv.FormatBool(field.ForeignKey), field.ForeignKeyTable, field.ForeignKeyColumn)
	}
	fmt.Println()
	return nil
}

// help view命令的处理器
func handleHelpView(help string) (err error) {
	s := strings.Split(help, " ")
	fileName, err := getFileByName(s[2] + ".txt")
	path := "./file/"
	if err != nil {
		return err
	}
	// 不存在这个名称的表文件，说明该视图不存在
	if fileName == "" {
		return fmt.Errorf("at HELP: unknown view name %s", s[2])
	}
	// 读文件内容
	bytes, err := ioutil.ReadFile(path + fileName)
	if err != nil {
		return err
	}
	fmt.Println(*(*string)(unsafe.Pointer(&bytes)))
	return nil
}

// help index命令的处理器
func handleHelpIndex(help string) (err error) {
	s := strings.Split(help, " ")
	fileNames, err := getFilesByNameLike(s[2])
	if err != nil {
		return err
	}
	fmt.Printf("Index Name: %s\n", s[2])
	fmt.Println("Type: Unique")
	for _, fileName := range fileNames {
		name := strings.TrimSuffix(fileName, ".json")
		indexInfo := strings.Split(name, "_")
		fmt.Printf("Table: %s, Field: %s, Type: %s\n", indexInfo[1], indexInfo[4], indexInfo[3])
	}
	return nil
}
