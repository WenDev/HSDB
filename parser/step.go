package parser

// 当前的解析步骤
// 在我们解析SQL语句的过程中，只有少数几个记号是合法的。
// 当我们找到这样的合法的记号之后，就又到了一个新节点，此时又有另一些记号是合法的。
// 如此往复，直到完成整个SQL语句的解析过程。
// 以"UPDATE Student SET Sage = '22' WHERE Sno = '201215121'"语句为例，节点之间的转换可以用一张表来表示：
// STEP              | ON        | TRANSITION
// ------------------+-----------+------------------
// stepBeginning     | "UPDATE"  | stepUpdateTable
// stepUpdateTable   | 'Student' | stepUpdateSet
// stepUpdateSet     | "SET"     | stepUpdateField
// stepUpdateField   | 'Sage'    | stepUpdateEqual
// stepUpdateEqual   | "="       | stepUpdateValue
// stepUpdateValue   | '22'      | stepUpdateComma
// stepUpdateComma   | ","       | stepUpdateField
// stepUpdateComma   | "WHERE"   | stepWhereField
// stepWhereField    | 'Sno'     | stepWhereOperator
// stepWhereOperator | "="       | stepWhereAnd
// stepWhereAnd      | "AND"     | stepWhereField
// 将这个表转换为一个巨大的Switch语句，就是下面的doParse函数。
type step int

