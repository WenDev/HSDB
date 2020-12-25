package parser

import (
	"encoding/csv"
	"os"
)

// 通过表名以读的方式打开一个CSV文件，适用于查询
func getReaderByName(fileName string) (reader *csv.Reader, error error) {
	// 打开文件名称对应的CSV文件
	file, err := os.Open(fileName + ".csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	return csv.NewReader(file), nil
}

// 通过表名以写的方式打开一个CSV文件，适用于增删改
func getWriterByName(fileName string) (*csv.Writer, error) {
	// 打开文件名称对应的CSV文件
	file, err := os.Open(fileName + ".csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	return csv.NewWriter(file), nil
}

// 根据文件名创建目录及新的CSV文件
func createCsvFile(fileName string) {
	fileDir := "./file"
	err := os.MkdirAll(fileDir, 0700)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(fileDir + "/" + fileName + ".csv")
	if err != nil {
		panic(err)
	}

	defer file.Close()
}

// 用给定的文件名新建txt文件
func createTxtFile(fileName string) {
	fileDir := "./file"
	err := os.MkdirAll(fileDir, 0700)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(fileDir + "/" + fileName + ".txt")
	if err != nil {
		panic(err)
	}

	defer file.Close()
}

// 增加记录
// 参数为要增加的记录（元组数组类型）
func addRecord(records []Record) {
	// TODO
}

// 删除记录
// 参数为删除条件的集合
func deleteRecord(conditions []Condition) {
	// TODO
}

// 修改符合指定条件记录的某个字段的值
// 参数为修改条件的集合与要修改的值（map类型，key为列名，value为要修改的值）
func updateRecord(conditions []Condition, records map[string]string) {
	// TODO
}

// 查找符合指定条件的记录
// 参数为查询条件的集合
func findRecord(conditions []Condition) {
	// TODO
}
