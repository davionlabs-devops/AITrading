#!/bin/bash

# NOFX 安全加固脚本
# 用于私有化部署前的安全配置

set -e

echo "🔒 NOFX 安全加固脚本"
echo "===================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 1. 检查并设置文件权限
print_info "设置文件权限..."
chmod 600 config.json 2>/dev/null || print_warning "config.json 不存在，跳过"
chmod 600 config.db 2>/dev/null || print_warning "config.db 不存在，跳过"
chmod 700 decision_logs/ 2>/dev/null || print_warning "decision_logs/ 不存在，跳过"
print_success "文件权限设置完成"

# 2. 检查JWT密钥
print_info "检查JWT密钥配置..."
if [ -z "$JWT_SECRET" ]; then
    print_warning "JWT_SECRET 环境变量未设置"
    print_info "生成随机JWT密钥..."
    JWT_SECRET=$(openssl rand -base64 64 2>/dev/null || python3 -c "import secrets; print(secrets.token_urlsafe(64))" 2>/dev/null || echo "")
    if [ -z "$JWT_SECRET" ]; then
        print_error "无法生成JWT密钥，请手动设置"
    else
        print_success "已生成JWT密钥"
        print_info "请将以下内容添加到 .env 文件:"
        echo "JWT_SECRET=$JWT_SECRET"
        echo ""
    fi
else
    print_success "JWT_SECRET 环境变量已设置"
fi

# 3. 检查CORS配置
print_info "检查CORS配置..."
if [ -z "$CORS_ALLOWED_ORIGIN" ]; then
    print_warning "CORS_ALLOWED_ORIGIN 环境变量未设置"
    print_info "默认将只允许 localhost:3000 访问"
    print_info "如需允许其他源，请设置环境变量:"
    echo "export CORS_ALLOWED_ORIGIN='http://your-domain.com'"
    echo ""
else
    print_success "CORS_ALLOWED_ORIGIN 环境变量已设置: $CORS_ALLOWED_ORIGIN"
fi

# 4. 检查管理员模式
print_info "检查管理员模式配置..."
if grep -q '"admin_mode":\s*true' config.json 2>/dev/null; then
    print_warning "管理员模式已启用（admin_mode: true）"
    print_warning "生产环境建议禁用管理员模式"
    read -p "是否禁用管理员模式？(y/n): " disable_admin
    if [ "$disable_admin" = "y" ] || [ "$disable_admin" = "Y" ]; then
        # 使用sed修改config.json（需要jq或手动编辑）
        print_info "请手动编辑 config.json，设置 admin_mode: false"
    fi
else
    print_success "管理员模式已禁用或未配置"
fi

# 5. 检查配置文件是否在.gitignore中
print_info "检查 .gitignore 配置..."
if grep -q "config.json" .gitignore 2>/dev/null && grep -q "config.db" .gitignore 2>/dev/null; then
    print_success "敏感文件已在 .gitignore 中"
else
    print_warning "请确保 config.json 和 config.db 在 .gitignore 中"
fi

# 6. 生成.env示例文件
if [ ! -f .env.example ]; then
    print_info "创建 .env.example 文件..."
    cat > .env.example << 'EOF'
# JWT密钥（必须修改！）
JWT_SECRET=your_very_long_and_random_secret_key_min_64_chars_here

# CORS允许的源
# 单个源: CORS_ALLOWED_ORIGIN=http://your-domain.com
# 多个源: CORS_ALLOWED_ORIGIN=http://domain1.com,http://domain2.com
# 开发环境: CORS_ALLOWED_ORIGIN=http://localhost:3000
CORS_ALLOWED_ORIGIN=http://localhost:3000
EOF
    print_success "已创建 .env.example 文件"
    print_info "请复制 .env.example 为 .env 并填写实际值:"
    echo "cp .env.example .env"
    echo "nano .env"
fi

# 7. 检查防火墙（如果可用）
print_info "检查防火墙配置..."
if command -v ufw &> /dev/null; then
    print_info "检测到 UFW 防火墙"
    print_info "建议配置防火墙规则:"
    echo "  sudo ufw allow from 127.0.0.1 to any port 8080"
    echo "  sudo ufw deny 8080"
    echo "  sudo ufw allow from 127.0.0.1 to any port 3000"
    echo "  sudo ufw deny 3000"
elif command -v firewall-cmd &> /dev/null; then
    print_info "检测到 firewalld"
    print_info "建议配置防火墙规则"
elif [ "$(uname)" = "Darwin" ]; then
    print_info "macOS 系统，建议使用 pfctl 配置防火墙"
else
    print_warning "未检测到防火墙工具，请手动配置"
fi

# 8. 安全建议总结
echo ""
echo "════════════════════════════════════════════════════════"
print_success "安全加固检查完成！"
echo ""
print_info "📋 下一步操作:"
echo "  1. 设置环境变量（创建 .env 文件）"
echo "  2. 修改 config.json（禁用管理员模式，生产环境）"
echo "  3. 配置防火墙规则"
echo "  4. 如果公网部署，配置HTTPS和反向代理"
echo ""
print_info "📚 详细文档:"
echo "  - 安全审计报告: docs/SECURITY_AUDIT_REPORT.md"
echo "  - 部署安全指南: docs/DEPLOYMENT_SECURITY_GUIDE.md"
echo ""


