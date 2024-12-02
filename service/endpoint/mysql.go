package endpoint

import (
	"cardappcanal/global"
	"cardappcanal/metrics"
	"cardappcanal/model"
	"database/sql"
	"fmt"
	"github.com/go-mysql-org/go-mysql/mysql"
	"log"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql" // MySQL 驱动
)

type MysqlEndpoint struct {
	db   *sql.DB
	lock sync.Mutex
}

func (s *MysqlEndpoint) Ping() error {
	if s.db == nil {
		return fmt.Errorf("MySQL connection is not initialized")
	}
	return s.db.Ping()
}

func (s *MysqlEndpoint) Stock(rows []*model.RowRequest) int64 {
	models := make(map[string][]map[string]interface{}) // 存储每个表的批量插入数据
	for _, row := range rows {
		rule, _ := global.RuleIns(row.RuleKey)
		if rule.TableColumnSize != len(row.Row) {
			log.Printf("schema mismatching for rule %s", row.RuleKey)
			continue
		}

		// 将行数据转换为键值对
		kvm := rowMap(row, rule, false)
		id := primaryKey(row, rule)
		kvm["id"] = id

		// 按表名分类数据
		key := rule.MysqlDatabase + "." + rule.Table
		models[key] = append(models[key], kvm)
	}

	var totalInserted int64
	for table, rows := range models {
		if len(rows) == 0 {
			continue
		}

		// 构建批量插入的 SQL 和参数
		columns := []string{}
		for col := range rows[0] {
			columns = append(columns, col)
		}

		valuePlaceholders := "(" + strings.Repeat("?, ", len(columns)-1) + "?)"
		values := []interface{}{}
		valueStrings := []string{}

		for _, row := range rows {
			valueStrings = append(valueStrings, valuePlaceholders)
			for _, col := range columns {
				values = append(values, row[col])
			}
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
			table,
			strings.Join(columns, ", "),
			strings.Join(valueStrings, ", "))

		// 执行批量插入
		result, err := s.db.Exec(query, values...)
		if err != nil {
			if s.isDuplicateKeyError(err.Error()) {
				log.Printf("Duplicate key error on table %s: %v", table, err)
				totalInserted += s.handleSlowInsert(rows, table) // 处理慢插入
			} else {
				log.Printf("Failed to insert into table %s: %v", table, err)
			}
			continue
		}

		// 获取插入的行数
		rowsAffected, _ := result.RowsAffected()
		totalInserted += rowsAffected
	}

	return totalInserted
}

// 处理慢插入（逐条插入）
func (s *MysqlEndpoint) handleSlowInsert(rows []map[string]interface{}, table string) int64 {
	var inserted int64
	for _, row := range rows {
		columns := []string{}
		values := []interface{}{}
		placeholders := []string{}

		for col, val := range row {
			columns = append(columns, col)
			values = append(values, val)
			placeholders = append(placeholders, "?")
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))

		_, err := s.db.Exec(query, values...)
		if err != nil {
			log.Printf("Failed to insert row into table %s: %v", table, err)
			continue
		}

		inserted++
	}

	return inserted
}

// 判断是否为主键冲突错误
func (s *MysqlEndpoint) isDuplicateKeyError(errMsg string) bool {
	return strings.Contains(errMsg, "Duplicate entry")
}
func mapValues(kvm map[string]interface{}, columns []string) []interface{} {
	values := make([]interface{}, len(columns))
	for i, col := range columns {
		if val, ok := kvm[col]; ok {
			values[i] = val
		} else {
			values[i] = nil // 如果键不存在，使用 nil 填充
		}
	}
	return values
}

