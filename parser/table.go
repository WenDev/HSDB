package parser

// 表结构的定义
type Table struct {
	// 表名
	TableName string
	// 列的集合
	Fields []Field
	// 列中数据（元组）的集合
	Records []Record
}

// 列的定义
type Field struct {
	Name                     string              // 列名
	DataType                 DataType            // 该列的数据类型
	DataLength               int                 // 该列数据的长度
	Constraint               []Constraint        // 列的约束条件——约束类型和具体的约束条件，是一个数组
	CheckConditions          []Condition         // Check约束的条件
	CheckConditionsOperator  []ConditionOperator // Check约束的连接运算符，只可能为And或者Or
	PrimaryKey               bool                // 是否为主键
	NotNull                  bool                // 是否有非空约束
	Unique                   bool                // 是否有唯一约束
	ForeignKey               bool                // 是否为外键
	ForeignKeyFlag           bool                // 当前正在定义这个字段的外键，一般为false，在Create Table的ForeignKey语句中使用
	ForeignKeyReferenceTable string              // 外键被参照表
	ForeignKeyReferenceField string              // 外键被参照列
}

// 元组的定义，用于返回
type Record struct {
	// 该元组对应的列
	Field Field
	// 具体数值，使用string数组类型存储
	Data []string
}

// 关系完整性约束
type Constraint struct {
	ConstraintType ConstraintType // 该约束的约束类型
}

type ConstraintType int

const (
	// 未知约束
	UnknownConstraint ConstraintType = iota
	// 该列不能为Null
	NotNull
	// 该列的每一行必须有唯一的值
	Unique
	// 主键，NotNull和Unique的结合
	PrimaryKey
	// 符合特定的条件，需要记录额外信息
	Check
	// 外键，也就是参照完整性约束，需要记录额外信息
	ForeignKey
	// 没有约束
	Default
)
