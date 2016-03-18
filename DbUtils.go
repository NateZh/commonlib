package commonlib

import (
	"database/sql"
	"errors"
	"strconv"
	"time"
)

/**
 * 数据库处理
 * @param action	数据库操作的具体方法
 * return		结果信息， 错误信息
 *
 * example:
 * res, err := Action(func(db *sql.Tx) (interface{}, error) {
 *   inRes, inErr := DbInsert(db, inSql, inParams)
 *   return inRes, inErr
 * })
 */
func Action(action interface{}) (map[string]interface{}, error) {
	dbAction, ok := action.(func(*sql.DB) (map[string]interface{}, error))
	if ok {
		//Log.Error("非事务处理")
		return DbAction(dbAction)
	}

	txAction, ok := action.(func(*sql.Tx) (map[string]interface{}, error))
	if ok {
		//Log.Error("事务处理")
		return DbTransactionAction(txAction)
	}

	return nil, errors.New("数据处理异常: 无法正确获取数据库数据处理方式")
}

/**
 * 数据库处理
 * @param dbAction 数据库操作的具体方法
 * return 结果信息， 错误信息
 *
 * example:
 * res, err := DbAction(func(db *sql.Tx) (interface{}, error) {
 *   inRes, inErr := DbInsert(db, inSql, inParams)
 *   return inRes, inErr
 * })
 */
func DbAction(dbAction func(*sql.DB) (map[string]interface{}, error)) (map[string]interface{}, error) {
	db := GetMySQL()
	defer db.Close()

	return dbAction(db)
}

/**
 * 包含事务的数据库处理
 * @param txAction 数据库操作的具体方法
 *
 * return 结果信息， 错误信息
 *
 * example:
 * res, err := DbTransactionAction(func(tx *sql.Tx) (interface{}, error) {
 *   inRes, inErr := TxInsert(tx, inSql, inParams)
 *   return inRes, inErr
 * })
 */
func DbTransactionAction(txAction func(*sql.Tx) (map[string]interface{}, error)) (map[string]interface{}, error) {
	db := GetMySQL()
	defer db.Close()

	// 开启事务
	tx, err := db.Begin()
	if err != nil {
		Log.Error("db.Begin: ", err.Error())
		return BuildDbErrorMessage("开启事务时，数据库异常： " + err.Error()), err
	}
	defer func() {
		if err != nil && tx != nil {
			// 事务回滚
			if rbErr := tx.Rollback(); rbErr != nil {
				Log.Error("tx.Rollback: ", rbErr.Error())
				return
			}
		}
	}()
	t := time.Now()
	/**
	 * @param db 用于事务内查询操作，无法查询到当前事务影响的结果
	 * @param tx 当前操作的事务
	 */
	actionResult, err := txAction(tx)
	if err != nil {
		return actionResult, err
	}
	Log.Debug("事务处理时间: ", time.Now().Sub(t))

	// 提交事务
	if err = tx.Commit(); err != nil {
		Log.Error("tx.Commit: ", err.Error())
		return BuildDbErrorMessage("提交事务，数据库异常" + err.Error()), err
	}
	// 关闭数据库连接
	if err = db.Close(); err != nil {
		Log.Error("db.Close: ", err.Error())
		return BuildDbErrorMessage("关闭数据库连接，数据库异常" + err.Error()), err
	}

	return actionResult, err
}

/****
 * 数据插入
 * @param opObj		操作数据库对象 *sql.DB | *sql.Tx
 * @param sqlStr	操作的sql语句
 * @param args		参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := Insert(db, "insert into table_name ('filed') values (?);", "hello")
 */
func Insert(opObj interface{}, sqlStr string, args ...interface{}) (sql.Result, error) {
	db, ok := opObj.(*sql.DB)
	if ok {
		return DbInsert(db, sqlStr, args...)
	}

	tx, ok := opObj.(*sql.Tx)
	if ok {
		return TxInsert(tx, sqlStr, args...)
	}

	return nil, errors.New("插入失败: 无法获取数据库操作对象")
}

/****
 * 数据删除
 * @param opObj		操作数据库对象 *sql.DB | *sql.Tx
 * @param sqlStr	操作的sql语句
 * @param args		参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := Delete(db, "delete from table_name where id=?;", 1)
 */
func Delete(opObj interface{}, sqlStr string, args ...interface{}) (sql.Result, error) {
	db, ok := opObj.(*sql.DB)
	if ok {
		return DbDelete(db, sqlStr, args...)
	}

	tx, ok := opObj.(*sql.Tx)
	if ok {
		return TxDelete(tx, sqlStr, args...)
	}

	return nil, errors.New("删除失败: 无法获取数据库操作对象")
}

/****
 * 数据更新
 * @param opObj		操作数据库对象 *sql.DB | *sql.Tx
 * @param sqlStr	操作的sql语句
 * @param args		参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := Update(db, "update table_name set field=?;", "hello")
 */
