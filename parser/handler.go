package parser

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// 表定义的存储结构
type TableDefine struct {
	ColumnName           string   // 列名
	DataType             DataType // 本列的数据类型
	DataLength           int      // 本列的数据长度
	NotNull              bool     // 是否存在非空约束
	Unique               bool     // 是否存在唯一约束
	PrimaryKey           bool     // 是否为主键
	ForeignKey           bool     // 是否为外键
	ForeignKeyTableName  string   // 外键参照表名
	ForeignKeyColumnName string   // 外键参照列名
}

// 索引的存储结构
type Index struct {
	Values     []string // 该行的值
	PrimaryKey string // 该行的主键
}

func Handle(sql Sql) (err error) {
	switch sql.Type {
	case CreateTable:
		err = handleCreateTable(sql)
		if err != nil {
			return err
		}
	case CreateView:
		err = handleCreateView(sql)
		if err != nil {
			return err
		}
	case CreateIndex:
		err = handleCreateIndex(sql)
		if err != nil {
			return err
		}
	case Insert:
		err = handleInsert(sql)
		if err != nil {
			return err
		}
	default:
		return nil
	}

	return nil
}

// 建表的处理器
func handleCreateTable(sql Sql) (err error) {
	// 创建表名对应的数据CSV文件
	createCsvFile(sql.Tables[0])
	// 创建表名对应的列定义CSV文件
	createCsvFile(sql.Tables[0] + "_def")

	// 打开文件名称对应的CSV文件
	file, err := os.OpenFile("./file/"+sql.Tables[0]+"_def.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}

	writer := csv.NewWriter(file)

	var tableDefineData [][]string
	var fieldDefineData []string
	for _, field := range sql.CreateFields {
		fieldDefineData = append(fieldDefineData, field.Name)
		fieldDefineData = append(fieldDefineData, DataTypeString[field.DataType])
		fieldDefineData = append(fieldDefineData, strconv.Itoa(field.DataLength))
		fieldDefineData = append(fieldDefineData, strconv.FormatBool(field.NotNull))
		fieldDefineData = append(fieldDefineData, strconv.FormatBool(field.Unique))
		fieldDefineData = append(fieldDefineData, strconv.FormatBool(field.PrimaryKey))
		fieldDefineData = append(fieldDefineData, strconv.FormatBool(field.ForeignKey))
		fieldDefineData = append(fieldDefineData, field.ForeignKeyReferenceTable)
		fieldDefineData = append(fieldDefineData, field.ForeignKeyReferenceField)
		// 把每一列的元数据放入表定义的元数据中
		tableDefineData = append(tableDefineData, fieldDefineData)
		fieldDefineData = []string{}
	}

	err = writer.WriteAll(tableDefineData)
	if err != nil {
		panic(err)
	}

	writer.Flush()
	defer file.Close()
	return nil
}

// 创建视图的处理器
func handleCreateView(sql Sql) (err error) {
	// 用视图名新建文件
	createTxtFile(sql.Tables[0])

	// 打开文件名称对应的txt文件
	file, err := os.OpenFile("./file/"+sql.Tables[0]+".txt", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.WriteString(sql.ViewSelect)
	if err != nil {
		panic(err)
	}

	err = writer.Flush()
	if err != nil {
		panic(err)
	}
	return nil
}

// 创建索引的处理器
func handleCreateIndex(sql Sql) (err error) {
	// 每个列一个CSV文件
	for index, name := range sql.Fields {
		// 该列没有定义升序还是降序就按升序存储
		if index >= len(sql.IndexArrangement)|| sql.IndexArrangement[index] == "ASC" {
			createCsvFile(sql.IndexName + "_" + sql.Tables[0] + "_idx_ASC_" + name)
		} else {
			createCsvFile(sql.IndexName + "_" + sql.Tables[0] + "_idx_DESC_" + name)
		}
	}

	return nil
}

func handleInsert(sql Sql) (err error) {
	fileName, err := getFileByName(sql.Tables[0] + ".csv")
	path := "./file/"
	if err != nil {
		panic(err)
	}
	if fileName == "" {
		return fmt.Errorf("at INSERT: unknown table name %s", sql.Tables[0])
	}
	tableDefFileName := strings.TrimSuffix(fileName, ".csv") + "_def.csv"
	tableDef, err := os.Open(path + tableDefFileName)
	if err != nil {
		panic(err)
	}
	defer tableDef.Close()
	table, err := os.OpenFile(path + fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	defer table.Close()
	tableDefineReader := csv.NewReader(tableDef)
	tableDefData, err := tableDefineReader.ReadAll()
	if err != nil {
		panic(err)
	}

	flag := false
	for _, field := range sql.Fields {
		for _, def := range tableDefData {
			if field == def[0] {
				flag = true
				break
			}
		}
		if flag == false {
			return fmt.Errorf("at SELECT: Unknown field name %s", field)
		}
		flag = false
	}

	table.Seek(0, io.SeekEnd)
	tableWriter := csv.NewWriter(table)
	tableWriter.WriteAll(sql.Inserts)
	tableWriter.Flush()

	return nil
}

//func handleSelect(sql Sql) (err error) {
//	fileName, err := getFileByName(sql.Tables[0])
//	if err != nil {
//		panic(err)
//	}
//	tableDefFileName := strings.TrimSuffix(fileName, ".csv") + "_def.csv"
//
//	tableDef, err := os.Open(tableDefFileName)
//	if err != nil {
//		panic(err)
//	}
//	table, err := os.Open(fileName)
//	if err != nil {
//		panic(err)
//	}
//}

//func handleWhere(data [][]string, sql Sql) (result [][]string) {}
