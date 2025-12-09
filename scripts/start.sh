#!/bin/bash

# Mempool Sniper 启动脚本
# 作者: Mempool Sniper Team
# 版本: 1.0.0

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.21 或更高版本"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "检测到 Go 版本: $GO_VERSION"
    
    if ! command -v git &> /dev/null; then
        log_warn "Git 未安装，某些功能可能受限"
    fi
}

# 检查环境文件
check_env_file() {
    if [ ! -f ".env" ]; then
        log_warn "未找到 .env 文件，使用默认配置"
        if [ -f ".env.example" ]; then
            log_info "请复制 .env.example 为 .env 并配置您的参数"
        fi
    else
        log_success "找到环境配置文件"
    fi
}

# 下载依赖
download_dependencies() {
    log_info "下载项目依赖..."
    go mod download
    if [ $? -eq 0 ]; then
        log_success "依赖下载完成"
    else
        log_error "依赖下载失败"
        exit 1
    fi
}

# 构建项目
build_project() {
    log_info "构建项目..."
    go build -o mempool-sniper .
    if [ $? -eq 0 ]; then
        log_success "项目构建成功"
    else
        log_error "项目构建失败"
        exit 1
    fi
}

# 运行项目
run_project() {
    log_info "启动 Mempool Sniper..."
    
    # 检查是否已构建
    if [ ! -f "mempool-sniper" ]; then
        log_warn "未找到可执行文件，重新构建..."
        build_project
    fi
    
    # 设置环境变量
    export GO_ENV=${GO_ENV:-"production"}
    
    # 启动程序
    ./mempool-sniper
}

# 显示帮助信息
show_help() {
    echo "Mempool Sniper 启动脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help     显示此帮助信息"
    echo "  -d, --dev      开发模式运行"
    echo "  -b, --build    仅构建项目"
    echo "  -c, --clean    清理构建文件"
    echo ""
    echo "示例:"
    echo "  $0              # 正常启动"
    echo "  $0 --dev        # 开发模式启动"
    echo "  $0 --build      # 仅构建项目"
}

# 清理构建文件
clean_project() {
    log_info "清理构建文件..."
    rm -f mempool-sniper
    log_success "清理完成"
}

# 主函数
main() {
    case "${1:--h}" in
        -h|--help)
            show_help
            ;;
        -d|--dev)
            log_info "开发模式启动..."
            export GO_ENV="development"
            check_dependencies
            check_env_file
            download_dependencies
            run_project
            ;;
        -b|--build)
            check_dependencies
            download_dependencies
            build_project
            ;;
        -c|--clean)
            clean_project
            ;;
        *)
            check_dependencies
            check_env_file
            download_dependencies
            build_project
            run_project
            ;;
    esac
}

# 脚本入口
main "$@"