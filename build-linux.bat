@echo off
chcp 65001 > nul

echo 开始构建Linux环境下的Go二进制文件...

:: 设置必要的环境变量
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64

:: 显示当前的构建环境配置
echo 构建环境配置:
echo CGO_ENABLED=%CGO_ENABLED%
echo GOOS=%GOOS%
echo GOARCH=%GOARCH%

:: 执行Go编译命令
echo 正在执行编译...
go build -o ffmpeg-oss-service

:: 检查编译结果
if %ERRORLEVEL% EQU 0 (
    echo 编译成功! Linux二进制文件已生成: ffmpeg-oss-service
) else (
    echo 编译失败! 请检查错误信息
)

pause