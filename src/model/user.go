package model

import ()

type User struct {
	Id         int
	Name       string
	Password   string
	Profile    string //存储文件地址
	Nickname   string
	CreateTime int
}

func (user *User) ToString() (str string) {
	str = "name:" + user.Name
	return str
}
