package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type TableJson struct {
	Name   string      `json:"name"`
	Fields []FieldJson `json:"fields"`
}

type FieldJson struct {
	Name             string   `json:"name"`
	DataType         DataType `json:"data_type"`
	DataLength       int      `json:"data_length"`
	NotNull          bool     `json:"not_null"`
	Unique           bool     `json:"unique"`
	PrimaryKey       bool     `json:"primary_key"`
	ForeignKey       bool     `json:"foreign_key"`
	ForeignKeyTable  string   `json:"foreign_key_table"`
	ForeignKeyColumn string   `json:"foreign_key_column"`
	Data             []string `json:"data"`
}

type IndexJson struct {
	Name  string           `json:"name"`
	Index []IndexValueJson `json:"index"`
}

// 索引的存储结构
type IndexValueJson struct {
	Value      string `json:"value"`
	PrimaryKey string `json:"primary_key"`
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
	// 创建表的JSON文件
	createJsonFile(sql.Tables[0])

	// 创建列定义的结构体数组
	var fields []FieldJson
	// 把每一个列都转换为一个对象，加入结构体数组
	for _, field := range sql.CreateFields {
		fields = append(fields, FieldJson{
			Name:             field.Name,
			DataType:         field.DataType,
			DataLength:       field.DataLength,
			NotNull:          field.NotNull,
			Unique:           field.Unique,
			PrimaryKey:       field.PrimaryKey,
			ForeignKey:       field.ForeignKey,
			ForeignKeyTable:  field.ForeignKeyReferenceTable,
			ForeignKeyColumn: field.ForeignKeyReferenceField,
			Data:             []string{},
		})
	}

	// 创建表定义的结构体
	table := TableJson{
		Name:   sql.Tables[0],
		Fields: fields,
	}

	tableJson, err := json.Marshal(table)
	if err != nil {
		panic(err)
	}

	// 生成JSON文件
	err = ioutil.WriteFile("./file/"+sql.Tables[0]+".json", tableJson, os.ModeAppend)
	if err != nil {
		panic(err)
	}

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
	// 每个列一个JSON文件
	for index, name := range sql.Fields {
		// 该列没有定义升序还是降序就按升序存储
		if index >= len(sql.IndexArrangement) || sql.IndexArrangement[index] == "ASC" {
			createJsonFile(sql.IndexName + "_" + sql.Tables[0] + "_idx_ASC_" + name)
		} else {
			createJsonFile(sql.IndexName + "_" + sql.Tables[0] + "_idx_DESC_" + name)
		}
	}

	return nil
}

func handleInsert(sql Sql) (err error) {
	fileName, err := getFileByName(sql.Tables[0] + ".json")
	path := "./file/"
	if err != nil {
		panic(err)
	}
	// 不存在这个名称的表文件，说明该表不存在
	if fileName == "" {
		return fmt.Errorf("at INSERT: unknown table name %s", sql.Tables[0])
	}
	// 读表文件内容
	bytes, err := ioutil.ReadFile(path+fileName)
	if err != nil {
		panic(err)
	}
	// 把表文件转换为结构体
	table := &TableJson{}
	err = json.Unmarshal(bytes, table)
	if err != nil {
		panic(err)
	}
	// 处理插入请求
	// 找到对应列名的数据，插入到对应的列中
	for index, insertFieldName := range sql.Fields {
		// 是否找到对应的列
		flag := false
		for tableIndex, tableField := range table.Fields {
			// 找到对应的列了，进行插入
			if insertFieldName == tableField.Name {
				flag = true
				// 把该行所有的数据都插入进去
				for _, insertValue := range sql.Inserts {
					table.Fields[tableIndex].Data = append(table.Fields[tableIndex].Data, insertValue[index])
				}
			}
		}
		if flag != true {
			return fmt.Errorf("at INSERT: unknown field %s in table %s", insertFieldName, table.Name)
		}
		flag = false
	}
	// 开始覆盖写入文件
	jsonTable, err := json.Marshal(table)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(path+fileName, jsonTable, os.ModeAppend)
	if err != nil {
		panic(err)
	}
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
