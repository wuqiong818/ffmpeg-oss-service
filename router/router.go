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
		// 视频转音频端点
		video := v1.Group("/video")
		video.POST("/convert", handler.ConvertHandler)
	}

}
