package parser

import (
	"io/ioutil"
	"os"
)

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

// 用文件名创建.json文件
func createJsonFile(fileName string) {
	fileDir := "./file"
	err := os.MkdirAll(fileDir, 0700)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(fileDir + "/" + fileName + ".json")
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

// 得到所有包含某个文件名的文件
func getFileByName(name string) (file string, err error) {
	dir, err := ioutil.ReadDir("./file")
	if err != nil {
		return "", err
	}
	for _, file := range dir {
		if name == file.Name() {
			return file.Name(), nil
		}
	}
	return "", err
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
