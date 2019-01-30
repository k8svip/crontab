package worker

import (
	"encoding/json"
	"io/ioutil"
)

// 程序配置
type Config struct {
	EtcdEndpoints   []string `json:"etcdEndpoints"`
	EtcdDialTimeout int      `json:"etcdDialTimeout"`
}

var (
	G_config *Config
)

// 加载配置
func InitConfig(filename string) (err error) {

	var (
		content []byte
		conf    Config
	)
	// 1. 把配置文件读起来

	if content, err = ioutil.ReadFile(filename); err != nil {
		return
	}
	// 2. 反序列化
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	// 3. 赋值单例

	G_config = &conf
	return

}