package config

import (
	"flag"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

var Conf Config

type Config struct {
	Server Server `json:"server"`
	OSS    OSS    `json:"oss"`
}

type OSS struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Location        string
	UseSSL          bool
}
type Server struct {
	Name    string `json:"name"`
	Mode    string `json:"mode"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Version string `json:"version"`
	TempDir string `json:"tempDir"`
}

// 热更新存在问题:需要进行回调，并且会执行多次
func LoadConfigFile(configPath string) {
	// 设置默认值
	var configFile string
	flag.StringVar(&configFile, "conf", configPath, "选择启动时加载的配置文件") //长短命令- -- 都可以
	flag.Parse()

	viper.SetConfigFile(configFile)
	fmt.Println("load config:", configFile)

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("ReadInConfig failed, err: %v", err)
	}

	if err := viper.Unmarshal(&Conf); err != nil {
		log.Fatalf("unmarshal to Conf failed, err:%v", err)
	}
}
