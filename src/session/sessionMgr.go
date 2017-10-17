package session

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

//Session会话管理，应该做单独的session服务，目前方案只适合单机内存
type SessionMgr struct {
	mCookieName  string //客户端cookie名称
	mLock        sync.RWMutex
	mMaxLifeTime int64 //垃圾回收时间
	mSessions    map[string]*Session
}

//session
type Session struct {
	mSessionID        string                      //唯一id
	mLastTimeAccessed time.Time                   //最后访问时间
	mValues           map[interface{}]interface{} //应该放入redis等，暂时用本机内存
}

//创建会话管理器(cookieName:在浏览器中cookie的名字;maxLifeTime:最长生命周期)
func NewSessionMgr(cookieName string, maxLifeTime int64) *SessionMgr {
	mgr := &SessionMgr{
		mCookieName:  cookieName,
		mMaxLifeTime: maxLifeTime,
		mLock:        *new(sync.RWMutex),
		mSessions:    make(map[string]*Session)}

	//启动定时回收
	go time.AfterFunc(time.Duration(mgr.mMaxLifeTime)*time.Second, func() { mgr.GC() })

	return mgr
}

//在开始页面登陆页面，开始Session
//需要传入新建的session
func (mgr *SessionMgr) StartSession(w http.ResponseWriter, r *http.Request, s string) string {
	mgr.mLock.Lock()
	defer mgr.mLock.Unlock()

	//无论原来有没有，都重新创建一个新的session
	newSessionID := url.QueryEscape(s)

	//存指针
	var session *Session = &Session{mSessionID: newSessionID, mLastTimeAccessed: time.Now(), mValues: make(map[interface{}]interface{})}
	mgr.mSessions[newSessionID] = session
	//让浏览器cookie设置过期时间
	cookie := http.Cookie{Name: mgr.mCookieName, Value: newSessionID, Path: "/", HttpOnly: true, MaxAge: int(mgr.mMaxLifeTime)}
	http.SetCookie(w, &cookie)

	return newSessionID
}

//结束Session
func (mgr *SessionMgr) EndSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(mgr.mCookieName)
	if err != nil || cookie.Value == "" {
		return
	} else {
		mgr.EndSessionBy(cookie.Value)
		//让浏览器cookie立刻过期
		expiration := time.Now()
		cookie := http.Cookie{Name: mgr.mCookieName, Path: "/", HttpOnly: true, Expires: expiration, MaxAge: -1}
		http.SetCookie(w, &cookie)
	}
}

//结束session
func (mgr *SessionMgr) EndSessionBy(sessionID string) {
	mgr.mLock.Lock()
	defer mgr.mLock.Unlock()

	delete(mgr.mSessions, sessionID)
}

//设置session里面的值
func (mgr *SessionMgr) SetSessionVal(sessionID string, key interface{}, value interface{}) {
	mgr.mLock.Lock()
	defer mgr.mLock.Unlock()

	if session, ok := mgr.mSessions[sessionID]; ok {
		session.mValues[key] = value
	}
}

//得到session里面的值
func (mgr *SessionMgr) GetSessionVal(sessionID string, key interface{}) (interface{}, bool) {
	mgr.mLock.RLock()
	defer mgr.mLock.RUnlock()

	if session, ok := mgr.mSessions[sessionID]; ok {
		if val, ok := session.mValues[key]; ok {
			return val, ok
		}
	}

	return nil, false
}

//得到sessionID列表
func (mgr *SessionMgr) GetSessionIDList() []string {
	mgr.mLock.RLock()
	defer mgr.mLock.RUnlock()

	sessionIDList := make([]string, 0)

	for k, _ := range mgr.mSessions {
		sessionIDList = append(sessionIDList, k)
	}

	return sessionIDList[0:len(sessionIDList)]
}

//判断Cookie的合法性（每进入一个页面都需要判断合法性）,不知道go有没有aop的概念，就不需要在每个页面请求的时候写同样的代码
func (mgr *SessionMgr) CheckCookieValid(w http.ResponseWriter, r *http.Request) string {
	var cookie, err = r.Cookie(mgr.mCookieName)

	if cookie == nil ||
		err != nil {
		return ""
	}

	mgr.mLock.Lock()
	defer mgr.mLock.Unlock()

	sessionID := cookie.Value

	if session, ok := mgr.mSessions[sessionID]; ok {
		session.mLastTimeAccessed = time.Now() //判断合法性的同时，更新最后的访问时间
		return sessionID
	}

	return ""
}

//GC回收
func (mgr *SessionMgr) GC() {
	mgr.mLock.Lock()
	defer mgr.mLock.Unlock()

	for sessionID, session := range mgr.mSessions {
		//删除超过时限的session
		if session.mLastTimeAccessed.Unix()+mgr.mMaxLifeTime < time.Now().Unix() {
			delete(mgr.mSessions, sessionID)
		}
	}
}

//
////创建唯一ID，这种方式在大流量的情况下很有可能重复，也不利于检验合法性，应该根据用户的各种信息加密之后生成一个session
//func (mgr *SessionMgr) NewSessionID() string {
//	return uuid.NewV1().String()
//}
