# 私有化部署安全指南

## 🎯 快速开始

本指南将帮助您安全地私有化部署 NOFX AI 交易系统。

---

## ⚠️ 部署前必读

**重要**: 在部署到生产环境之前，必须完成以下安全检查：

1. ✅ 修改JWT密钥
2. ✅ 配置CORS
3. ✅ 设置文件权限
4. ✅ 配置防火墙
5. ✅ 禁用管理员模式（生产环境）

---

## 🔧 安全配置步骤

### 步骤 1: 设置环境变量

创建 `.env` 文件（不要提交到版本控制）：

```bash
# JWT密钥（必须修改！）
JWT_SECRET=your_very_long_and_random_secret_key_min_64_chars_here

# CORS允许的源（生产环境必须配置）
# 单个源: CORS_ALLOWED_ORIGIN=http://your-domain.com
# 多个源: CORS_ALLOWED_ORIGIN=http://domain1.com,http://domain2.com
# 开发环境: CORS_ALLOWED_ORIGIN=http://localhost:3000
CORS_ALLOWED_ORIGIN=http://localhost:3000
```

**生成安全的JWT密钥**:
```bash
# 方法1: 使用openssl
openssl rand -base64 64

# 方法2: 使用Python
python3 -c "import secrets; print(secrets.token_urlsafe(64))"

# 方法3: 使用Go
go run -c "package main; import (\"crypto/rand\"; \"encoding/base64\"; \"fmt\"); func main() { b := make([]byte, 64); rand.Read(b); fmt.Println(base64.URLEncoding.EncodeToString(b)) }"
```

### 步骤 2: 设置文件权限

```bash
# 设置配置文件权限（只有所有者可读）
chmod 600 config.json
chmod 600 config.db

# 设置日志目录权限
chmod 700 decision_logs/

# 设置可执行文件权限
chmod 755 nofx
```

### 步骤 3: 配置防火墙

#### Linux (UFW)

```bash
# 只允许本地访问API（推荐）
sudo ufw allow from 127.0.0.1 to any port 8080
sudo ufw deny 8080

# 只允许本地访问Web界面
sudo ufw allow from 127.0.0.1 to any port 3000
sudo ufw deny 3000

# 如果需要在局域网访问，允许特定IP
sudo ufw allow from 192.168.1.0/24 to any port 8080
sudo ufw allow from 192.168.1.0/24 to any port 3000
```

#### macOS (pfctl)

```bash
# 创建防火墙规则文件 /etc/pf.anchors/nofx
# 只允许本地访问
block in on en0 proto tcp from any to any port 8080
pass in on lo0 proto tcp from 127.0.0.1 to any port 8080

block in on en0 proto tcp from any to any port 3000
pass in on lo0 proto tcp from 127.0.0.1 to any port 3000
```

### 步骤 4: 修改配置文件

编辑 `config.json`:

```json
{
  "admin_mode": false,
  "beta_mode": false,
  "jwt_secret": "",
  "api_server_port": 8080
}
```

**重要**:
- `admin_mode`: 生产环境必须设置为 `false`
- `jwt_secret`: 留空，使用环境变量 `JWT_SECRET`
- 不要将真实的API密钥写入配置文件

### 步骤 5: 使用环境变量加载配置

修改启动脚本，加载环境变量：

```bash
#!/bin/bash
# start_nofx.sh

# 加载环境变量
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# 启动服务
./nofx
```

---

## 🚀 部署场景配置

### 场景 1: 本地开发环境

**配置**:
```bash
# .env
JWT_SECRET=dev_secret_key_here
CORS_ALLOWED_ORIGIN=http://localhost:3000
```

**config.json**:
```json
{
  "admin_mode": true,
  "api_server_port": 8080
}
```

**防火墙**: 不需要（仅本地访问）

---

### 场景 2: 内网部署（局域网）

**配置**:
```bash
# .env
JWT_SECRET=production_secret_key_min_64_chars
CORS_ALLOWED_ORIGIN=http://192.168.1.100:3000,http://nofx.local:3000
```

**config.json**:
```json
{
  "admin_mode": false,
  "api_server_port": 8080
}
```

**防火墙**: 只允许内网IP访问

---

### 场景 3: 公网部署（互联网）

**配置**:
```bash
# .env
JWT_SECRET=very_secure_production_secret_key_min_64_chars
CORS_ALLOWED_ORIGIN=https://your-domain.com
```

**config.json**:
```json
{
  "admin_mode": false,
  "api_server_port": 8080
}
```

