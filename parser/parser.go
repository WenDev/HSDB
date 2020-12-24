package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 解析完成的SQL
type Sql struct {
	Type               Type                // 该条SQL语句的类型
	Tables             []string            // 该条SQL语句操作的表名，因为要实现多表查询所以可能有多个
	Conditions         []Condition         // 查询条件：Where语句后的部分
	Updates            map[string]string   // 更新数据的Map
	Inserts            [][]string          // 插入的数据，如果不是Insert类型则为nil
	Fields             []string            // 受影响的列
	CreateFields       []Field             // 新建的列，如果不是CreateTable类型则为nil
	ConditionOperators []ConditionOperator // Where字句之间的连接符
	ViewSelect         string              // 创建视图时使用，为该视图定义的Select语句
}

// 查询条件
type Condition struct {
	Operand1        string   // 操作数1
	Operand2        string   // 操作数2
	Operator        Operator // 操作符
	Operand1IsField bool     // 操作数1是不是某一个列
	Operand2IsField bool     // 操作数2是不是某一个列
	IsBetween       bool     // 是否为Between-And语句，不是则BetweenOperand1和2都为nil
	IsNotBetween    bool     // 是否为Not Between-And语句
	BetweenOperand1 string   // Between子句操作数1
	BetweenOperand2 string   // Between字句操作数2
	IsIn            bool     // 是否为In语句
	IsNotIn         bool     // 是否为NotIn语句
	InConditions    []string // In语句的查询条件
}

// 该条SQL语句的类型
type Type int

const (
	// 未知的查询类型
	Unknown Type = iota
	// 增
	Insert
	// 删
	Delete
	// 改
	Update
	// 查
	Select
	CreateTable
	CreateView
	CreateIndex
	CreateUser
	// 对用户赋予权限
	Grant
	// 删除用户的权限
	Revoke
)

var TypeString = []string{
	"Unknown",
	"Insert",
	"Delete",
	"Update",
	"Select",
	"Create Table",
	"Create View",
	"Create Index",
	"Create User",
	"Grant",
	"Revoke",
}

// 操作符的类型
type Operator int

const (
	UnknownOperator Operator = iota // 未知操作符
	Eq                              // 相等：=
	Ne                              // 不相等：!=
	Gt                              // 大于： >
	Lt                              // 小于： <
	Gte                             // 大于等于：>=
	Lte                             // 小于等于：<=
	Between                         // Between - And子句
	NotBetween                      // Not Between - And 子句
	Like                            // 相似于Operand2
	NotLike                         // 不相似于Operand2
	In                              // 必须取值为Operand2的值
	NotIn                           // 不能是Operand2的值
)

var OperatorString = []string{
	"Unknown",
	"Eq",
	"Ne",
	"Gt",
	"Lt",
	"Gte",
	"Lte",
}

// Where、Check等子句的连接条件
type ConditionOperator int

const (
	// 未知的Where字句连接条件
	UnknownConditionOperator ConditionOperator = iota
	And
	Or
)

var WhereConditionString = []string{
	"Unknown",
	"And",
	"Or",
	"Between",
	"In",
	"Like",
}

// SQL语句中的合法字符，未出现在此处表示不合法
var legalWords = []string{
	"(",
	")",
	">=",
	"<=",
	"!=",
	",",
	"=",
	">",
	"<",
	"SELECT",
	"INSERT INTO",
	"VALUES",
	"UPDATE",
	"SET",
	"DELETE FROM",
	"CREATE TABLE",
	"CREATE VIEW",
	"CREATE INDEX",
	"CREATE USER",
	"CHECK",
	"WHERE",
	"FROM",
	"AND",
	"IN",
	"NOT IN",
	"LIKE",
	"NOT LIKE",
	"GROUP BY",
	"ORDER BY",
	"HAVING",
	"BETWEEN",
	"NOT BETWEEN",
	"IDENTIFIED BY",
	"ON TABLE",
	"TO",
	"GRANT",
	"REVOKE",
	"NOT NULL",
	"UNIQUE",
	"PRIMARY KEY",
	"FOREIGN KEY",
	"REFERENCES",
}

type parser struct {
	sql             string // 待解析的SQL语句，字符串类型
	position        int    // 当前所在查询字符串中的位置
	query           Sql    // 解析完成的查询结构体
	step            step   // 当前步骤
	err             error  // 解析过程中出现的错误
	nextUpdateField string // 下一个要更新的列
}

func Parse(sql string) (parsedSql Sql, err error) {
	qs, err := ParseMany([]string{sql})
	if len(qs) == 0 {
		return Sql{}, err
	}
	return qs[0], err
}

func ParseMany(sqls []string) (parsedSqls []Sql, err error) {
	var qs []Sql
	for _, sql := range sqls {
		q, err := parse(sql)
		if err != nil {
			return qs, err
		}
		qs = append(qs, q)
	}

	return qs, nil
}

func parse(sql string) (parsedSql Sql, err error) {
	return (&parser{
		sql:             strings.TrimSpace(sql),
		position:        0,
		query:           Sql{},
		step:            stepBeginning,
		err:             nil,
		nextUpdateField: "",
	}).parse()
}

// 返回一个查询结构体或一个错误
func (p *parser) parse() (parsedSql Sql, err error) {
	sql, err := p.doParse()
	p.err = err

	if err != nil {
		p.logError()
	}

	return sql, err
}

