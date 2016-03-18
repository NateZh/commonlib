
package commonlib

import (
	"database/sql"
	"fmt"
	"github.com/astaxie/beego"
	_ "github.com/go-sql-driver/mysql"
)

var mySQLPool chan *sql.DB

func GetMySQL() *sql.DB {

	maxPoolSize, _ := beego.AppConfig.Int("maxPoolSize")

	if mySQLPool == nil {
		mySQLPool = make(chan *sql.DB, maxPoolSize)
	}

	dbUrl := beego.AppConfig.String("mysqlurls")
	dbName := beego.AppConfig.String("mysqldb")
	dbUserName := beego.AppConfig.String("mysqluser")
	dbPwd := beego.AppConfig.String("mysqlpass")

	// Log.Debug("url: ", dbUrl, "    name: ", dbName, "    uname: ", dbUserName, "    pwd: ", dbPwd)

	if len(mySQLPool) == 0 {
		go func() {
			for i := 0; i < maxPoolSize/2; i++ {
				db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v)/%v?charset=utf8", dbUserName, dbPwd, dbUrl, dbName))
				if err != nil {
					Log.Warn(err)
					continue
				}
				putMySQL(db)
			}
		}()
	}
	return <-mySQLPool
}

func putMySQL(conn *sql.DB) {

	maxPoolSize, _ := beego.AppConfig.Int("maxPoolSize")

	if mySQLPool == nil {
		mySQLPool = make(chan *sql.DB, maxPoolSize)
	}

	if len(mySQLPool) == maxPoolSize {
		conn.Close()
		return
	}

	mySQLPool <- conn
}
