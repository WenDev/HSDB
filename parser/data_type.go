package parser

// 基本数据类型定义
type DataType int

const (
	// 未知数据类型
	UnknownDataType DataType = iota
	// 有符号32位整数类型，对应Go的int32
	SmallInt
	// 有符号64位浮点数类型，对应Go的float64
	Double
	// 日期和时间类型，以固定格式的字符串存储，格式为YYYY-MM-DD HH:MM:SS
	DateTime
	// 变长字符串类型，对应Go的string
	Varchar
)

var DataTypeString = []string{
	"UnknownDataType",
	"SMALLINT",
	"DOUBLE",
	"DATETIME",
	"VARCHAR",
}
