package parser

import (
	"encoding/csv"
	"os"
	"strconv"
)

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

func Handle(sql Sql) (err error) {
	switch sql.Type {
	case CreateTable:
		err = handleCreateTable(sql)
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
	file, err := os.OpenFile("./file/" + sql.Tables[0] + "_def.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
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
