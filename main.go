package main

import (
	"config_server/lib/stlog"
	"config_server/lib/util"
	"encoding/json"
	"flag"
	"fmt"
	"idipserver/utils"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Config struct {
	ListenAddr string
	RootPath   string
	ChildPaths []string
}

var cfg *Config

func GetConf(rep http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		stlog.Error("GetConf", err)
	}
	if len(req.Form["key"]) == 0 || len(req.Form["ip"]) == 0 {
		stlog.Error("GetConf", "参数不正确")
	}
	key := req.Form["key"][0]
	ip := req.Form["ip"][0]
	idipConfStr := utils.ReadJsonFileAsString()
	if idipConfStr == "" {
		stlog.Error("访问 IDIP 配置表错误")
	}
	idipConf, err := utils.JsonStrToMap(idipConfStr)
	if err != nil {
		stlog.Error("配置文件解析失败", err)
	}
	resIdipconf := idipConf[key].(map[string]interface{})[ip]
	jsstr, _ := json.Marshal(resIdipconf)
	rep.Write(jsstr)
}
func GetUdpConf(rep http.ResponseWriter, req *http.Request) {
	key := "ZK_GMAEHLY_SEVER_SET"
	idipConfStr := utils.ReadJsonFileAsString()
	if idipConfStr == "" {
		stlog.Error("访问配置表错误")
	}
	idipConf, err := utils.JsonStrToMap(idipConfStr)
	if err != nil {
		stlog.Error("配置文件解析失败", err)
	}
	udpmap := make(map[string]string, 0)
	resIdipconf := idipConf[key].(map[string]interface{})
	index := 1
	for k, v := range resIdipconf {
		ukey := ""
		if index >= 10 {
			ukey = "0" + strconv.Itoa(index)
		} else {
			ukey = "00" + strconv.Itoa(index)
		}
		udpmap["UDP_SVR_SET_"+ukey] = k + v.(map[string]interface{})["UDP_PORT"].(string) + "|" + v.(map[string]interface{})["LOCAL_SERVER_TYPE"].(string)
		index++
	}
	udpMap := make(map[string]interface{})
	udpMap["UDP_SVR_SET"] = udpmap
	js, _ := json.Marshal(udpMap)
	rep.Write(js)
}

//管理服务器的配置文件
var ListenPort string

func main() {
	port := flag.String("p", "", "配置服务监听端口")
	flag.Parse() //解析输入的参数
	if (*port == "") {
		fmt.Printf("请输配置服务监听端口")
		fmt.Scanln(&ListenPort)
	} else {
		ListenPort = *port
	}
	if ListenPort == "" {
		return
	}
	path, err := util.GetCurrentPath()
	logDir := path + "log"
	oldLogDir := path + "oldlog"
	err = os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		log.Fatal("%v", err)
	}

	err = os.MkdirAll(oldLogDir, os.ModePerm)
	if err != nil {
		log.Fatal("%v", err)
	}
	stlog.Initialize(logDir, oldLogDir, "cs_", "log", log.Lshortfile|log.LstdFlags, 3600, 10)
	stlog.SetLogLevel(stlog.LogDebug)
	stlog.SetOutConsole(true)

	f, err := os.OpenFile("cfg.json", os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal("%v", err)
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal("%v", err)
	}

	cfg = &Config{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		log.Fatal("%v", err)
	}

	for i := 0; i < len(cfg.ChildPaths); i++ {
		path := cfg.ChildPaths[i]
		stlog.Info("dir:%s", path)
		http.Handle(
			fmt.Sprintf("/%s/", path),
			http.StripPrefix(
				fmt.Sprintf("/%s/", path),
				http.FileServer(http.Dir(fmt.Sprintf("%s./%s/", cfg.RootPath, path)))))
	}
	stlog.Info("start success.")
	log.Println("config server address:", cfg.ListenAddr)
	go func() {
		http.HandleFunc("/GetConf", GetConf)
		stlog.Info("ListenAndServe  ListenIPPortCGI :", ListenPort+"/GetConf")
		listenAddr := "0.0.0.0:" + ListenPort
		http.ListenAndServe(listenAddr, nil)
	}()
	go func() {
		http.HandleFunc("/GetSvrUdpAddr", GetUdpConf)
		stlog.Info("ListenAndServe  ListenIPPortCGI :", ListenPort+"/GetSvrUdpAddr")
		listenAddr := "0.0.0.0:" + ListenPort
		http.ListenAndServe(listenAddr, nil)
	}()
	err = http.ListenAndServe(cfg.ListenAddr, nil)
	if err != nil {
		stlog.Fatal("%v", err)
	}
}
