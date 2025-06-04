package handler

import (
	"os/exec"

	"github.com/gin-gonic/gin"
)

func HealthHandler(c *gin.Context) {
	checkInfo := checkFFmpeg()
	c.JSON(200, gin.H{
		"status":  "ok",
		"version": "1.0.0",
		"ffmpeg":  checkInfo,
	})
}

// 检查 FFmpeg 是否可用
func checkFFmpeg() string {
	cmd := exec.Command("ffmpeg", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "ffmpeg服务不可用: " + err.Error()
	}
	if len(output) > 50 {
		return string(output[:50])
	}
	return string(output)
}
