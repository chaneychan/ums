package dao

import (
	"log"
	"model"
)

type UserDao struct {
}

func (userDao *UserDao) QueryUser(sqlstr string, args ...interface{}) (objs *model.User) {
	val, err := fetchRow(sqlstr, args...)
	if err != nil {
		log.Println("queryUser err ：", err)
		return nil
	}
	var Id int
	var Name string
	var Password string
	var Profile string
	var Nickname string
	var CreateTime int
	val.Scan(&Id, &Name, &Password, &Profile, &Nickname, &CreateTime)
	result := &model.User{
		Id, Name, Password, Profile, Nickname, CreateTime}
	return result
}

//返回id
func (userDao *UserDao) Save(sqlstr string, args ...interface{}) int64 {
	i, err := insert(sqlstr, args...)
	if err != nil {
		log.Println("queryUser err ：", err)
		return 0
	}
	return i
}
