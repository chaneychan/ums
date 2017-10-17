package dao

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/Go-SQL-Driver/MySQL"
	"github.com/dlintw/goconf"
	"log"
	"os"
	"time"
)

var (
	db          *sql.DB
	name        string
	password    string
	addr        string
	dbname      string
	maxOpen     int
	maxIdle     int
	maxLifetime int
)

func newDbPool(name string, password string, addr string, dbname string, maxOpen int, maxIdle int, maxLifetime int) *sql.DB {
	db, err := sql.Open("mysql", name+":"+password+"@tcp("+addr+")/"+dbname+"?charset=utf8")
	if err != nil {
		panic(err.Error())
		fmt.Println(err)
	}
	//	defer db.Close()
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)
	return db
}

func InitDbPool() {
	var err error
	conf_file := flag.String("configDB", "./configs/config.ini", "set db config file.")

	conf, err := goconf.ReadConfigFile(*conf_file)
	if err != nil {
		log.Print("LoadConfiguration: Error: Could not open config file：", conf_file, err)
		os.Exit(1)
	}
	name, _ = conf.GetString("dbPool", "name")
	password, _ = conf.GetString("dbPool", "password")
	addr, _ = conf.GetString("dbPool", "addr")
	dbname, _ = conf.GetString("dbPool", "dbname")
	maxOpen, _ = conf.GetInt("dbPool", "maxOpen")
	maxIdle, _ = conf.GetInt("dbPool", "maxIdle")
	maxLifetime, _ = conf.GetInt("dbPool", "maxLifetime")
	db = newDbPool(name, password, addr, dbname, maxOpen, maxIdle, maxLifetime)
}

func insert(sqlstr string, args ...interface{}) (int64, error) {
	stmtIns, err := db.Prepare(sqlstr)
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	result, err := stmtIns.Exec(args...)
	if err != nil {
		panic(err.Error())
	}
	return result.LastInsertId()
}

func exec(sqlstr string, args ...interface{}) (int64, error) {
	stmtIns, err := db.Prepare(sqlstr)
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	result, err := stmtIns.Exec(args...)
	if err != nil {
		panic(err.Error())
	}
	return result.RowsAffected()
}

func fetchRow(sqlstr string, args ...interface{}) (*sql.Row, error) {
	stmtOut, err := db.Prepare(sqlstr)
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	row := stmtOut.QueryRow(args...)
	if err != nil {
		panic(err.Error())
	}
	return row, nil

	//	columns, err := rows.Columns()
	//	if err != nil {
	//		panic(err.Error())
	//	}
	//
	//	values := make([]sql.RawBytes, len(columns))
	//	scanArgs := make([]interface{}, len(values))
	//	ret := make(map[string]string, len(scanArgs))
	//
	//	for i := range values {
	//		scanArgs[i] = &values[i]
	//	}
	//	for rows.Next() {
	//		err = rows.Scan(scanArgs...)
	//		if err != nil {
	//			panic(err.Error())
	//		}
	//		var value string
	//
	//		for i, col := range values {
	//			if col == nil {
	//				value = "NULL"
	//			} else {
	//				value = string(col)
	//			}
	//			ret[columns[i]] = value
	//		}
	//		break //get the first row only
	//	}
	//	return &ret, nil
}

func fetchRows(sqlstr string, args ...interface{}) (*[]map[string]string, error) {
	stmtOut, err := db.Prepare(sqlstr)
	if err != nil {
		panic(err.Error())
	}
	defer stmtOut.Close()

	rows, err := stmtOut.Query(args...)
	defer rows.Close()

	if err != nil {
		panic(err.Error())
	}

	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error())
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))

	ret := make([]map[string]string, 0)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error())
		}
		var value string
		vmap := make(map[string]string, len(scanArgs))
		for i, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			vmap[columns[i]] = value
		}
		ret = append(ret, vmap)
	}
	return &ret, nil
}
