package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ffmpeg-oss-service/config"
	"ffmpeg-oss-service/types"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// 转换处理函数
func ConvertHandler(c *gin.Context) {
	log.Println("接收请求")
	startTime := time.Now()

	conf := config.Conf

	// 1. 接收上传的视频文件
	file, header, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.Response{
			Code:    types.ClientError,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}
	defer file.Close()

	fileName := header.Filename[:strings.LastIndex(header.Filename, ".")]
	fmt.Println("fileName = ", fileName)

	// 验证文件类型，目前仅支持.mp4的视频提取
	ext := filepath.Ext(header.Filename)
	if ext != ".mp4" {
		c.JSON(http.StatusBadRequest, types.Response{
			Code:    types.ClientError,
			Message: types.FileFormatError,
			Data:    nil,
		})
		return
	}

	// 2. 保存为临时文件
	videoPath := filepath.Join(conf.Server.TempDir, fileName+time.Now().Format("20060102150405")+ext)
	fmt.Println("videoPaht = ", videoPath)
	tempVideo, err := os.Create(videoPath) //创建一个空文件
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{
			Code:    types.ServerInternalError,
			Message: types.CreateFileError + ":" + err.Error(),
			Data:    nil,
		})
		return
	}
	defer os.Remove(videoPath)
	defer tempVideo.Close()

	if _, err := io.Copy(tempVideo, file); err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{
			Code:    types.ServerInternalError,
			Message: types.SaveFileError + ":" + err.Error(),
			Data:    nil,
		})
		return
	}
	log.Println("保存为临时文件")

	// 3. 使用 FFmpeg 转换
	audioPath := videoPath[:len(videoPath)-len(ext)] + ".aac"
	fmt.Println("audioPath = ", audioPath)
	convertTime, err := convertToAAC(videoPath, audioPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{
			Code:    types.ServerInternalError,
			Message: types.ConvertFileError + ":" + err.Error(),
			Data:    nil,
		})
		return
	}
	defer os.Remove(audioPath)
	log.Println("转换成功")

	ctx := context.Background()

	objectName := fileName + ".acc"

	// 4. 上传到 OSS
	audioURL, uploadTime, err := uploadToOSS(ctx, objectName, audioPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.Response{
			Code:    types.ServerInternalError,
			Message: types.UploadFileError + ":" + err.Error(),
			Data:    nil,
		})
		return
	}
	log.Println("上传oss")

	// 5. 返回结果
	totalTime := time.Since(startTime).Seconds()
	c.JSON(http.StatusOK, gin.H{
		"audio_url":   audioURL,
		"duration":    totalTime,
		"convert_sec": convertTime.Seconds(),
		"upload_sec":  uploadTime.Seconds(),
		"file_size":   header.Size,
	})
}

// 视频转 AAC 音频
func convertToAAC(videoPath, audioPath string) (time.Duration, error) {
	start := time.Now()

	cmd := exec.Command("ffmpeg",
		"-i", videoPath, // 输入文件
		"-vn",          // 禁用视频流
		"-c:a", "copy", // 直接复制音频流，不重新编码
		audioPath,
	)

	// 捕获错误输出
	var stderr []byte
	if output, err := cmd.CombinedOutput(); err != nil {
		if len(output) > 0 {
			stderr = output
		}
		return 0, fmt.Errorf("ffmpeg error: %v, %s", err, string(stderr))
	}

	return time.Since(start), nil
}

// 上传到 OSS , 并返回一个支持访问的URL,目前是上传到minio中
func uploadToOSS(ctx context.Context, objectName, filePath string) (string, time.Duration, error) {
	start := time.Now()

	ossConf := config.Conf.OSS
	minioClient, err := minio.New(ossConf.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(ossConf.AccessKeyID, ossConf.SecretAccessKey, ""),
		Secure: ossConf.UseSSL,
	})
	if err != nil {
		return "", 0, err
	}

	// Upload file with FPutObject
	_, err = minioClient.FPutObject(ctx, ossConf.BucketName, objectName, filePath, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return "", 0, err
	}
	audioURL := "http://" + ossConf.Endpoint + "/" + ossConf.BucketName + "/" + objectName
	return audioURL, time.Since(start), nil
}