// 主解析函数
func (p *parser) doParse() (parsedSql Sql, err error) {
	for {
		if p.position >= len(p.sql) {
			return p.query, p.err
		}
		switch p.step {
		case stepBeginning:
			switch strings.ToUpper(p.peek()) {
			case "SELECT":
				p.query.Type = Select
				p.pop()
				p.step = stepSelectField
			case "INSERT INTO":
				p.query.Type = Insert
				p.pop()
				p.step = stepInsertTable
			case "UPDATE":
				p.query.Type = Update
				p.query.Updates = map[string]string{}
				p.pop()
				p.step = stepUpdateTable
			case "DELETE FROM":
				p.query.Type = Delete
				p.pop()
				p.step = stepDeleteFromTable
			case "CREATE TABLE":
				p.query.Type = CreateTable
				p.pop()
				p.step = stepCreateTableName
			case "CREATE VIEW":
				p.query.Type = CreateView
				p.pop()
				p.step = stepCreateViewName
			case "CREATE INDEX":
				p.query.Type = CreateIndex
				p.pop()
				p.step = stepCreateIndexName
			case "CREATE USER":
				p.query.Type = CreateUser
				p.pop()
				p.step = stepCreateUserName
			case "GRANT":
				p.query.Type = Grant
				p.pop()
				p.step = stepGrantPrivilege
			case "REVOKE":
				p.query.Type = Revoke
				p.pop()
				p.step = stepRevokePrivilege
			default:
				p.query.Type = Unknown
				return p.query, fmt.Errorf("unknown query type: %s", strings.ToUpper(p.peek()))
			}
		case stepCreateTableName:
			tableName := p.peek()
			// 表名不合法
			if !isIdentifierOrAsterisk(tableName) {
				return p.query, fmt.Errorf("at CREATE TABLE: expected a legal table name to CREATE")
			}
			// 把表名放入SQL查询的表名中
			p.query.Tables = append(p.query.Tables, tableName)
			p.pop()
			// 下一步：读建表的左括号
			p.step = stepCreateTableOpeningParens
		case stepCreateTableOpeningParens:
			openingParens := p.peek()
			// 读到的不是左括号
			if len(openingParens) != 1 || openingParens != "(" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected opening parens: '('")
			}
			p.pop()
			// 下一步：读列名
			p.step = stepCreateTableField
		case stepCreateTableField:
			fieldName := p.peek()
			// 列名不合法
			if !isIdentifierOrAsterisk(fieldName) {
				return p.query, fmt.Errorf("at CREATE TABLE: field name %s is illegal", fieldName)
			}
			// 列名合法，新建一个列放到最后
			p.query.CreateFields = append(p.query.CreateFields, Field{
				Name: fieldName,
			})
			p.pop()
			// 下一步：确定该列的类型
			p.step = stepCreateTableFieldType
		case stepCreateTableFieldType:
			fieldType := p.peek()
			// 取出最后一个放入创建列的数组中的列的地址进行修改，也就是当前还未创建完成的列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// 判断数值类型
			switch strings.ToUpper(fieldType) {
			case "SMALLINT":
				nowField.DataType = SmallInt
			case "DOUBLE":
				nowField.DataType = Double
			case "VARCHAR":
				nowField.DataType = Varchar
			case "DATETIME":
				nowField.DataType = DateTime
			default:
				nowField.DataType = UnknownDataType
				return p.query, fmt.Errorf("at CREATE TABLE: unknown data type %s", fieldType)
			}
			// 判断、写入完成，弹出这个数值类型，判断下一个是否是括号/逗号，不是括号或逗号则为约束类型，然后跳转相应操作
			p.pop()
			nextIdentifier := p.peek()
			switch nextIdentifier {
			case "(":
				// 左括号的下一步操作：读左括号
				p.step = stepCreateTableFieldOpeningParens
			case ",":
				// 逗号带下一步操作：读逗号
				p.step = stepCreateTableComma
			default:
				// 其他字符的下一步操作：确定约束类型
				p.step = stepCreateTableConstraintType
			}
		case stepCreateTableFieldOpeningParens:
			openingParens := p.peek()
			// 读到的不是左括号
			if openingParens != "(" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected opening parens: '('")
			}
			p.pop()
			// 下一步操作：设置该列的数值类型
			p.step = stepCreateTableFieldLength
		case stepCreateTableFieldLength:
			fieldLength := p.peek()
			// 不是整数
			fieldLengthInt, err := strconv.ParseInt(fieldLength, 10, 64)
			if err != nil {
				return p.query, fmt.Errorf("at CREATE TABLE: field length %s is not an integer", fieldLength)
			}
			// 是整数，给列赋值
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			nowField.DataLength = int(fieldLengthInt)
			p.pop()
			// 下一步操作：读右括号
			p.step = stepCreateTableFieldClosingParens
		case stepCreateTableFieldClosingParens:
			closingParens := p.peek()
			// 读到的不是右括号
			if closingParens != ")" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected closing parens: ')'")
			}
			// 读到的是右括号，弹出，读下一个
			p.pop()
			switch strings.ToUpper(p.peek()) {
			case ",":
				// 读到的是逗号，说明本列已经定义完成，开始定义下一列
				p.step = stepCreateTableComma
			case ")":
				// 读到的是右括号，说明全部的列已经定义完成，跳转到结束定义
				p.step = stepCreateTableClosingParens
			default:
				// 读到的是其他东西，则是该列的约束类型，跳转
				p.step = stepCreateTableConstraintType
			}
		case stepCreateTableConstraintType:
			// 读约束类型
			constraintType := p.peek()
			// 取出当前列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// 判断是什么约束
			switch strings.ToUpper(constraintType) {
			case "NOT NULL":
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: NotNull})
				nowField.NotNull = true
			case "UNIQUE":
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: Unique})
				nowField.Unique = true
			case "PRIMARY KEY":
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: PrimaryKey})
			case "CHECK":
			case "DEFAULT":
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: Default})
			default:
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: UnknownConstraint})
				return p.query, fmt.Errorf("at CREATE TABLE: unknown constraint type %s", constraintType)
			}
			if strings.ToUpper(constraintType) == "CHECK" {
				// Check约束需要确定Check条件，所以下一步跳转到Check条件
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: Check})
				p.step = stepCheck
			} else {
				// 约束判断完毕，弹出，判断下一个是什么
				p.pop()
				nextIdentifier := p.peek()
				switch nextIdentifier {
				case ",":
					// 本列已经定义完成，转下一列定义的逗号
					p.step = stepCreateTableComma
				case ")":
					// 本表已经定义完成，转表定义结束的右括号
					p.step = stepCreateTableClosingParens
				default:
					// 其他非法标识符
					return p.query, fmt.Errorf("at CREATE TABLE: unexpected token: %s", nextIdentifier)
				}
			}
		case stepCheck:
			check := p.peek()
			if strings.ToUpper(check) != "CHECK" {
				return p.query, fmt.Errorf("excepted CHECK")
			}
			p.pop()
			// 下一步：读Check左括号
			p.step = stepCheckOpeningParens
		case stepCheckOpeningParens:
			openingParens := p.peek()
			// 读到的不是左括号
			if openingParens != "(" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected opening parens: '('")
			}
			// 下一步：读Check约束需要检查的字段
			p.pop()
			p.step = stepCheckField
		case stepCheckField:
			field := p.peek()
			// 取出当前列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// 设置Check约束条件
			nowField.CheckConditions = append(nowField.CheckConditions, Condition{
				Operand1:        field,
				Operand1IsField: true,
			})
			// 下一步：读操作符
			p.pop()
			p.step = stepCheckOperator
		case stepCheckOperator:
			operator := p.peek()
			// 取出当前列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// 拿到当前操作的Check条件
			currentCondition := &nowField.CheckConditions[len(nowField.CheckConditions)-1]
			// 判断操作符
			switch operator {
			case "=":
				currentCondition.Operator = Eq
			case ">":
				currentCondition.Operator = Gt
			case ">=":
				currentCondition.Operator = Gte
			case "<":
				currentCondition.Operator = Lt
			case "<=":
				currentCondition.Operator = Lte
			case "!=":
				currentCondition.Operator = Ne
			default:
				switch strings.ToUpper(operator) {
				case "LIKE":
					// 读到的是Like
					currentCondition.Operator = Like
				case "NOT LIKE":
					// 读到的是Not Like
					currentCondition.Operator = NotLike
				case "IN":
					// 读到的是In
					currentCondition.Operator = In
					// In需要跳转到In约束条件
					p.step = stepCheckIn
				default:
					currentCondition.Operator = UnknownOperator
					return p.query, fmt.Errorf("at CHECK: unknown operator")
				}
			}
			if strings.ToUpper(operator) != "IN" {
				// 只要不是In，就只有一个需要检查的数值，跳转到对应的条件
				p.step = stepCheckValue
				p.pop()
			}
		case stepCheckValue:
			// 取得Check约束的检查值
			checkValue := p.peek()
			// 取出当前列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// 拿到当前操作的Check条件
			currentCondition := &nowField.CheckConditions[len(nowField.CheckConditions)-1]
			// 设置Check约束的值
			currentCondition.Operand2 = checkValue
			currentCondition.Operand2IsField = false
			// 赋值完毕，弹出这个值，判断下一个值
			p.pop()
			nextIdentifier := p.peek()
			switch strings.ToUpper(nextIdentifier) {
			case ")":
				// 读到左括号，跳转到左括号的条件
				p.step = stepCheckClosingParens
			case "AND":
				p.step = stepCheckAnd
			case "OR":
				p.step = stepCheckOr
			default:
				return p.query, fmt.Errorf("at CHECK: unexpected token %s", nextIdentifier)
			}
		case stepCheckIn:
			in := p.peek()
			// 读到的不是In
			if strings.ToUpper(in) != "IN" {
				return p.query, fmt.Errorf("at CHECK: expected IN")
			}
			// 取出当前列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// 拿到当前操作的Check条件，设置操作符为In
			currentCondition := &nowField.CheckConditions[len(nowField.CheckConditions)-1]
			currentCondition.Operator = In
			p.pop()
			// 下一步：读左括号
			p.step = stepCheckInOpeningParens
		case stepCheckInOpeningParens:
			openingParens := p.peek()
			// 读到的不是左括号
			if openingParens != "(" {
				return p.query, fmt.Errorf("at CHECK: expected opening parens '('")
			}
			p.pop()
			// 下一步：读In运算的值
			p.step = stepCheckInValue
		case stepCheckInValue:
			value := p.peek()
			// 取出当前列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// 拿到当前操作的Check条件，设置为In，并赋值
			currentCondition := &nowField.CheckConditions[len(nowField.CheckConditions)-1]
			currentCondition.IsIn = true
			currentCondition.InConditions = append(currentCondition.InConditions, value)
			p.pop()
			// 下一步：读右括号或逗号
			p.step = stepCheckInCommaOrClosingParens
		case stepCheckInCommaOrClosingParens:
			commaOrClosingParens := p.peek()
			// 如果读到的不是逗号或右括号
			if commaOrClosingParens != "," && commaOrClosingParens != ")" {
				return p.query, fmt.Errorf("at CHECK: expected comma ',' or closing parens ')'")
			}
			if commaOrClosingParens == "," {
				p.step = stepCheckInValue
				p.pop()
			}
			if commaOrClosingParens == ")" {
				// 读到左括号，表示In语句定义完毕，跳转到Check语句结束
				p.step = stepCheckClosingParens
				p.pop()
			}
		case stepCheckClosingParens:
			closingParens := p.peek()
			// 读到的不是右括号
			if closingParens != ")" {
				return p.query, fmt.Errorf("at CHECK: expected closing parens ')'")
			}
			p.pop()
			// Check字句定义结束，下一步：继续定义下一个列
			nextIdentifier := p.peek()
			if nextIdentifier == "," {
				p.step = stepCreateTableComma
			} else {
				p.step = stepCreateTableClosingParens
			}
		case stepCheckAnd:
			and := p.peek()
			// 读到的不是And
			if strings.ToUpper(and) != "AND" {
				return p.query, fmt.Errorf("at CHECK: expected AND")
			}
			// 取出当前列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// Check子句运算符运算条件设置为And
			nowField.CheckConditionsOperator = append(nowField.CheckConditionsOperator, And)
			p.pop()
			// 下一步：继续解析下一条Check子句
			p.step = stepCheckField
		case stepCheckOr:
			or := p.peek()
			// 读到的不是Or
			if strings.ToUpper(or) != "OR" {
				return p.query, fmt.Errorf("at CHECK: expected OR")
			}
			// 取出当前列
			nowField := &p.query.CreateFields[len(p.query.CreateFields)-1]
			// Check字句运算符运算条件设置为Or
			nowField.CheckConditionsOperator = append(nowField.CheckConditionsOperator, Or)
			p.pop()
			// 下一步：继续解析下一条Check子句
			p.step = stepCheckField
		case stepCreateTableClosingParens:
			// 表定义结束
			closingParens := p.peek()
			if closingParens != ")" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected closing parens: ')'")
			}
			p.pop()
			p.step = stepCreateTableField
		case stepCreateTableComma:
			// 读取字段定义完成的逗号
			comma := p.peek()
			// 读到的不是逗号
			if comma != "," {
				return p.query, fmt.Errorf("at CREATE TABLE: expected comma: ','")
			}
			p.pop()
			// 读取下一个标识符
			nextIdentifier := p.peek()
			switch strings.ToUpper(nextIdentifier) {
			case ")":
				// 是右括号，则表定义结束
				p.step = stepCreateTableClosingParens
			case "PRIMARY KEY":
				// 跳转主键约束
				p.step = stepPrimaryKey
			case "FOREIGN KEY":
				// 跳转外键约束
				p.step = stepForeignKey
			case "CHECK":
				// 跳转到Check约束
				p.step = stepCheck
			default:
				// 读到的是其他东西，则是下一个字段的字段名
				p.step = stepCreateTableField
			}
		case stepPrimaryKey:
			primaryKey := p.peek()
			// 读到的不是主键关键字
			if strings.ToUpper(primaryKey) != "PRIMARY KEY" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected PRIMARY KEY")
			}
			p.pop()
			// 下一步：读左括号
			p.step = stepPrimaryKeyOpeningParens
		case stepPrimaryKeyOpeningParens:
			openingParens := p.peek()
			if openingParens != "(" {
				return p.query, fmt.Errorf("at CHECK: expected opening parens '('")
			}
			p.pop()
			// 下一步：读主键字段
			p.step = stepPrimaryKeyField
		case stepPrimaryKeyField:
			fieldName := p.peek()
			flag := false // flag：是否找到名称相同的字段
			i := 0
			// 遍历已有的列名，判断是否存在名称相同的字段
			for index, field := range p.query.CreateFields {
				// 找到名称相同的字段，为其设置主键约束
				if field.Name == fieldName {
					i = index
					flag = true
				}
			}
			// 没有找到名称相同的列
			if flag == false {
				return p.query, fmt.Errorf("at CREATE TABLE: unknown field %s", fieldName)
			}
			field := &p.query.CreateFields[i]
			// 约束类型：主键
			field.Constraint = append(field.Constraint, Constraint{ConstraintType: PrimaryKey})
			// 设置该列为主键
			field.PrimaryKey = true
			p.pop()
			// 下一步：读逗号或右括号
			p.step = stepPrimaryKeyCommaOrClosingParens
		case stepPrimaryKeyCommaOrClosingParens:
			commaOrClosingParens := p.peek()
			// 读到的不是逗号或右括号
			if commaOrClosingParens != "," && commaOrClosingParens != ")" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected comma ',' or closing parens ')'")
			}
			if commaOrClosingParens == "," {
				// 读到逗号，说明有多个主键字段
				p.pop()
				p.step = stepPrimaryKeyField
			}
			if commaOrClosingParens == ")" {
				// 读到右括号，表示Primary Key约束定义完成
				p.pop()
				if p.peek() == "," {
					// 读到逗号，说明还有其他字段
					p.step = stepCreateTableComma
				} else {
					// 读到右括号，说明表的定义已经完成
					p.step = stepCreateTableClosingParens
				}
			}
		case stepForeignKey:
			foreignKey := p.peek()
			// 读到的不是外键关键字
			if strings.ToUpper(foreignKey) != "FOREIGN KEY" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected FOREIGN KEY")
			}
			p.pop()
			// 下一步：读左括号
			p.step = stepForeignKeyOpeningParens
		case stepForeignKeyOpeningParens:
			openingParens := p.peek()
			if openingParens != "(" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected opening parens '('")
			}
			p.pop()
			// 下一步：读参照字段
			p.step = stepForeignKeyField
		case stepForeignKeyField:
			fieldName := p.peek()
			flag := false // flag：是否找到名称相同的字段
			i := 0
			// 遍历已有的列名，判断是否存在名称相同的字段
			for index, field := range p.query.CreateFields {
				// 找到名称相同的字段，为其设置外键约束
				if field.Name == fieldName {
					i = index
					flag = true
				}
			}
			// 没有找到名称相同的列
			if flag == false {
				return p.query, fmt.Errorf("at CREATE TABLE: unknown field %s", fieldName)
			}
			field := &p.query.CreateFields[i]
			// 约束类型为外键
			field.Constraint = append(field.Constraint, Constraint{ConstraintType: ForeignKey})
			// 设置外键约束
			field.ForeignKey = true
			// 开始定义该外键
			field.ForeignKeyFlag = true
			p.pop()
			// 下一步：读右括号
			p.step = stepForeignKeyClosingParens
		case stepForeignKeyClosingParens:
			closingParens := p.peek()
			// 读到的不是右括号
			if closingParens != ")" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected closing parens ')'")
			}
			p.pop()
			// 下一步：读Reference关键字
			p.step = stepForeignKeyReference
		case stepForeignKeyReference:
			reference := p.peek()
			if strings.ToUpper(reference) != "REFERENCES" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected REFERENCES")
			}
			p.pop()
			p.step = stepForeignKeyReferenceTable
		case stepForeignKeyReferenceTable:
			// 读约束表名
			tableName := p.peek()
			i := 0
			for index, field := range p.query.CreateFields {
				// 拿到当前操作的字段
				if field.ForeignKeyFlag == true {
					i = index
				}
			}
			nowField := &p.query.CreateFields[i]
			nowField.ForeignKeyReferenceTable = tableName
			p.pop()
			// 下一步：读左括号
			p.step = stepForeignKeyReferenceFieldOpeningParens
		case stepForeignKeyReferenceFieldOpeningParens:
			openingParens := p.peek()
			if openingParens != "(" {
				return p.query, fmt.Errorf("at FOREIGN KEY: expected opening parens '('")
			}
			p.pop()
			// 下一步：读被参照字段
			p.step = stepForeignKeyReferenceField
		case stepForeignKeyReferenceField:
			// 读约束字段名
			fieldName := p.peek()
			i := 0
			for index, field := range p.query.CreateFields {
				// 拿到当前操作的字段
				if field.ForeignKeyFlag == true {
					i = index
				}
			}
			nowField := &p.query.CreateFields[i]
			nowField.ForeignKeyReferenceField = fieldName
			// 该字段定义完成
			nowField.ForeignKeyFlag = false
			p.pop()
			// 下一步：读右括号
			p.step = stepForeignKeyReferenceFieldClosingParens
		case stepForeignKeyReferenceFieldClosingParens:
			closingParens := p.peek()
			// 读到的不是右括号
			if closingParens != ")" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected closing parens ')'")
			}
			p.pop()
			// 根据读到的内容，判断下一步操作
			nextIdentifier := p.peek()
			switch nextIdentifier {
			case ",":
				// 读到逗号说明还有其他字段
				p.step = stepCreateTableComma
			case ")":
				// 读到右括号说明表定义已经结束
				p.step = stepCreateTableClosingParens
			default:
				return p.query, fmt.Errorf("at CREATE TABLE: unexpected token %s", nextIdentifier)
			}
		case stepSelectField:
			field := p.peek()
			if !isIdentifierOrAsterisk(field) {
				return p.query, fmt.Errorf("at SELECT: expected field to SELECT")
			}
			// 将读到的字段放入解析出的字段中
			p.query.Fields = append(p.query.Fields, field)
			p.pop()
			// 读下一个标识符，根据是否为FROM判断是否还有其他字段
			nextIdentifier := p.peek()
			if strings.ToUpper(nextIdentifier) == "FROM" {
				p.step = stepSelectFrom
				continue
			}
			p.step = stepSelectComma
		case stepSelectComma:
			comma := p.peek()
			// 读到的不是逗号
			if comma != "," {
				return p.query, fmt.Errorf("at SELECT: expected comma ',' or FROM")
			}
			p.pop()
			// 下一步：读下一个列
			p.step = stepSelectField
		case stepSelectFrom:
			from := p.peek()
			// 读到的不是FROM
			if strings.ToUpper(from) != "FROM" {
				return p.query, fmt.Errorf("at SELECT: expected FROM")
			}
			p.pop()
			// 下一步：读表名
			p.step = stepSelectFromTable
		case stepSelectFromTable:
			tableName := p.peek()
			if len(tableName) == 0 {
				return p.query, fmt.Errorf("at SELECT: expected quoted table name")
			}
			p.query.Tables = append(p.query.Tables, tableName)
			p.pop()
			nextIdentifier := p.peek()
			if nextIdentifier == "," {
				// 读到的是逗号，说明还没有读完，读逗号
				p.step = stepSelectFromTableComma
			} else {
				// 表名读取完毕，跳转到Where子句
				p.step = stepWhere
			}
		case stepSelectFromTableComma:
			comma := p.peek()
			// 读取到的不是逗号
			if comma != "," {
				return p.query, fmt.Errorf("at SELECT: expected comma or WHERE")
			}
			// 弹出这个逗号，开始读下一个表名
			p.pop()
			p.step = stepSelectFromTable
		case stepInsertTable:
			tableName := p.peek()
			// 如果读到的表名长度为0
			if len(tableName) == 0 {
				return p.query, fmt.Errorf("at INSERT INTO: expected a table name to INSERT")
			}
			p.query.Tables = append(p.query.Tables, tableName)
			p.pop()
			// 下一步：读左括号
			p.step = stepInsertFieldsOpeningParens
		case stepInsertFieldsOpeningParens:
			openingParens := p.peek()
			// 读到的不是左括号
			if len(openingParens) != 1 || openingParens != "(" {
				return p.query, fmt.Errorf("at INSERT INTO: expected opening parens '('")
			}
			p.pop()
			// 下一步：读需要插入的字段
			p.step = stepInsertFields
		case stepInsertFields:
			field := p.peek()
			// 读到的字段名不合法
			if !isIdentifier(field) {
				return p.query, fmt.Errorf("at INSERT INTO: expected at least one field to INSERT")
			}
			// 将这个字段加入字段数组中
			p.query.Fields = append(p.query.Fields, field)
			p.pop()
			// 下一步：读逗号或右括号
			p.step = stepInsertFieldsCommaOrClosingParens
		case stepInsertFieldsCommaOrClosingParens:
			commaOrClosingParens := p.peek()
			// 如果读到的不是逗号或右括号
			if commaOrClosingParens != "," && commaOrClosingParens != ")" {
				return p.query, fmt.Errorf("at INSERT INTO: expected comma or closing parens")
			}
			p.pop()
			// 根据读到的是逗号还是右括号判断接下来需要执行什么操作
			if commaOrClosingParens == "," {
				// 逗号，则继续读下一个字段
				p.step = stepInsertFields
			} else {
				// 右括号，则读VALUES关键字
				p.step = stepInsertValue
			}
		case stepInsertValue:
			values := p.peek()
			// 读到的不是VALUES
			if strings.ToUpper(values) != "VALUES" {
				return p.query, fmt.Errorf("at INSERT INTO: expected VALUES")
			}
			p.pop()
			// 下一步：读左括号
			p.step = stepInsertValuesOpeningParens
		case stepInsertValuesOpeningParens:
			openingParens := p.peek()
			// 读到的不是左括号
			if openingParens != "(" {
				return p.query, fmt.Errorf("at INSERT INTO: expected opening parens")
			}
			// 将需要插入的字段的数组初始化
			p.query.Inserts = append(p.query.Inserts, []string{})
			p.pop()
			// 下一步：读需要插入的数值
			p.step = stepInsertValues
		case stepInsertValues:
			value := p.peek()
			// 将读到的数值放入待插入的数组中
			p.query.Inserts[len(p.query.Inserts)-1] = append(p.query.Inserts[len(p.query.Inserts)-1], value)
			p.pop()
			// 下一步：读逗号或右括号
			p.step = stepInsertValuesCommaOrClosingParens
		case stepInsertValuesCommaOrClosingParens:
			commaOrClosingParens := p.peek()
			// 读到的不是逗号或右括号
			if commaOrClosingParens != "," && commaOrClosingParens != ")" {
				return p.query, fmt.Errorf("at INSERT INTO: expected comma or closing parens")
			}
			p.pop()
			// 读到的是逗号，说明还有值需要插入
			if commaOrClosingParens == "," {
				p.step = stepInsertValues
				continue
			}
			// 判断需要插入的字段数是否与给定的数值个数一致
			currentInsertRow := p.query.Inserts[len(p.query.Inserts)-1]
			if len(currentInsertRow) < len(p.query.Fields) {
				return p.query, fmt.Errorf("at INSERT INTO: value count doesn't match field count")
			}
			// 如果一致，说明Insert语句解析完毕
			p.step = stepInsertValue
		case stepUpdateTable:
			tableName := p.peek()
			// 如果读到的表名长度为0
			if len(tableName) == 0 {
				return p.query, fmt.Errorf("at UPDATE: expected a table name to UPDATE")
			}
			// 把表名添加到待更新的表中
			p.query.Tables = append(p.query.Tables, tableName)
			p.pop()
			// 下一步：读"SET"
			p.step = stepUpdateSet
		case stepUpdateSet:
			set := p.peek()
			// 如果读取到的不是SET
			if set != "SET" {
				return p.query, fmt.Errorf("at UPDATE: expected SET")
			}
			p.pop()
			// 下一步：读要更新的字段名
			p.step = stepUpdateField
		case stepUpdateField:
			field := p.peek()
			// 读到的字段名非法
			if !isIdentifier(field) {
				return p.query, fmt.Errorf("at UPDATE: expected at least one field to update")
			}
			p.nextUpdateField = field
			p.pop()
			// 下一步：读等号
			p.step = stepUpdateEqual
		case stepUpdateEqual:
			equal := p.peek()
			// 读到的不是等号
			if equal != "=" {
				return p.query, fmt.Errorf("at UPDATE: expected equal '='")
			}
			p.pop()
			// 下一步：读字段值
			p.step = stepUpdateValue
		case stepUpdateValue:
			value := p.peek()
			// 将字段值放入要更新的字段列表中
			p.query.Updates[p.nextUpdateField] = value
			p.nextUpdateField = ""
			p.pop()
			// 根据下一个标识符决定进行什么操作
			nextIdentifier := p.peek()
			// 读到的是where，跳转到Where子句解析
			if strings.ToUpper(nextIdentifier) == "WHERE" {
				p.step = stepWhere
				continue
			}
			// 读到的是逗号，说明还有其他要更新的字段
			p.step = stepUpdateComma
		case stepUpdateComma:
			comma := p.peek()
			// 读到的不是逗号
			if comma != "," {
				return p.query, fmt.Errorf("at UPDATE: expected comma ','")
			}
			p.pop()
			// 下一步： 读下一个字段名
			p.step = stepUpdateField
		case stepDeleteFromTable:
			tableName := p.peek()
			// 读到的要删除的表名长度为0
			if len(tableName) == 0 {
				return p.query, fmt.Errorf("at DELETE FROM: expected a table name to DELETE FROM")
			}
			p.query.Tables = append(p.query.Tables, tableName)
			p.pop()
			if p.peek() == "WHERE" {
				// 有WHERE，则下一步：读WHERE子句
				p.step = stepWhere
			}
		case stepWhere:
			where := p.peek()
			// 读到的不是Where
			if strings.ToUpper(where) != "WHERE" {
				return p.query, fmt.Errorf("expected WHERE")
			}
			p.pop()
			// 下一步：读取要被Where所判断的列
			p.step = stepWhereField
		case stepWhereField:
			field := p.peek()
			// 读到的列名不合法
			if !isIdentifier(field) {
				return p.query, fmt.Errorf("at WHERE: expected field")
			}
			p.query.Conditions = append(p.query.Conditions, Condition{Operand1: field, Operand1IsField: true})
			p.pop()
			// 下一步：读取Where子句的操作符
			p.step = stepWhereOperator
		case stepWhereOperator:
			operator := p.peek()
			currentCondition := &p.query.Conditions[len(p.query.Conditions)-1]
			switch operator {
			case "=":
				currentCondition.Operator = Eq
			case ">":
				currentCondition.Operator = Gt
			case ">=":
				currentCondition.Operator = Gte
			case "<":
				currentCondition.Operator = Lt
			case "<=":
				currentCondition.Operator = Lte
			case "!=":
				currentCondition.Operator = Ne
			case "LIKE":
				currentCondition.Operator = Like
			case "NOT LIKE":
				currentCondition.Operator = NotLike
			case "IN":
				currentCondition.Operator = In
				p.step = stepWhereIn
			case "NOT IN":
				currentCondition.Operator = NotIn
				p.step = stepWhereNotIn
			case "BETWEEN":
				currentCondition.Operator = Between
				p.step = stepWhereBetween
			case "NOT BETWEEN":
				currentCondition.Operator = NotBetween
				p.step = stepWhereNotBetween
			default:
				return p.query, fmt.Errorf("at WHERE: unknown operator")
			}
			if p.step != stepWhereBetween && p.step != stepWhereNotBetween && p.step != stepWhereIn && p.step != stepWhereNotIn {
				p.pop()
				p.step = stepWhereValue
			}
		case stepWhereValue:
			whereValue := p.peek()
			// 拿到当前操作的Where条件子句
			currentCondition := &p.query.Conditions[len(p.query.Conditions)-1]
			// 为当前的Where操作赋值
			currentCondition.Operand2 = whereValue
			currentCondition.Operand2IsField = false
			// 赋值完毕，弹出这个值，判断下一个值
			p.pop()
			nextIdentifier := p.peek()
			switch strings.ToUpper(nextIdentifier) {
			case "AND":
				p.step = stepWhereAnd
			case "OR":
				p.step = stepWhereOr
			case "IN":
				p.step = stepWhereIn
			case "NOT IN":
				p.step = stepWhereNotIn
			case "BETWEEN":
				p.step = stepWhereBetween
			}
		case stepWhereAnd:
			and := p.peek()
			// 读到的不是And
			if strings.ToUpper(and) != "AND" {
				return p.query, fmt.Errorf("expected AND")
			}
			// 放入一个And，表示Where的第一、二个子句之间的操作条件是And
			p.query.ConditionOperators = append(p.query.ConditionOperators, And)
			p.pop()
			// 下一步：读下一个要被操作的列
			p.step = stepWhereField
		case stepWhereOr:
			or := p.peek()
			// 读到的不是Or
			if strings.ToUpper(or) != "OR" {
				return p.query, fmt.Errorf("expected OR")
			}
			// 放入一个OR，表示Where的第一二个子句之间的操作条件为OR
			p.query.ConditionOperators = append(p.query.ConditionOperators, Or)
			p.pop()
			// 下一步：读取下一个要被操作的列
			p.step = stepWhereField
		case stepWhereIn:
			in := p.peek()
			// 读到的不是In
			if strings.ToUpper(in) != "IN" {
				return p.query, fmt.Errorf("at WHERE: expected IN")
			}
			// 获得当前正在操作的条件
			currentCondition := &p.query.Conditions[len(p.query.Conditions)-1]
			currentCondition.IsIn = true
			p.pop()
			// 下一步：读左括号
			p.step = stepWhereInOpeningParens
		case stepWhereNotIn:
			notIn := p.peek()
			// 读到的不是Not In
			if strings.ToUpper(notIn) != "NOT IN" {
				return p.query, fmt.Errorf("at WHERE: expected NOT IN")
			}
			// 获得当前正在操作的条件
			currentCondition := &p.query.Conditions[len(p.query.Conditions)-1]
			currentCondition.IsNotIn = true
			p.pop()
			// 下一步：读左括号
			p.step = stepWhereInOpeningParens
		case stepWhereInOpeningParens:
			openingParens := p.peek()
			// 读到的不是左括号
			if openingParens != "(" {
				return p.query, fmt.Errorf("at WHERE: expected opening parens '('")
			}
			p.pop()
			// 下一步：读具体数值
			p.step = stepWhereInValue
		case stepWhereInValue:
			value := p.peek()
			// 获得当前正在操作的条件
			currentCondition := &p.query.Conditions[len(p.query.Conditions)-1]
			// 将读取到的值追加到In操作符条件中
			currentCondition.InConditions = append(currentCondition.InConditions, value)
			p.pop()
			// 下一步：读逗号或右括号
			p.step = stepWhereInCommaOrClosingParens
		case stepWhereInCommaOrClosingParens:
			commaOrClosingParens := p.peek()
			// 如果读到的不是逗号或右括号
			if commaOrClosingParens != "," && commaOrClosingParens != ")" {
				return p.query, fmt.Errorf("at CHECK: expected comma ',' or closing parens ')'")
			}
			if commaOrClosingParens == "," {
				p.step = stepWhereInValue
				p.pop()
			}
			if commaOrClosingParens == ")" {
				// 读到左括号，表示In语句定义完毕，跳转到Where结束
				p.step = stepWhere
				p.pop()
			}
		case stepWhereBetween:
			between := p.peek()
			// 如果读到的不是between
			if between != "BETWEEN" {
				return p.query, fmt.Errorf("expected BETWEEN")
			}
			p.pop()
			// 下一步：读第一个操作数
			p.step = stepWhereBetweenValue
		case stepWhereNotBetween:
			notBetween := p.peek()
			// 如果读到的不是not between
			if notBetween != "NOT BETWEEN" {
				return p.query, fmt.Errorf("expected NOT BETWEEN")
			}
			// 拿到当前操作的Where条件子句
			currentCondition := &p.query.Conditions[len(p.query.Conditions)-1]
			// 是一个Not-Between语句
			currentCondition.IsNotBetween = true
			p.pop()
			// 下一步：读第一个操作数
			p.step = stepWhereBetweenValue
		case stepWhereBetweenValue:
			value := p.peek()
			// 拿到当前操作的Where条件子句
			currentCondition := &p.query.Conditions[len(p.query.Conditions)-1]
			// 设置具体数值：Between与And之间是操作数1
			currentCondition.Operand1 = value
			// Between-And中肯定不会出现列名
			currentCondition.Operand1IsField = false
			p.pop()
			// 下一步：读AND
			p.step = stepWhereBetweenAnd
		case stepWhereBetweenAnd:
			and := p.peek()
			// 如果读到的不是AND
			if and != "AND" {
				return p.query, fmt.Errorf("expected AND")
			}
			p.pop()
			// 下一步：读第二个操作数
			p.step = stepWhereBetweenAndValue
		case stepWhereBetweenAndValue:
			value := p.peek()
			// 拿到当前操作的Where条件子句
			currentCondition := &p.query.Conditions[len(p.query.Conditions)-1]
			// 设置具体数值：And之后是操作数2
			currentCondition.Operand2 = value
			// Between-And中肯定不会出现列名
			currentCondition.Operand2IsField = false
			p.pop()
			// Between-And语句处理完成，返回
			p.step = stepWhere
		case stepCreateViewName:
			name := p.peek()
			if !isIdentifierOrAsterisk(name) {
				return p.query, fmt.Errorf("at CREATE VIEW: expected view name to CREATE")
			}
			p.query.Tables = append(p.query.Tables, name)
			p.pop()
			p.step = stepCreateViewOpeningParens
		case stepCreateViewOpeningParens:
			openingParens := p.peek()
			if openingParens != "(" {
				return p.query, fmt.Errorf("at CREATE VIEW: expected opening parens '('")
			}
			p.pop()
			p.step = stepCreateViewField
		case stepCreateViewField:
			field := p.peek()
			if !isIdentifierOrAsterisk(field) {
				return p.query, fmt.Errorf("at CREATE VIEW: expected field name to CREATE")
			}
			p.query.Fields = append(p.query.Fields, field)
			p.pop()
			p.step = stepCreateViewCommaOrClosingParens
		case stepCreateViewCommaOrClosingParens:
			commaOrClosingParens := p.peek()
			if commaOrClosingParens != "," && commaOrClosingParens != ")" {
				return p.query, fmt.Errorf("at CREATE VIEW: expected comma ',' or closing parens ')'")
			}
			p.pop()
			if commaOrClosingParens == "," {
				p.step = stepCreateViewField
			}
			if commaOrClosingParens == ")" {
				p.step = stepCreateViewAs
			}
		case stepCreateViewAs:
			as := p.peek()
			if strings.ToUpper(as) != "AS" {
				return p.query, fmt.Errorf("at CREATE VIEW: expected AS")
			}
			p.pop()
			p.step = stepCreateViewSelect
		case stepCreateViewSelect:
			selectSql := p.peekToEnd()
			p.query.ViewSelect = selectSql
			p.popToEnd()
			p.step = stepCreateViewName
		}
	}
}

