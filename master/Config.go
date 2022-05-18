package master

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Config Application configuration
type Config struct {
	ApiPort               string   `json:"apiPort"`
	EtcdEndPoints         []string `json:"etcdEndPoints"`
	EtcdDialTimeout       int      `json:"etcdDialTimeout"`
	Webroot               string   `json:"webroot"`
	StaticRelativePath    string   `json:"staticRelativePath"`
	StaticRoot            string   `json:"staticRoot"`
	MongodbUri            string   `json:"mongodbUri"`
	MongodbConnectTimeout int      `json:"mongodbConnectTimeout"`
}

var (
	G_config *Config
)

func InitConfig(filename string) (err error) {
	var (
		content []byte
		conf    Config
	)
	//  1.Read in the configuration file
	if content, err = ioutil.ReadFile(filename); err != nil {
		return
	}

	// 2.Json deserialization
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	// 3.Assignment singleton
	G_config = &conf

	fmt.Println(conf.ApiPort)

	return
}
