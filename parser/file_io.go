package parser

import (
	"io/ioutil"
	"os"
	"strings"
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

// 查找包含某个名称的文件，用于查看已有索引
func getFilesByNameLike(name string) (files []string, err error) {
	dir, err := ioutil.ReadDir("./file")
	if err != nil {
		return nil, err
	}
	for _, file := range dir {
		if strings.Contains(file.Name(), name) {
			files = append(files, file.Name())
		}
	}
	return files, nil
}

// 将文件分类，用于help database命令
func getFilesForHelpDataBase() (tables []string, indexes []string, views []string, err error) {
	dir, err := ioutil.ReadDir("./file")
	if err != nil {
		return nil, nil, nil, err
	}
	for _, file := range dir {
		// users.json是存储用户和权限的文件，不需要处理
		if file.Name() == "users.json" {
			continue
		}
		// txt文件是视图文件
		if strings.Contains(file.Name(), ".txt") {
			views = append(views, file.Name())
		}
		// 含有idx的是索引
		if strings.Contains(file.Name(), "idx") {
			indexes = append(indexes, file.Name())
		}
		// 不含有idx的json文件是表
		if strings.Contains(file.Name(), ".json") && !strings.Contains(file.Name(), "idx") {
			tables = append(tables, file.Name())
		}
	}
	// 没有错误，返回
	return tables, indexes, views, nil
}
