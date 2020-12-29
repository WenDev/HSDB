package parser

import (
	"io/ioutil"
	"os"
)

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
