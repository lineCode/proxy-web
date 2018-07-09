package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"proxy-web/utils"
	"runtime"
	"strings"

	"github.com/astaxie/beego/session"
	"github.com/snail007/goproxy/sdk/android-ios"
	"golang.org/x/net/websocket"
)

var globalSessions *session.Manager
var version = "v2.0"
var lock = false
var sessionId string
var dir string

func basicAuth(handler func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, _ := globalSessions.SessionStart(w, r)
		newSessionId := sess.SessionID()
		if sessionId != newSessionId {
			login(w, r)
			return
		}
		handler(w, r)
	})
}

func StartServer() {
	// 文件路径
	dir, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	dir = strings.Replace(dir, "\\", "/", -1)

	// 启动一个websocket，判断是否有人登陆
	go StartWebscoket()
	SetProxy()
	AutoStart()
	InitShowLog()
	initSession()
	http.Handle("/static/", http.StripPrefix(dir + "/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", basicAuth(show))
	http.HandleFunc("/add", add)
	http.HandleFunc("/update", update)
	http.HandleFunc("/close", close)
	http.HandleFunc("/link", link)
	http.HandleFunc("/getData", getData)
	http.HandleFunc("/uploade", uploade)
	http.HandleFunc("/delete", deleteParameter)
	http.HandleFunc("/saveSetting", saveSetting)
	http.HandleFunc("/login", login)
	http.HandleFunc("/doLogin", doLogin)
	//http.Handle("/keygen", basicAuth(keygen))
	port, err := utils.NewConfig().GetServerPort()
	if err != nil {
		log.Fatal("get port failure: ", err)
	}
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("listen port failure", err)
	}
}

func AutoStart() {
	datas, err := utils.InitParams()
	if err != nil {
		return
	}
	for _, data := range datas {
		var command string
		command += data["command"].(string)
		command = strings.Replace(command, "\n", "", -1)
		command = strings.Replace(command, "\r", "", -1)
		command = strings.Replace(command, "  ", " ", -1)
		if data["key_file"].(string) != "" {
			command += " -K " + data["key_file"].(string)
		}
		if data["crt_file"].(string) != "" {
			command += " -C " + data["crt_file"].(string)
		}
		if data["log"] == "是" {
			command += " --log ./log/" + data["id"].(string) + ".log"
		}
		s, err := os.Stat("./log/")
		if err != nil || !s.IsDir() {
			os.Mkdir("./log/", os.ModePerm)
		}
		go autoRunCommand(data["id"].(string), command)
	}
}

func autoRunCommand(id, command string) {
	fmt.Println(command)
	errStr := proxy.Start(id, command)
	if errStr != "" {
		utils.ChangeParameterDataById(id, "未开启")
	}
}

func initSession() {
	sessionConfig := &session.ManagerConfig{
		CookieName:      "sessionid",
		EnableSetCookie: true,
		Gclifetime:      360000,
		Maxlifetime:     360000,
		Secure:          false,
		CookieLifeTime:  360000,
		ProviderConfig:  "./tmp",
	}
	globalSessions, _ = session.NewManager("file", sessionConfig)
	go globalSessions.GC()
}

func SetProxy() {
	data, err := utils.GetProxy()
	if err != nil {
		return
	}
	proxy := utils.NewConfig().GetProxySetting()
	if !proxy {
		return
	}
	switch runtime.GOOS {
	case "windows":
		addr := data["ip"] + ":" + data["port"]
		command := dir + "/config/proxysetting.exe http=" + addr + " https=" + addr
		fmt.Println(command)
		commandSlice := strings.Split(command, " ")
		cmd := exec.Command(commandSlice[0], commandSlice[1:]...)
		output, _ := cmd.CombinedOutput()
		fmt.Println(string(output))

	case "darwin":
	case "linux":
	}
}

func StartWebscoket() {
	http.Handle("/websocket", websocket.Handler(svrConnHandler))
	log.Fatal(http.ListenAndServe(":8222", nil))
}

func svrConnHandler(conn *websocket.Conn) {
	request := make([]byte, 128)
	defer conn.Close()
	readLen, err := conn.Read(request)
	if err != nil {
		return
	}

	if string(request[:readLen]) == "close" {
		lock = false
	} else {
		lock = true
	}

	request = make([]byte, 128)
}
