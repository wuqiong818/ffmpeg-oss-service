package router

import (
	"ffmpeg-oss-service/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRouter(router *gin.Engine) {
	v1 := router.Group("v1")

	{
		base := v1.Group("/base")
		// 健康检查端点
		base.GET("/health", handler.HealthHandler)
	}

	{
		// 视频转音频，并上传到OSS中
		video := v1.Group("/video")
		video.POST("/convert", handler.ConvertUploadHandler)
	}

	{
		// 将音频文件上传到OSS上
		oss := v1.Group("/oss")
		oss.POST("/upload")
	}

}