// 检验生成的SQL语句是否合法
func (p *parser) validate() error {
	// WHERE语句的条件为空
	if len(p.query.Conditions) == 0 && p.step == stepWhereField {
		return fmt.Errorf("at WHERE: empty WHERE clause")
	}

	// 查询类型不正确
	if p.query.Type == Unknown {
		return fmt.Errorf("query type cannot be empty")
	}

	// 需要操作的表名中有至少一个为空
	for _, name := range p.query.Tables {
		if name == "" {
			return fmt.Errorf("table name cannot be empty")
		}
	}

	// 更新和删除语句没有WHERE子句
	if len(p.query.Conditions) == 0 && (p.query.Type == Update || p.query.Type == Delete) {
		return fmt.Errorf("at WHERE: WHERE clause is mandatory for UPDATE & DELETE")
	}

	// WHERE字句后的表达式不正确
	for _, c := range p.query.Conditions {
		// 操作符未知
		if c.Operator == UnknownOperator {
			return fmt.Errorf("at WHERE: condition without operator")
		}

		// 缺少操作数1
		if c.Operand1 == "" && c.Operand1IsField {
			return fmt.Errorf("at WHERE: condition with empty left side operand")
		}

		// 缺少操作数2
		if c.Operand2 == "" && c.Operand2IsField {
			return fmt.Errorf("at WHERE: condition with empty right side operand")
		}
	}

	// INSERT语句缺少要插入的数据
	if p.query.Type == Insert && len(p.query.Inserts) == 0 {
		return fmt.Errorf("at INSERT INTO: need at least one row to insert")
	}

	// INSERT语句要插入的数据与列数不匹配
	if p.query.Type == Insert {
		for _, i := range p.query.Inserts {
			if len(i) != len(p.query.Fields) {
				return fmt.Errorf("at INSERT INTO: value count doesn't match field count")
			}
		}
	}

	return nil
}

