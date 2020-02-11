package utils

import (
	"io/ioutil"
	"encoding/json"
)

const CONF_PAHT = "./conf/app.conf"
const JSON_CONF_PATH = "./conf/web/config/eleidip/idip.json"

func ReadJsonFileAsString() (s string) {
	s = ""
	data, err := ioutil.ReadFile(JSON_CONF_PATH)
	if err != nil {
		return
	}
	s = string(data)
	return
}
func JsonStrToMap(jsonStr string) (dynamic map[string]interface{}, err error) {
	err = json.Unmarshal([]byte(jsonStr), &dynamic)
	return
}
