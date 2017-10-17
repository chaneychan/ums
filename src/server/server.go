package main

import (
	"dao"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dlintw/goconf"
	"github.com/garyburd/redigo/redis"
	"html/template"
	"io"
	"log"
	"model"
	"net/http"
	"os"
	"path"
	"redisCliPool"
	"tcp"
)

func main() {
	StartUp()
}

const (
	UPLOAD_DIR = "./uploads"
)

func StartUp() {
	log.Println("start...")
	dao.InitDbPool()
	InitRedis()
	initTcpSer()
}

type Server struct {
	Phost string
	Pmap  map[uint8]string
}

func (m Server) PdoRegister(in []byte) (out []byte) {
	fmt.Printf("客户端发来name包:%s\n", in)
	/** process... **/
	bytes := []byte("hello2")
	return bytes
}

func initTcpSer() {
	m := Server{
		Phost: ":9095",
		Pmap:  make(map[uint8]string),
	}
	m.Pmap[0x01] = "dologin"
	m.Pmap[0x02] = "doRegister"
	err := tcp.New(m)
	if err != nil {
		fmt.Println(err)
	}
}

func InitRedis() {
	var err error
	conf_file := flag.String("configserver", "./configs/config.ini", "set redis config file.")

	l_conf, err := goconf.ReadConfigFile(*conf_file)
	if err != nil {
		log.Print("LoadConfiguration: Error: Could not open config file：", conf_file, err)
		os.Exit(1)
	}
	redisCliPool.InitRedisPool(l_conf)
	//	defer redisCliPool.Clipool.Close()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

var userDao *dao.UserDao = new(dao.UserDao)

func (m Server) Pdologin(in []byte) (out []byte) {
	fmt.Printf("客户端发来type包:%s\n", in)
	var user model.User
	var r model.Result
	json.Unmarshal(in, &user)

	conn := redisCliPool.Clipool.Get()
	defer conn.Close()
	val, err := conn.Do("GET", user.Name)
	valueBytes, err := redis.Bytes(val, err) //应该验证返回值的错误与否

	var obj *model.User

	if err != nil {
		sql := "SELECT * FROM userinfo where name=? and password =?"
		log.Print("name does not exists redis,sql : "+sql, err)
		obj = userDao.QueryUser(sql, user.Name, user.Password)
		if obj == nil {
			r.IsOk = false
			//			err = tmpl.Execute(w, map[string]string{"err_msg": "username or password is invalid!"})
			//			checkErr(err)
		} else {
			r.IsOk = true
			user = *obj
			//			err = tmpl.Execute(w, map[string]interface{}{"user": user})
			//			checkErr(err)
			value, err := json.Marshal(user)
			r.User = user
			//			if err != nil {
			//				log.Print("json marshal err,%s", err)
			//				return
			//			}
			log.Print("send data：", user.Name, user)
			_, err = conn.Do("SET", user.Name, value)
			if err != nil {
				log.Print("redis do set error：", err)
				//从数据库查出数据，存入redis，如果有错误，可以根据情况重试
			}
		}
	} else {
		log.Print("redis data:", user.Name, string(valueBytes))
		var userRedis model.User
		err = json.Unmarshal(valueBytes, &userRedis)
		//		if err != nil {
		//			log.Print("json unmarshal err:", err)
		//			return
		//		}
		log.Print(user)
		if user.Password != userRedis.Password {
			r.IsOk = false
			//			err = tmpl.Execute(w, map[string]string{"err_msg": "username or password is invalid!"})
			//			checkErr(err)
			//			return
		} else {
			//			err = tmpl.Execute(w, map[string]interface{}{"user": user})
			//			checkErr(err)
			r.IsOk = true
			r.User = userRedis
		}
	}
	value, err := json.Marshal(r)
	return value
}

//doRegister
func doRegisterHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("doRegisterHandler...")
	r.ParseMultipartForm(32 << 20) //接收文件的时候应该控制文件大小，前段后端都做验证
	name := r.FormValue("name")
	password := r.FormValue("password")
	nickname := r.FormValue("nickname")
	f, h, err := r.FormFile("profile")
	//	log.Print(name + password + nickname + h.Filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filename := h.Filename
	log.Print("filename : " + filename)
	defer f.Close()

	filedir := UPLOAD_DIR + "/" + filename //这里可以给文件按照用户特性或者全局唯一id的方式重命名以免不同用户传入相同的文件名

	if err := os.MkdirAll(path.Dir(filedir), 0777); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	t, err := os.Create(filedir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer t.Close()
	if _, err := io.Copy(t, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sql := "INSERT INTO userinfo(name, password, profile, nickname) VALUES (?,?,?,?)"
	log.Print(sql)
	i := userDao.Save(sql, name, password, filedir, nickname) // name唯一，当出现Duplicate entry，可以捕获这个异常返回给前端，给予更友好的提示，比如:用户名已被使用，请换一个，或者直接给用户生成一个友好且系统中不存在的
	if i > 0 {
		tmpl, err := template.ParseFiles("src/web/registered.html")
		checkErr(err)
		err = tmpl.Execute(w, nil)
		checkErr(err)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//updateUserHandler
//此方最好应该判断是否是本人，是否登录等校验，不能让人篡改信息
//doRegister和此方法有大量代码重用，可以优化
func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("updateUserHandler...")
	r.ParseMultipartForm(32 << 20) //接收文件的时候应该控制文件大小，前段后端都做验证
	id := r.FormValue("id")
	name := r.FormValue("name")
	nickname := r.FormValue("nickname")
	f, h, err := r.FormFile("profile")
	log.Print(id, nickname, err)
	//	if err != nil {
	//		http.Error(w, err.Error(), http.StatusInternalServerError)
	//		return
	//	}

	var sql string
	if h != nil { //如果文件有update
		filename := h.Filename
		log.Print("filename : " + filename)
		filedir := UPLOAD_DIR + "/" + filename //这里可以给文件按照用户特性或者全局唯一id的方式重命名以免不同用户传入相同的文件名
		if err := os.MkdirAll(path.Dir(filedir), 0777); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		t, err := os.Create(filedir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer t.Close()
		if _, err := io.Copy(t, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sql = "UPDATE userinfo SET nickname=?,profile=? where id =?"
		log.Print(sql)
		userDao.Save(sql, nickname, filedir, id)
		defer f.Close()
	} else {
		sql = "UPDATE userinfo SET nickname=? where id =?"
		log.Print(sql)
		userDao.Save(sql, nickname, id)
	}
	//删除redis值，or 更新，这里可以根据用户习惯数据分析用户更新自己的信息后是否会登录，如果用户更新自己的信息后大概率不登录了，直接删除以节省空间
	//这一步和上一步的数据库sava最好在一个事物当中，防止redis删除失败，防止save失败redis被删除等导致脏数据的情况
	conn := redisCliPool.Clipool.Get()
	if conn == nil {
		log.Print("redis connnection is nil")
	}
	// 用完后将连接放回连接池
	defer conn.Close()
	log.Print("move data name:", name)
	conn.Do("DEL", name) //暂时不处理失败的情况

	tmpl, err := template.ParseFiles("src/web/updated.html")
	checkErr(err)
	err = tmpl.Execute(w, nil)
	checkErr(err)
}
