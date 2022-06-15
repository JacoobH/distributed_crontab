package worker

import (
	"encoding/json"
	"io/ioutil"
)

// Config Application configuration
type Config struct {
	EtcdEndPoints         []string `json:"etcdEndPoints"`
	EtcdDialTimeout       int      `json:"etcdDialTimeout"`
	MongodbUri            string   `json:"mongodbUri"`
	MongodbConnectTimeout int      `json:"mongodbConnectTimeout"`
	JobLogBatchSize       int      `json:"jobLogBatchSize"`
	JobLogCommitTimeout   int      `json:"jobLogCommitTimeout"`
}

var (
	// Set a global singleton config
	G_config *Config
)

// InitConfig load read configuration file
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

	return
}