func (s *MysqlEndpoint) batchInsert(tx *sql.Tx, table string, columns []string, values [][]interface{}) (int64, error) {
	if len(values) == 0 {
		return 0, nil
	}

	// 构建插入 SQL
	placeholder := "(" + strings.TrimRight(strings.Repeat("?, ", len(columns)), ", ") + ")"
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s ON DUPLICATE KEY UPDATE %s",
		table,
		strings.Join(columns, ", "),
		strings.TrimRight(strings.Repeat(placeholder+", ", len(values)), ", "),
		buildUpdateFields(columns),
	)

	// 打平参数
	args := make([]interface{}, 0)
	for _, row := range values {
		args = append(args, row...)
	}

	// 执行插入
	result, err := tx.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	// 返回影响的行数
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rows, nil
}

func buildUpdateFields(columns []string) string {
	updates := make([]string, 0, len(columns))
	for _, col := range columns {
		if col != "id" { // 忽略主键更新
			updates = append(updates, fmt.Sprintf("%s = VALUES(%s)", col, col))
		}
	}
	return strings.Join(updates, ", ")
}

func newMysqlEndpoint() *MysqlEndpoint {
	// 获取配置
	username := global.Cfg().MysqlUsername
	password := global.Cfg().MysqlPassword
	addr := global.Cfg().MysqlAddr
	port := global.Cfg().MysqlPort
	database := global.Cfg().MysqlDatabase

	// 构建 DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, addr, port, database)
	fmt.Println(dsn)
	// 打开 MySQL 连接
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to connect to MySQL: %v", err)
	}

	// 返回连接实例
	return &MysqlEndpoint{
		db: db,
	}
}

func (s *MysqlEndpoint) Connect() error {
	return s.db.Ping()
}
func (s *MysqlEndpoint) ChangeTableName(query []byte) error {
	fmt.Print("----------------------")
	fmt.Print(string(query))
	fmt.Print("----------------------")
	_, err := s.db.Exec(string(query))
	if err != nil {
		fmt.Print(err.Error())
	}
	return nil
}

func (s *MysqlEndpoint) Consume(from mysql.Position, rows []*model.RowRequest) error {
	for _, row := range rows {
		rule, _ := global.RuleIns(row.RuleKey)
		if rule.TableColumnSize != len(row.Row) {
			log.Printf("schema mismatching for rule %s", row.RuleKey)
			continue
		}

		metrics.UpdateActionNum(row.Action, row.RuleKey)

		kvm := rowMap(row, rule, false)
		id := primaryKey(row, rule)
		kvm["id"] = id

		switch row.Action {
		case "insert":
			// INSERT 操作
			columns := []string{}
			values := []interface{}{}
			for col, val := range kvm {
				columns = append(columns, col)
				values = append(values, val)
			}
			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
				rule.MysqlDatabase+"."+rule.Table,
				strings.Join(columns, ", "),
				strings.Repeat("?, ", len(columns)-1)+"?")
			_, err := s.db.Exec(query, values...)
			if err != nil {
				log.Printf("insert action failed on table %s: %v", rule.Table, err)
				continue // 继续处理其他行
			}

		case "update":
			// UPDATE 操作
			setClauses := []string{}
			values := []interface{}{}
			for col, val := range kvm {
				setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
				values = append(values, val)
			}
			query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?",
				rule.MysqlDatabase+"."+rule.Table,
				strings.Join(setClauses, ", "))
			values = append(values, id)
			_, err := s.db.Exec(query, values...)
			if err != nil {
				log.Printf("update action failed on table %s: %v", rule.Table, err)
				continue
			}

		case "delete":
			// DELETE 操作
			query := fmt.Sprintf("DELETE FROM %s WHERE id = ?",
				rule.MysqlDatabase+"."+rule.Table)
			_, err := s.db.Exec(query, id)
			if err != nil {
				log.Printf("delete action failed on table %s: %v", rule.Table, err)
				continue
			}
		}
	}
	log.Printf("Processed %d rows successfully", len(rows))
	return nil
}

func (s *MysqlEndpoint) Close() {
	if s.db != nil {
		s.db.Close()
	}
}