// 返回但不弹出解析的下一个记号
func (p *parser) peek() (peeked string) {
	// 返回下一个记号（这里不需要长度，pop才需要）
	peeked, _ = p.peekWithLength()
	return peeked
}

// 弹出解析的下一个记号
func (p *parser) pop() (peeked string) {
	// 得到下一个记号，并把当前解析位置移动到下一个记号后
	peeked, length := p.peekWithLength()
	p.position += length

	// 把空格全部弹出
	p.popWhitespace()

	return peeked
}

// pop到最后，用于创建视图
func (p *parser) popToEnd() {
	p.position += len(p.peekToEnd())
}

// 弹出所有空格
func (p *parser) popWhitespace() {
	for ; p.position < len(p.sql) && p.sql[p.position] == ' '; p.position++ {
	}
}

// 返回读到的字句及其长度
func (p *parser) peekWithLength() (string, int) {
	// 读到末尾
	if p.position >= len(p.sql) {
		return "", 0
	}

	// 合法字符
	for _, lw := range legalWords {
		token := strings.ToUpper(p.sql[p.position:min(len(p.sql), p.position+len(lw))])
		if token == lw {
			return token, len(token)
		}
	}

	// 有单引号的字句
	if p.sql[p.position] == '\'' {
		return p.peekQuotedStringWithLength()
	}

	// 其他子句
	return p.peekIdentifierWithLength()
}

