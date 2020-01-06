package util

import (
	"config_server/lib/stlog"
	"io/ioutil"
	"net/http"
)

func get_external_ip() (ip string) {
	ip = ""
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		stlog.Error("get_external_ip", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	ip = string(body)
	return
}
