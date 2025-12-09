@echo off
setlocal enabledelayedexpansion

REM Mempool Sniper Windows启动脚本
REM 作者: Mempool Sniper Team
REM 版本: 1.0.0

echo.
echo ========================================
echo    Mempool Sniper 启动脚本
echo ========================================
echo.

REM 检查Go是否安装
echo [INFO] 检查系统依赖...
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go未安装，请先安装Go 1.21或更高版本
    pause
    exit /b 1
)

for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
set GO_VERSION=!GO_VERSION:go=!
echo [INFO] 检测到Go版本: !GO_VERSION!

REM 检查环境文件
if not exist ".env" (
    echo [WARN] 未找到.env文件，使用默认配置
    if exist ".env.example" (
        echo [INFO] 请复制.env.example为.env并配置您的参数
    )
) else (
    echo [SUCCESS] 找到环境配置文件
)

REM 下载依赖
echo [INFO] 下载项目依赖...
go mod download
if errorlevel 1 (
    echo [ERROR] 依赖下载失败
    pause
    exit /b 1
)
echo [SUCCESS] 依赖下载完成

REM 构建项目
echo [INFO] 构建项目...
go build -o mempool-sniper.exe .
if errorlevel 1 (
    echo [ERROR] 项目构建失败
    pause
    exit /b 1
)
echo [SUCCESS] 项目构建成功

REM 设置环境变量
if "%GO_ENV%"=="" set GO_ENV=production

REM 启动程序
echo [INFO] 启动Mempool Sniper...
echo.
mempool-sniper.exe

if errorlevel 1 (
    echo [ERROR] 程序启动失败
    pause
    exit /b 1
)

echo.
echo [SUCCESS] 程序正常退出
pause