func Update(opObj interface{}, sqlStr string, args ...interface{}) (sql.Result, error) {
	db, ok := opObj.(*sql.DB)
	if ok {
		return DbUpdate(db, sqlStr, args...)
	}

	tx, ok := opObj.(*sql.Tx)
	if ok {
		return TxUpdate(tx, sqlStr, args...)
	}

	return nil, errors.New("更新失败: 无法获取数据库操作对象")
}

/****
 * 数据查询
 * @param opObj		操作数据库对象 *sql.DB | *sql.Tx
 * @param sqlStr	操作的sql语句
 * @param args		参数列表
 *
 * return 数据集sql.Rows， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := Query(db, "select fields from table_name where field_name=?;", "hello")
 */
func Query(opObj interface{}, sqlStr string, args ...interface{}) ([]map[string]string, error) {
	db, ok := opObj.(*sql.DB)
	if ok {
		return DbQuery(db, sqlStr, args...)
	}

	tx, ok := opObj.(*sql.Tx)
	if ok {
		return TxQuery(tx, sqlStr, args...)
	}

	return nil, errors.New("查询错误: 无法获取数据库操作对象")
}

/****
 * 数据查询(一个)
 * @param opObj		操作数据库对象 *sql.DB | *sql.Tx
 * @param sqlStr	操作的sql语句
 * @param args		参数列表
 *
 * return 数据集sql.Rows， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := QueryOne(db, "select fields from table_name where field_name=?;", "hello")
 */
func QueryOne(opObj interface{}, sqlStr string, args ...interface{}) (map[string]string, error) {
	db, ok := opObj.(*sql.DB)
	if ok {
		return DbQueryOne(db, sqlStr, args...)
	}

	tx, ok := opObj.(*sql.Tx)
	if ok {
		return TxQueryOne(tx, sqlStr, args...)
	}

	return nil, errors.New("查询错误: 无法获取数据库操作对象")
}

/****
 * 数据插入
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := DbInsert(db, "insert into table_name ('filed') values (?);", "hello")
 */
func DbInsert(db *sql.DB, sqlStr string, args ...interface{}) (sql.Result, error) {
	return dbOperation(db, sqlStr, args...)
}

/****
 * 数据删除
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := DbDelete(db, "delete from table_name where id=?;", 1)
 */
func DbDelete(db *sql.DB, sqlStr string, args ...interface{}) (sql.Result, error) {
	return dbOperation(db, sqlStr, args...)
}

/****
 * 数据更新
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := DbUpdate(db, "update table_name set field=?;", "hello")
 */
func DbUpdate(db *sql.DB, sqlStr string, args ...interface{}) (sql.Result, error) {
	return dbOperation(db, sqlStr, args...)
}

/****
 * 数据查询
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 数据集sql.Rows， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := DbQuery(db, "select fields from table_name where field_name=?;", "hello")
 */
