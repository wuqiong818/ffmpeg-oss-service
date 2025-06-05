package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
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

type ConvertResponse struct {
	AudioUrl       string  `json:"audioUrl"`
	SendFileSize   int64   `json:"sendfileSize"`
	AudioFileSize  int64   `json:"audioFileSize"`
	TotalDuration  float64 `json:"totalDuration"`
	ReceiveFileSec float64 `json:"receiveFileSec"`
	SaveFileSec    float64 `json:"saveFileSec"`
	ConvertFileSec float64 `json:"convertFileSec"`
	UploadFileSec  float64 `json:"uploadFileSec"`
}

// 转换处理函数
func ConvertUploadHandler(c *gin.Context) {
	log.Println("接收请求")
	startTime := time.Now()

	conf := config.Conf
	// 1. 接收上传的音视频文件
	fileReceiveStart := time.Now()
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
	fileReceiveDuration := time.Since(fileReceiveStart)

	fileName := header.Filename[:strings.LastIndex(header.Filename, ".")]
	log.Println("文件接收完毕,fileName = ", fileName)

	// 验证文件类型，如果是.mp4的视频就提取成音频；如果是音频的话，就直接进行上传操作。
	ext := filepath.Ext(header.Filename)
	if ext != ".mp4" && ext != ".aac" && ext != ".mp3" && ext != ".m4a" {
		c.JSON(http.StatusBadRequest, types.Response{
			Code:    types.ClientError,
			Message: types.FileFormatError,
			Data:    nil,
		})
		return
	}

	// 2. 保存为临时文件
	tempSaveFileStart := time.Now()
	videoPath := filepath.Join(conf.Server.TempDir, fileName+time.Now().Format("20060102150405")+randTwoDigits()+ext)
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
	saveFileDuration := time.Since(tempSaveFileStart)
	log.Println("保存为临时文件")

	var audioPath string
	var convertFileDuration time.Duration
	var objectName string
	if ext == ".mp4" {
		// 3. 使用 FFmpeg 转换
		audioPath = videoPath[:len(videoPath)-len(ext)] + ".aac"
		fmt.Println("audioPath = ", audioPath)
		convertFileDuration, err = convertToAAC(videoPath, audioPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.Response{
				Code:    types.ServerInternalError,
				Message: types.ConvertFileError + ":" + err.Error(),
				Data:    nil,
			})
			return
		}
		log.Println("转换成功")
		objectName = filepath.Base(audioPath)
		log.Println("objectName = ", objectName)
	} else {
		// 3.如果为音频文件的话，不进行任何处理，直接上传到OSS中
		audioPath = videoPath
		fmt.Println("audioPath = ", audioPath)
		objectName = filepath.Base(audioPath)
		log.Println("objectName = ", objectName)
	}
	defer os.Remove(audioPath)

	ctx := context.Background()

	// 4. 上传到 OSS
	audioURL, audioSize, uploadDuration, err := uploadToOSS(ctx, objectName, audioPath)
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
	c.JSON(http.StatusOK, ConvertResponse{
		AudioUrl:       audioURL,
		SendFileSize:   header.Size,
		AudioFileSize:  audioSize,
		TotalDuration:  totalTime,
		ReceiveFileSec: fileReceiveDuration.Seconds(),
		SaveFileSec:    saveFileDuration.Seconds(),
		ConvertFileSec: convertFileDuration.Seconds(),
		UploadFileSec:  uploadDuration.Seconds(),
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
func uploadToOSS(ctx context.Context, objectName, filePath string) (string, int64, time.Duration, error) {
	start := time.Now()

	ossConf := config.Conf.OSS
	minioClient, err := minio.New(ossConf.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(ossConf.AccessKeyID, ossConf.SecretAccessKey, ""),
		Secure: ossConf.UseSSL,
	})
	if err != nil {
		return "", 0, 0, err
	}

	// Upload file with FPutObject
	info, err := minioClient.FPutObject(ctx, ossConf.BucketName, objectName, filePath, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return "", 0, 0, err
	}
	audioURL := "http://" + ossConf.Endpoint + "/" + ossConf.BucketName + "/" + objectName
	return audioURL, info.Size, time.Since(start), nil
}

func randTwoDigits() string {
	// 生成0到99之间的随机整数
	n := rand.Intn(100) // 范围：[0, 100)，即0到99

	// 格式化为两位字符串（不足两位时前补0）
	return fmt.Sprintf("%02d", n)
}
