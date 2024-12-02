package luaengine

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/yuin/gopher-lua"

	"cardappcanal/global"
	"cardappcanal/util/stringutil"
)

func mysqlModule(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, _mysqlModuleApi)
	L.Push(t)
	return 1
}

var _mysqlModuleApi = map[string]lua.LGFunction{
	"INSERT": mysqlInsert,
	"UPDATE": mysqlUpdate,
	"DELETE": mysqlDelete,
}

func mysqlInsert(L *lua.LState) int {
	table := L.CheckAny(1)
	data := L.CheckTable(2)

	sqlQuery := fmt.Sprintf("INSERT INTO %s SET ", lvToString(table))
	sqlQuery += tableToSQL(data)

	ret := L.GetGlobal(_globalRET)
	L.SetTable(ret, lua.LString(stringutil.UUID()), lua.LString(sqlQuery))
	return 0
}

func mysqlUpdate(L *lua.LState) int {
	table := L.CheckAny(1)
	id := L.CheckAny(2)
	data := L.CheckTable(3)

	sqlQuery := fmt.Sprintf("UPDATE %s SET %s WHERE id='%s'", lvToString(table), tableToSQL(data), lvToString(id))

	ret := L.GetGlobal(_globalRET)
	L.SetTable(ret, lua.LString(stringutil.UUID()), lua.LString(sqlQuery))
	return 0
}

func mysqlDelete(L *lua.LState) int {
	table := L.CheckAny(1)
	id := L.CheckAny(2)

	sqlQuery := fmt.Sprintf("DELETE FROM %s WHERE id='%s'", lvToString(table), lvToString(id))

	ret := L.GetGlobal(_globalRET)
	L.SetTable(ret, lua.LString(stringutil.UUID()), lua.LString(sqlQuery))
	return 0
}

func tableToSQL(data *lua.LTable) string {
	queryParts := make([]string, 0)

	data.ForEach(func(key lua.LValue, value lua.LValue) {
		queryParts = append(queryParts, fmt.Sprintf("%s='%s'", lvToString(key), lvToString(value)))
	})

	return strings.Join(queryParts, ", ")
}

func DoMysqlOps(input map[string]interface{}, action string, rule *global.Rule, db *sql.DB) ([]sql.Result, error) {
	L := _pool.Get()
	defer _pool.Put(L)

	row := L.NewTable()
	paddingTable(L, row, input)
	ret := L.NewTable()
	L.SetGlobal(_globalRET, ret)
	L.SetGlobal(_globalROW, row)
	L.SetGlobal(_globalACT, lua.LString(action))

	funcFromProto := L.NewFunctionFromProto(rule.LuaProto)
	L.Push(funcFromProto)
	err := L.PCall(0, lua.MultRet, nil)
	if err != nil {
		return nil, err
	}

	results := make([]sql.Result, 0, ret.Len())
	ret.ForEach(func(k lua.LValue, v lua.LValue) {
		sqlQuery := lvToString(v)

		// 执行 MySQL 查询
		result, err := db.Exec(sqlQuery)
		if err != nil {
			results = append(results, nil)
		} else {
			results = append(results, result)
		}
	})

	return results, nil
}