func DbQuery(db *sql.DB, sqlStr string, args ...interface{}) ([]map[string]string, error) {
	stmt, err := db.Prepare(sqlStr)
	defer stmt.Close()

	if err != nil {
		Log.Error(err)
		return nil, err
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	result, err := rowsToMap(rows)
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	//	Log.Debug(sqlStr, args)

	return result, err
}

/****
 * 数据查询(一个)
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 数据集sql.Rows， 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := DbQueryOne(db, "select fields from table_name where field_name=?;", "hello")
 */
func DbQueryOne(db *sql.DB, sqlStr string, args ...interface{}) (map[string]string, error) {
	result, err := DbQuery(db, sqlStr, args...)
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	//	Log.Debug(sqlStr, args)

	if len(result) > 0 {
		return result[0], nil
	}

	return make(map[string]string), err
}

/**
 * 数据查询(事务)
 * @param tx     数据库事务对象
 * @param sqlStr 查詢的sql语句
 * @param args   参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   res, err := TxQuery(tx, "select fields from table_name where field_name=?;", "hello")
 */
func TxQuery(tx *sql.Tx, sqlStr string, args ...interface{}) ([]map[string]string, error) {
	stmt, err := tx.Prepare(sqlStr)

	if err != nil {
		Log.Error(err)
		return nil, err
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	result, err := rowsToMap(rows)
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	return result, err
}

/****
 * 数据查询(一个)(事务)
 * @param tx     数据库事务对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 数据集sql.Rows， 错误信息
 *
 * example:
 *   res, err := TxQueryOne(tx, "select fields from table_name where field_name=?;", "hello")
 */
func TxQueryOne(tx *sql.Tx, sqlStr string, args ...interface{}) (map[string]string, error) {
	result, err := TxQuery(tx, sqlStr, args...)
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	//	Log.Debug(sqlStr, args)

	if len(result) > 0 {
		return result[0], nil
	}

	return make(map[string]string), err
}

/**
 * 数据插入(事务)
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   res, err := TxInsert(tx, "insert into table_name ('filed') values (?);", "hello")
 */
func TxInsert(tx *sql.Tx, sqlStr string, args ...interface{}) (sql.Result, error) {
	return txOperation(tx, sqlStr, args...)
}

/**
 * 数据删除(事务)
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   res, err := TxDelete(tx, "delete from table_name where id=?;", 1)
 */
func TxDelete(tx *sql.Tx, sqlStr string, args ...interface{}) (sql.Result, error) {
	return txOperation(tx, sqlStr, args...)
}

/**
 * 数据更新(事务)
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 处理结果， 错误信息
 *
 * example:
 *   res, err := TxUpdate(tx, "update table_name set field=?;", "hello")
 */
func TxUpdate(tx *sql.Tx, sqlStr string, args ...interface{}) (sql.Result, error) {
	return txOperation(tx, sqlStr, args...)
}

/****
 * 数据增删改处理
 * @param db     操作数据库对象
 * @param sqlStr 操作的sql语句
 * @param args   参数列表
 *
 * return 处理结果， 错误信息
 */
func dbOperation(db *sql.DB, sqlStr string, args ...interface{}) (sql.Result, error) {
	stmt, err := db.Prepare(sqlStr)

	if err != nil {
		Log.Error(err)
		return nil, err
	}

	result, err := stmt.Exec(args...)

	if err != nil {
		Log.Error(err)
		return nil, err
	}

	return result, err
}

/**
 * 数据增删改处理(事务)
 * @param tx     当前处理的事务
 * @param sqlStr 待处理的sql
 * @param args   sql所需的参数列表
 *
 * return 处理结果， 错误信息
 */
func txOperation(tx *sql.Tx, sqlStr string, args ...interface{}) (sql.Result, error) {
	stmt, err := tx.Prepare(sqlStr)
	if err != nil {
		Log.Error("tx.Prepare: ", err.Error())
		return nil, err
	}
	defer func() {
		if stmtErr := stmt.Close(); stmtErr != nil {
			Log.Error("stmt.Close: ", stmtErr.Error())
			return
		}
	}()

	result, err := stmt.Exec(args...)

	if err != nil {
		Log.Error(err)
		return nil, err
	}

	//Log.Debug(sqlStr)

	return result, err
}

/****
 * 分页
 * @param db     操作数据库对象
 * @param countSqlStr countsql语句
 * @param dataSqlStr 数据sql语句
 * @param args   count参数
 * @param args   数据参数
 * @param pageId   第几页
 * @param recPerPage   每页几条
 *
 * return 数据集，pager对象 错误信息
 *
 * example:
 *   db := GetMySQL()
 *   res, err := DbQuery(db, "select fields from table_name where field_name=?;", "hello")
 */
func DbPage(db *sql.DB, countSql string, dataSql string, countParams []interface{}, params []interface{}, pageId, recPerPage int) ([]map[string]string, *Pager, error) {

	rec, err := DbQueryOne(db, countSql, countParams...)

	if err != nil {
		Log.Error(err)
		return nil, nil, err
	}

	total, _ := strconv.Atoi(rec["count(1)"])

	pager := buildPager(pageId, recPerPage, total)

	dataSql += " limit ?,?"
	params = append(params, (pager.PageId-1)*pager.RecPerPage)
	params = append(params, pager.RecPerPage)

	dataRec, err := DbQuery(db, dataSql, params...)

	if err != nil {
		Log.Error(err)
		return nil, nil, err
	}

	return dataRec, buildPager(pageId, recPerPage, total), nil
}

func rowsToMap(rows *sql.Rows) ([]map[string]string, error) {
	cols, _ := rows.Columns()
	values := make([]sql.RawBytes, len(cols))
	scans := make([]interface{}, len(cols))
	for i := range values {
		scans[i] = &values[i]
	}
	var results []map[string]string
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			Log.Error("Error: ", err)
			return nil, err
		}
		row := make(map[string]string)
		for i, v := range values {
			key := cols[i]
			row[key] = string(v)
		}
		results = append(results, row)
	}
	rows.Close()

	return results, nil
}

/*
func DbCommonUpdate(tableName,id string,tx *sql.Tx, data map[string]interface{}) (sql.Result, error) {

	sql := "update "+tableName+" set %v where id=?"

	params := []interface{}{}

	setSql := ""

	for key, value := range data {
		setSql += key + "=?,"
		params = append(params, value)
	}

	params = append(params, id)

	setSql = Substr(setSql, 0, len(setSql)-1)

	sql = fmt.Sprintf(sql, setSql)
	Log.Debug(sql)

	return TxUpdate(tx,sql,params)
}


func DbCommonInsert(tableName string,tx *sql.Tx, data map[string]interface{}) (sql.Result, error) {

	sql := "insert into "+tableName+"(%v) values(%v)"

	params := []interface{}{}

	columnSql := ""
	valueSql := ""

	for key, value := range data {
		columnSql += key + ","
		valueSql += "?,"
		params = append(params, value)
	}

	columnSql = Substr(columnSql, 0, len(columnSql)-1)
	valueSql = Substr(valueSql, 0, len(columnSql)-1)

	sql = fmt.Sprintf(sql, columnSql,valueSql)
	Log.Debug(sql)

	return TxInsert(tx,sql,params)
}
*/