// 返回读到的子句及其长度（针对有单引号的子句）
func (p *parser) peekQuotedStringWithLength() (identifier string, length int) {
	if len(p.sql) < p.position || p.sql[p.position] != '\'' {
		return "", 0
	}

	for i := p.position + 1; i < len(p.sql); i++ {
		// 如果读到分号，并且前一个符号不是转义字符
		if p.sql[i] == '\'' && p.sql[i-1] != '\\' {
			return p.sql[p.position+1 : i], len(p.sql[p.position+1:i]) + 2 // 因为有两个分号，所以长度要加2
		}
	}

	return "", 0
}

// 返回读到的子句及其长度
func (p *parser) peekIdentifierWithLength() (identifier string, length int) {
	for i := p.position; i < len(p.sql); i++ {
		// 不在语句的最后
		if matched, _ := regexp.MatchString(`[a-zA-Z0-9_*]`, string(p.sql[i])); !matched {
			return p.sql[p.position:i], len(p.sql[p.position:i])
		}
	}

	// 在语句的最后
	return p.sql[p.position:], len(p.sql[p.position:])
}

// 用于视图创建，直接返回当前位置到末尾的语句
func (p *parser) peekToEnd() (identifier string) {
	return p.sql[p.position:]
}

// 检测读到的标识符是否合法，或者为星号
func isIdentifierOrAsterisk(s string) bool {
	return isIdentifier(s) || s == "*"
}

// 检测读到的是否合法
func isIdentifier(s string) (result bool) {
	for _, lw := range legalWords {
		if strings.ToUpper(s) == lw {
			return false
		}
	}

	matched, _ := regexp.MatchString("[a-zA-Z_][a-zA-Z_0-9]*", s)
	return matched
}

// 打印错误信息
func (p *parser) logError() {
	// 打印错误的SQL语句和错误原因
	fmt.Println(p.sql)
	fmt.Println(strings.Repeat(" ", p.position) + "^")
	fmt.Println(p.err)
}

// 返回两个数中较小的一个
func min(a, b int) (min int) {
	if a < b {
		return a
	} else {
		return b
	}
}

// 判断一个字符串是否是浮点数
func IsNum(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
