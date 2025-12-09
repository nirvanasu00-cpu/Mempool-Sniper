#!/bin/bash

# Mempool Sniper 测试脚本
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

# 运行单元测试
run_unit_tests() {
    log_info "运行单元测试..."
    
    # 测试配置模块
    log_info "测试配置模块..."
    go test ./internal/config/... -v
    if [ $? -ne 0 ]; then
        log_error "配置模块测试失败"
        return 1
    fi
    
    # 测试类型模块
    log_info "测试类型模块..."
    go test ./pkg/types/... -v
    if [ $? -ne 0 ]; then
        log_error "类型模块测试失败"
        return 1
    fi
    
    log_success "单元测试完成"
}

# 运行集成测试
run_integration_tests() {
    log_info "运行集成测试..."
    
    # 检查环境变量
    if [ -z "$ETH_WSS_URL" ] || [ -z "$ETH_RPC_URL" ]; then
        log_warn "未设置以太坊节点URL，跳过集成测试"
        return 0
    fi
    
    # 构建测试二进制文件
    log_info "构建测试程序..."
    go build -o test-sniper examples/basic_usage.go
    if [ $? -ne 0 ]; then
        log_error "测试程序构建失败"
        return 1
    fi
    
    # 运行短时间测试
    log_info "运行集成测试(30秒)..."
    timeout 30s ./test-sniper > test-output.log 2>&1 &
    TEST_PID=$!
    
    # 等待测试完成
    wait $TEST_PID
    TEST_EXIT_CODE=$?
    
    # 检查测试结果
    if [ $TEST_EXIT_CODE -eq 124 ]; then
        log_success "集成测试超时(正常行为)"
    elif [ $TEST_EXIT_CODE -eq 0 ]; then
        log_success "集成测试完成"
    else
        log_error "集成测试失败，退出码: $TEST_EXIT_CODE"
        cat test-output.log
        return 1
    fi
    
    # 清理测试文件
    rm -f test-sniper test-output.log
}

# 运行代码检查
run_code_checks() {
    log_info "运行代码检查..."
    
    # 检查代码格式
    log_info "检查代码格式..."
    go fmt ./...
    if [ $? -ne 0 ]; then
        log_error "代码格式检查失败"
        return 1
    fi
    
    # 运行静态分析
    log_info "运行静态分析..."
    go vet ./...
    if [ $? -ne 0 ]; then
        log_error "静态分析发现问题"
        return 1
    fi
    
    log_success "代码检查完成"
}

# 运行性能测试
run_performance_tests() {
    log_info "运行性能测试..."
    
    # 构建性能测试程序
    go build -o perf-test .
    if [ $? -ne 0 ]; then
        log_error "性能测试程序构建失败"
        return 1
    fi
    
    # 运行基准测试
    log_info "运行基准测试..."
    go test -bench=. -benchmem ./internal/decoder/...
    go test -bench=. -benchmem ./internal/simulator/...
    
    # 清理
    rm -f perf-test
    
    log_success "性能测试完成"
}

# 生成测试报告
generate_test_report() {
    log_info "生成测试报告..."
    
    # 创建测试报告目录
    mkdir -p test-reports
    
    # 运行测试并生成报告
    go test ./... -v -coverprofile=test-reports/coverage.out
    
    # 生成HTML覆盖率报告
    go tool cover -html=test-reports/coverage.out -o test-reports/coverage.html
    
    log_success "测试报告生成完成"
    log_info "覆盖率报告: test-reports/coverage.html"
}

# 显示帮助信息
show_help() {
    echo "Mempool Sniper 测试脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help         显示此帮助信息"
    echo "  -u, --unit         仅运行单元测试"
    echo "  -i, --integration  仅运行集成测试"
    echo "  -c, --check        仅运行代码检查"
    echo "  -p, --performance  仅运行性能测试"
    echo "  -r, --report       生成测试报告"
    echo "  -a, --all          运行所有测试(默认)"
    echo ""
    echo "示例:"
    echo "  $0                  # 运行所有测试"
    echo "  $0 --unit           # 仅运行单元测试"
    echo "  $0 --report         # 生成测试报告"
}

# 主函数
main() {
    case "${1:--a}" in
        -h|--help)
            show_help
            ;;
        -u|--unit)
            run_unit_tests
            ;;
        -i|--integration)
            run_integration_tests
            ;;
        -c|--check)
            run_code_checks
            ;;
        -p|--performance)
            run_performance_tests
            ;;
        -r|--report)
            generate_test_report
            ;;
        -a|--all)
            run_code_checks
            run_unit_tests
            run_integration_tests
            run_performance_tests
            generate_test_report
            ;;
        *)
            echo "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
}

# 脚本入口
main "$@"