const (
	stepBeginning                             step = iota // "SELECT" / "UPDATE"
	stepSelectField                                       // 'Sno' => stepSelectComma(多字段) / stepSelectFrom(单字段)
	stepSelectComma                                       // "," => stepSelectField
	stepSelectFrom                                        // "FROM" => stepSelectFromTable
	stepSelectFromTable                                   // 'Student' => stepSelectFromTableComma(多表) / stepWhere(单表)
	stepSelectFromTableComma                              // "," => stepSelectFromTable
	stepSelectGroupBy                                     // "GROUP BY" => TODO GROUP BY状态实现
	stepSelectHaving                                      // "HAVING" => TODO HAVING状态实现
	stepSelectOrderBy                                     // "ORDER BY" => TODO ORDER BY状态实现
	stepInsertTable                                       // 'SC' => stepInsertFieldsOpeningParens
	stepInsertFieldsOpeningParens                         // "(" => stepInsertFields
	stepInsertFields                                      // 'Sno' => stepInsertFieldsCommaOrClosingParens
	stepInsertFieldsCommaOrClosingParens                  // "," / ")" => stepInsertFields(多字段) / stepInsertValuesRWord(单字段)
	stepInsertValues                                      // "VALUES" => stepInsertValuesOpeningParens
	stepInsertValuesOpeningParens                         // "(" => stepInsertValues
	stepInsertValue                                       // '201215128' => stepInsertValuesCommaOrClosingParens
	stepInsertValuesCommaOrClosingParens                  // "," / ")" => stepInsertValues(多字段) / stepInsertFieldsOpeningParens(单字段)
	stepUpdateTable                                       // 'Student' => stepUpdateSet
	stepUpdateSet                                         // "SET" => stepUpdateField
	stepUpdateField                                       // 'Sage' => stepUpdateEqual
	stepUpdateEqual                                       // "=" => stepUpdateValue
	stepUpdateValue                                       // '22' => stepUpdateComma
	stepUpdateComma                                       // "," / "WHERE" => stepUpdateField
	stepDeleteFromTable                                   // 'Student' => stepWhere
	stepWhere                                             // "WHERE" => stepWhereField
	stepWhereField                                        // 'Sdept' => stepWhereOperator
	stepWhereOperator                                     // "=" => stepWhereValue
	stepWhereValue                                        // 'CS' => stepWhereAnd
	stepWhereAnd                                          // "AND" => stepWhereField
	stepWhereOr                                           // "OR" => stepWhereField
	stepWhereBetween                                      // "BETWEEN" => stepWhereBetweenValue
	stepWhereNotBetween                                   // "NOT BETWEEN" => stepWhereBetweenValue
	stepWhereBetweenValue                                 // '100' => stepWhereBetweenAnd
	stepWhereBetweenAnd                                   // "AND"(after a value) => stepWhereBetweenAndValue
	stepWhereBetweenAndValue                              // '200' => stepWhereAnd / Or//
	stepWhereIn                                           // "IN" => stepWhereInOpeningParens
	stepWhereNotIn                                        // "NOT IN" => stepWhereInOpeningParens
	stepWhereInOpeningParens                              // "(" => stepWhereInValue
	stepWhereInValue                                      // 'CS' => stepWhereInCommaOrClosingParens
	stepWhereInCommaOrClosingParens                       // ",", ")" => stepWhereInValue / and or等
	stepCreateTableName                                   // 'Student' => stepCreateTableOpeningParens
	stepCreateTableOpeningParens                          // "(" => stepCreateTableField
	stepCreateTableField                                  // 'Sno' => stepCreateTableFieldType
	stepCreateTableFieldType                              // "CHAR" => stepCreateTableFieldOpeningParens(有长度) / stepCreateTableComma(无长度) / 约束
	stepCreateTableFieldOpeningParens                     // "(" => stepCreateTableFieldLength
	stepCreateTableFieldLength                            // '9' => stepCreateTableFieldClosingParens
	stepCreateTableFieldClosingParens                     // ")" => stepCreateTableComma / stepCreateTableClosingParens / stepCreateTableConstraintType
	stepCreateTableComma                                  // "," => stepCreateTableField(多字段) / stepCreateTableClosingParens(单字段) / 主键、外键约束
	stepCreateTableConstraintType                         // "NOT NULL" => stepCreateTableComma / stepCheck(约束类型为Check) / stepCreateTableClosingParens
	stepCreateTableClosingParens                          // ")" => stepCreateTableOpeningParens
	stepCheck                                             // "CHECK" => stepCheckOpeningParens
	stepCheckOpeningParens                                // "(" => stepCheckField
	stepCheckField                                        // 'Grade' => stepCheckOperator
	stepCheckOperator                                     // '>=' => stepCheckValue
	stepCheckValue                                        // '0' => stepCheckClosingParens / stepCheckAnd / Or
	stepCheckClosingParens                                // ")" => stepCreateTableComma
	stepCheckAnd                                          // "AND" => stepCheckField
	stepCheckOr                                           // "OR" => stepCheckField
	stepCheckIn                                           // "IN" => stepCheckInOpeningParens
	stepCheckInOpeningParens                              // "(" => stepCheckInValue
	stepCheckInValue                                      // '男' => stepCheckInCommaOrClosingParens
	stepCheckInCommaOrClosingParens                       // "," / ")" => stepCheckInValue / stepCheckClosingParens
	stepPrimaryKey                                        // "PRIMARY KEY" => stepPrimaryKeyOpeningParens
	stepPrimaryKeyOpeningParens                           // "(" => stepPrimaryKeyField
	stepPrimaryKeyField                                   // 'Sno' => stepPrimaryKeyCommaOrClosingParens
	stepPrimaryKeyCommaOrClosingParens                    // "," / ")" => stepPrimaryKeyField(多字段) / stepCreateTableComma(单字段)
	stepForeignKey                                        // "FOREIGN KEY" => stepForeignKeyOpeningParens
	stepForeignKeyOpeningParens                           // "(" => stepForeignKeyField
	stepForeignKeyField                                   // 'Cpno' => stepForeignKeyClosingParens
	stepForeignKeyClosingParens                           // ")" => stepForeignKeyReference
	stepForeignKeyReference                               // "REFERENCES" => stepForeignKeyReferenceTable
	stepForeignKeyReferenceTable                          // 'Course' => stepForeignKeyReferenceFieldOpeningParens
	stepForeignKeyReferenceFieldOpeningParens             // "(" => stepForeignKeyReferenceField
	stepForeignKeyReferenceField                          // 'Cno' => stepForeignKeyReferenceFieldClosingParens
	stepForeignKeyReferenceFieldClosingParens             // ")" => stepCreateTableComma / stepCreateTableClosingParens
	stepCreateViewName                                    // 'IS_STUDENT' => stepCreateViewOpeningParens(有列名) / stepCreateViewAs(无列名)
	stepCreateViewOpeningParens                           // "(" => stepCreateViewField
	stepCreateViewField                                   // 'Sno' => stepCreateViewCommaOrClosingParens
	stepCreateViewCommaOrClosingParens                    // "," / ")" => stepCreateViewField(多字段) / stepCreateViewAs(单字段)
	stepCreateViewAs                                      // "AS" => stepCreateViewSelect
	stepCreateViewSelect                                  // "SELECT" => stepCreateView。注：整个SELECT语句存入文件
	stepCreateIndexName                                   // 'index_name' => stepCreateIndexOn
	stepCreateIndexOn                                     // "ON" => stepCreateIndexTableName
	stepCreateIndexTableName                              // 'table_name' => stepCreateIndexOpeningParens
	stepCreateIndexOpeningParens                          // "(" => stepCreateIndexField
	stepCreateIndexField                                  // 'column_name' => stepCreateIndexCommaOrClosingParens
	stepCreateIndexAscOrDesc                              // "ASC" / "DESC" => stepCreateIndexCommaOrClosingParens
	stepCreateIndexCommaOrClosingParens                   // ")", "," => stepCreateIndex(单字段) / stepCreateIndexField(多字段)
	stepCreateUserName                                    // 'username' => stepCreateUserIdentifiedBy
	stepCreateUserIdentifiedBy                            // "IDENTIFIED BY" => stepCreateUserPassword
	stepCreateUserPassword                                // 'password' => stepCreateUser
	stepGrantPrivilege                                    // "SELECT" => stepGrantPrivilege(多权限) / stepGrantOnTable(单权限) / stepGrantPrivilegeOpeningParens(有括号)
	stepGrantPrivilegeOpeningParens                       // "(" => stepGrantPrivilegeField
	stepGrantPrivilegeField                               // 'Sno' => stepGrantPrivilegeCommaOrClosingParens
	stepGrantPrivilegeCommaOrClosingParens                // "," / ")" => stepGrantPrivilegeComma(多字段) / stepGrantOnTable(单字段)
	stepGrantOnTable                                      // "ON TABLE" => stepGrantTableName
	stepGrantTableName                                    // 'Student' => stepGrantTableComma / stepGrantTo
	stepGrantTableComma                                   // "," => stepGrantTableName
	stepGrantTo                                           // "TO" => stepGrantUserName
	stepGrantUserName                                     // 'U1' => stepGrantUserComma / stepGrantUser
	stepGrantUserComma                                    // "," => stepGrantUserName
	stepRevokePrivilege                                   // "UPDATE" => stepRevokePrivilege / stepRevokeOpeningParens / stepRevokeOnTable
	stepRevokeOpeningParens                               // "(" => stepRevokePrivilegeField
	stepRevokePrivilegeField                              // 'Sno' => stepRevokePrivilegeCommaOrClosingParens
	stepRevokePrivilegeCommaOrClosingParens               // "," / ")" => stepRevokeComma / stepRevokePrivilegeField
	stepRevokeOnTable                                     // "ON TABLE" => stepRevokeTableName
	stepRevokeTableName                                   // 'Student' => stepRevokeTableComma / stepRevokeTo
	stepRevokeFrom                                        // "FROM" => stepRevokeUserName
	stepRevokeUserName                                    // 'U1' => stepRevokeUserComma / stepRevokeUser
	stepRevokeUserComma                                   // "," => stepRevokeUserName
)
