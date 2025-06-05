package main

import (
	"ffmpeg-oss-service/config"
	"ffmpeg-oss-service/router"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// 环境变量配置
type Config struct {
	OSSEndpoint        string
	OSSAccessKeyID     string
	OSSAccessKeySecret string
	OSSBucketName      string
	TempDir            string
}

func init() {
	// 加载配置文件
	config.LoadConfigFile("./etc/config.yaml")
	conf := config.Conf

	// 创建临时目录
	if err := os.MkdirAll(conf.Server.TempDir, 0755); err != nil {
		log.Fatalf("无法创建临时目录: %v", err)
	}
	// 设置随机种子
	rand.Seed(time.Now().UnixNano())
}

func main() {
	// 初始化 Gin
	engine := gin.Default()
	engine.MaxMultipartMemory = 4 << 30 // 4GB

	router.RegisterRouter(engine)

	// 启动服务器
	portInfo := fmt.Sprintf(":%d", config.Conf.Server.Port)
	fmt.Println(portInfo)
	engine.Run(portInfo)
}
