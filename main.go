package main

import (
	"config_server/lib/stlog"
	"config_server/lib/util"
	"encoding/json"
	"fmt"
	"config_server/lib/utils"
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
func main() {
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
	stlog.Initialize(logDir, oldLogDir, "configserver_", "log", log.Lshortfile|log.LstdFlags, 2, 10)
	stlog.SetLogLevel(stlog.LogDebug)
	stlog.SetOutConsole(true)
	ListenPort := "8090"
	f, err := os.OpenFile("./conf/cfg.json", os.O_RDONLY, 0666)
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
	log.Println("start success.")
	log.Println("config server address:", cfg.ListenAddr)
	go func() {
		http.HandleFunc("/GetConf", GetConf)
		log.Println("ListenAndServe  ListenIPPortCGI :" + ListenPort + "/GetConf")
		listenAddr := "0.0.0.0:" + ListenPort
		http.ListenAndServe(listenAddr, nil)
	}()
	go func() {
		http.HandleFunc("/GetSvrUdpAddr", GetUdpConf)
		log.Println("ListenAndServe  ListenIPPortCGI :" + ListenPort + "/GetSvrUdpAddr")
		listenAddr := "0.0.0.0:" + ListenPort
		http.ListenAndServe(listenAddr, nil)
	}()
	err = http.ListenAndServe(cfg.ListenAddr, nil)
	if err != nil {
		log.Println("%v", err)
	}
}