**必须完成**:
1. ✅ 使用HTTPS（nginx反向代理 + SSL证书）
2. ✅ 配置防火墙（只允许特定IP）
3. ✅ 启用2FA（强制所有用户）
4. ✅ 设置API限流
5. ✅ 定期安全审计

**Nginx配置示例**:
```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # 限制访问IP
    allow 1.2.3.4;  # 您的IP
    deny all;
    
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    location /api {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 🔐 数据库安全

### 当前状态

- ✅ 密码使用bcrypt哈希（安全）
- ⚠️ API密钥明文存储（建议加密）

### 加密敏感字段（可选）

如果需要加密数据库中的敏感字段，可以：

1. **使用SQLCipher**（SQLite加密扩展）
2. **应用层加密**（在存储前加密，读取时解密）

**示例代码**（应用层加密）:
```go
// 加密函数
func encryptSensitiveData(data string) (string, error) {
    key := os.Getenv("DB_ENCRYPTION_KEY")
    // 使用AES-256-GCM加密
    // ...
}

// 解密函数
func decryptSensitiveData(encrypted string) (string, error) {
    key := os.Getenv("DB_ENCRYPTION_KEY")
    // 解密
    // ...
}
```

---

## 🛡️ 额外安全措施

### 1. API限流

防止暴力破解和DoS攻击：

```go
// 使用中间件限制API调用频率
// 例如：每个IP每分钟最多100次请求
```

### 2. IP白名单

限制Web界面访问来源：

```nginx
# nginx配置
location / {
    allow 192.168.1.0/24;  # 内网
    allow 1.2.3.4;         # 特定IP
    deny all;
}
```

### 3. 安全审计日志

记录所有敏感操作：

```go
// 记录所有API调用
log.Printf("[AUDIT] User %s accessed %s from %s", userID, endpoint, ip)
```

### 4. 定期备份

```bash
#!/bin/bash
# backup.sh

# 备份数据库（加密）
tar czf backup-$(date +%Y%m%d-%H%M%S).tar.gz config.db
gpg --encrypt --recipient your@email.com backup-*.tar.gz

# 删除未加密的备份
rm backup-*.tar.gz

# 保留最近7天的备份
find . -name "backup-*.tar.gz.gpg" -mtime +7 -delete
```

---

## ✅ 部署检查清单

### 部署前

- [ ] 生成并设置 `JWT_SECRET` 环境变量
- [ ] 配置 `CORS_ALLOWED_ORIGIN` 环境变量
- [ ] 设置 `config.json` 和 `config.db` 文件权限为 600
- [ ] 设置 `decision_logs/` 目录权限为 700
- [ ] 生产环境设置 `admin_mode: false`
- [ ] 配置防火墙规则
- [ ] 检查 `.gitignore` 确保敏感文件不被提交

### 部署后

- [ ] 验证API只能从允许的源访问
- [ ] 测试JWT认证是否正常工作
- [ ] 确认管理员模式已禁用（生产环境）
- [ ] 检查日志中是否有敏感信息泄露
- [ ] 测试防火墙规则是否生效
- [ ] 设置监控和告警

### 定期维护

- [ ] 定期更新依赖项
- [ ] 定期检查安全公告
- [ ] 定期轮换API密钥
- [ ] 定期备份数据库
- [ ] 定期审查访问日志

---

## 🚨 常见安全问题

### 问题 1: JWT密钥泄露

**症状**: 攻击者可以伪造JWT token

**解决**:
1. 立即更换JWT密钥
2. 所有用户需要重新登录
3. 检查是否有未授权访问

### 问题 2: CORS配置错误

**症状**: 其他网站可以访问您的API

**解决**:
1. 立即修复CORS配置
2. 检查访问日志
3. 撤销可能泄露的token

### 问题 3: 数据库文件泄露

**症状**: 所有API密钥和配置暴露

**解决**:
1. 立即更换所有API密钥
2. 加密数据库文件
3. 检查文件权限

---

## 📞 安全支持

如果发现安全漏洞，请：
1. **不要**公开披露
2. 联系维护者：[@Web3Tinkle](https://x.com/Web3Tinkle)
3. 提供详细的漏洞描述

---

## 📚 相关文档

- [安全审计报告](./SECURITY_AUDIT_REPORT.md)
- [安全政策](../SECURITY.md)
- [部署指南](../docs/getting-started/README.md)

---

**最后更新**: 2025-11-06


