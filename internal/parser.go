package internal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 解析完成的SQL
type Sql struct {
	Type         Type              // 该条SQL语句的类型
	Tables       []string          // 该条SQL语句操作的表名，因为要实现多表查询所以可能有多个
	Conditions   []Condition       // 查询条件：Where语句后的部分
	Updates      map[string]string // 更新数据的Map
	Inserts      [][]string        // 插入的数据
	Fields       []string          // 受影响的列
	CreateFields []Field           // 新建的列，如果不是CreateTable类型则为nil
}

// 查询条件
type Condition struct {
	Operand1        string   // 操作数1
	Operand2        string   // 操作数2
	Operator        Operator // 操作符
	Operand1IsField bool     // 操作数1是不是某一个列
	Operand2IsField bool     // 操作数2是不是某一个列
	IsBetween       bool     // 是否为Between-And语句，不是则BetweenOperand1和2都为nil
	BetweenOperand1 string   // Between子句操作数1
	BetweenOperand2 string   // Between字句操作数2
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
	Between                         // Between - And字句
	Like                            // 相似于Operand2
	NotLike                         // 不相似于Operand2
	In                              // 必须取值为Operand2的值
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

// Where字句的连接条件
type WhereCondition int

const (
	// 未知的Where字句连接条件
	UnknownWhereCondition = iota
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
	"LIKE",
	"NOT LIKE",
	"GROUP BY",
	"ORDER BY",
	"HAVING",
	"BETWEEN",
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

// 返回一个查询结构体或一个错误
func (p *parser) Parse() (parsedSql Sql, err error) {}

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
			case "INT":
				nowField.DataType = Int
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
				p.step = stepCreateTableField
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
			case "UNIQUE":
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: Unique})
			case "PRIMARY KEY":
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: PrimaryKey})
			case "CHECK":
				// Check约束需要确定Check条件，所以下一步跳转到Check条件
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: Check})
				p.step = stepCheck
			case "DEFAULT":
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: Default})
			default:
				nowField.Constraint = append(nowField.Constraint, Constraint{ConstraintType: UnknownConstraint})
				return p.query, fmt.Errorf("at CREATE TABLE: unknown constraint type %s", constraintType)
			}
			// 约束判断完毕，弹出，判断下一个是什么
			p.pop()
			nextIdentifier := p.peek()
			switch nextIdentifier {
			case ",":
				// 本列已经定义完成，转下一列定义的逗号
				// Check约束需要跳转到Check，所以这里需要不是Check约束才可以
				if strings.ToUpper(constraintType) != "CHECK" {
					p.step = stepCreateTableComma
				}
			case ")":
				// 本表已经定义完成，转表定义结束的右括号
				p.step = stepCreateTableClosingParens
			default:
				// 其他非法标识符
				return p.query, fmt.Errorf("at CREATE TABLE: unexpected token: %s", nextIdentifier)
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
			// 标识符不合法
			if field != nowField.Name {
				return p.query, fmt.Errorf("at CREATE TABLE -> CHECK: Check field %s does not match field %s", field, nowField.Name)
			}
			// 标识符合法，设置Check约束条件
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
			currentCondition := &nowField.CheckConditions[len(nowField.CheckConditions) - 1]
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
					return p.query, fmt.Errorf("at WHERE: unknown operator")
				}
			}
			p.pop()
			if strings.ToUpper(operator) != "IN" {
				// 只要不是In，就只有一个需要检查的数值，跳转到对应的条件
				p.step = stepCheckValue
			}
		case stepCheckValue:
		case stepCheckIn:

		case stepCreateTableClosingParens:
			// 表定义结束
			closingParens := p.peek()
			if closingParens != ")" {
				return p.query, fmt.Errorf("at CREATE TABLE: expected closing parens: ')'")
			}
			p.pop()
			p.step = stepCreateTableField
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

// 检测读到的标识符是否合法，或者为星号
func isIdentifierOrAsterisk(s string) bool {
	return isIdentifier(s) || s == "*"
}

// 检测读到的标识符是否合法，若为保留字（LegalWords，则不合法）
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
	// 没有错误，不需要打印
	if p.err == nil {
		return
	}

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
