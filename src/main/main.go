package main

import (
	"common"
	"dao"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dlintw/goconf"
	"github.com/satori/go.uuid"
	"html/template"
	"io"
	"log"
	"model"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"redisCliPool"
	"session"
	"tcp"
	"time"
)

func main() {
	StartUp()
}

const (
	UPLOAD_DIR = "./uploads"
)

func StartUp() {
	log.Println("start...")
	Register()
	InitRedis()
	dao.InitDbPool()
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Println("ListenAndServe", err.Error())
	}
}

type Client struct {
	Phost string
	Pmap  map[byte]string
}

var tcpclient tcp.Client = initTcpCli()
var sessionMgr *session.SessionMgr = session.NewSessionMgr("umsCookie", 3600)

func initTcpCli() tcp.Client {
	client, _ := tcp.NewClient(Client{
		Phost: "127.0.0.1:9095",
		Pmap: map[byte]string{
			0x01: "dologin",
			0x02: "doRegister",
		},
	})

	return client
}

func InitRedis() {
	var err error
	conf_file := flag.String("configclient", "./configs/config.ini", "set redis config file.")

	l_conf, err := goconf.ReadConfigFile(*conf_file)
	if err != nil {
		log.Print("LoadConfiguration: Error: Could not open config file：", conf_file, err)
		os.Exit(1)
	}
	redisCliPool.InitRedisPool(l_conf)
	//	defer redisCliPool.Clipool.Close()
}

var userDao *dao.UserDao = new(dao.UserDao)

func Register() {
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads")))) //用来访问静态文件
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/doLogin.do", doLoginHandler)
	http.HandleFunc("/register.do", registerHandler)
	http.HandleFunc("/doRegister.do", doRegisterHandler)
	http.HandleFunc("/updateUser.do", updateUserHandler)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

//index
func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("indexHandler...")
	tmpl, err := template.ParseFiles("src/web/index.html")
	checkErr(err)
	err = tmpl.Execute(w, nil)
	checkErr(err)
}

var loginch = make(chan model.Result, 1)
var registercg = make(chan model.Result)
var aesEnc *common.AesEncrypt = new(common.AesEncrypt)

//doLogin
func doLoginHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("src/web/login.html")
	checkErr(err)
	name := r.FormValue("name")
	password := r.FormValue("password")
	var user model.User
	user.Name = name
	user.Password = password
	value, err := json.Marshal(user)
	tcpclient.Send(0x01, value)
	ch := <-loginch
	if ch.IsOk {
		arrEncrypt, err := aesEnc.Encrypt(ch.User.Name + ch.User.Password + uuid.NewV1().String())
		if err != nil {
			fmt.Println(arrEncrypt)
			return
		}
		//登录成功设置session
		var sessionID = sessionMgr.StartSession(w, r, string(arrEncrypt))
		//踢除重复登录的,这里用map会更好，不用每次都去循环
		var onlineSessionIDList = sessionMgr.GetSessionIDList()
		for _, onlineSessionID := range onlineSessionIDList {
			if userInfo, ok := sessionMgr.GetSessionVal(onlineSessionID, "user"); ok {
				if value, ok := userInfo.(model.User); ok {
					if value.Id == ch.User.Id {
						sessionMgr.EndSessionBy(onlineSessionID)
					}
				}
			}
		}
		//设置当前用户的session值,应该存入redis，简易做法
		sessionMgr.SetSessionVal(sessionID, "user", ch.User)
		err = tmpl.Execute(w, map[string]interface{}{"user": ch.User})
	} else {
		err = tmpl.Execute(w, map[string]string{"err_msg": "username or password is invalid!"})
		checkErr(err)
	}
	checkErr(err)
	log.Print("loginHandler...  " + name + "  " + password)
	log.Print("loginHandler  over")
}

//登录请求tcp回调函数
func (c Client) Pdologin(in []byte) {
	fmt.Printf("收到了dologin包的回复:%s\n", in)
	var result model.Result
	err := json.Unmarshal(in, &result)
	fmt.Printf("pdologin err unmarshal:", err)
	fmt.Printf("Pdologin  before", len(loginch))
	loginch <- result
	fmt.Printf("Pdologin  after", len(loginch))
	fmt.Printf("pdologin over:")
}

//register
func registerHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("src/web/register.html")
	checkErr(err)
	err = tmpl.Execute(w, nil)
	checkErr(err)
}

func (c Client) PdoRegister(in []byte) {
	fmt.Printf("收到了doregister包的回复:%s\n", in)
	var result model.Result
	json.Unmarshal(in, result)
	//	loginch <- true
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

	u2 := uuid.NewV1()
	fmt.Printf("Successfully parsed: %s", u2)
	filedir := UPLOAD_DIR + "/" + u2.String() //这里可以给文件按照用户特性或者全局唯一id的方式重命名以免不同用户传入相同的文件名

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

	sql := "INSERT INTO userinfo(name, password, profile, nickname, createtime) VALUES (?,?,?,?,?)"
	log.Print(sql)
	i := userDao.Save(sql, name, password, filedir, nickname, time.Now().Unix())
	if i > 0 {
		tmpl, err := template.ParseFiles("src/web/registered.html")
		checkErr(err)
		err = tmpl.Execute(w, nil)
		checkErr(err)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

//updateUserHandler
//doRegister和此方法有大量代码重用，可以优化
func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("updateUserHandler...")
	var sessionID = sessionMgr.CheckCookieValid(w, r)

	if sessionID == "" {
		http.Redirect(w, r, "/doLogin.do", http.StatusFound)
		return
	}
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
		u2, err := uuid.FromString(name)
		if err != nil {
			fmt.Printf("Something gone wrong: %s", err)
		}
		fmt.Printf("Successfully parsed: %s", u2)
		filedir := UPLOAD_DIR + "/" + u2.String() //这里可以给文件按照用户特性或者全局唯一id的方式重命名以免不同用户传入相同的文件名
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
		sql = "UPDATE userinfo SET nickname='" + nickname + "',profile='" + filedir + "' where id ='" + id + "';"
		defer f.Close()
	}

	sql = "UPDATE userinfo SET nickname='" + nickname + "' where id ='" + id + "';"
	log.Print(sql)
	userDao.Save(sql)
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
