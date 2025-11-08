# 安全审计报告 - NOFX 私有化部署风险评估

**审计日期**: 2025-11-06  
**审计范围**: 完整代码库安全审查  
**审计目标**: 评估私有化部署的安全风险

---

## 📋 执行摘要

本报告对 NOFX AI 交易系统进行了全面的安全审计，识别了多个安全风险和需要改进的地方。总体而言，系统**可以私有化部署**，但需要采取一些安全加固措施。

### 风险等级总览

| 风险类型 | 严重程度 | 状态 | 建议 |
|---------|---------|------|------|
| JWT密钥硬编码 | 🔴 高 | ⚠️ 需要修复 | 使用环境变量 |
| CORS配置过宽 | 🟠 中 | ⚠️ 需要修复 | 限制允许的源 |
| 数据库未加密 | 🟠 中 | ⚠️ 建议改进 | 考虑加密敏感字段 |
| 文件权限过宽 | 🟡 低 | ⚠️ 建议改进 | 设置严格权限 |
| SQL注入风险 | ✅ 低 | ✅ 已防护 | 使用参数化查询 |
| 密码存储 | ✅ 良好 | ✅ 已加密 | 使用bcrypt |

---

## 🔴 高风险问题

### 1. JWT密钥硬编码在配置文件中

**位置**: `config.json`

```json
{
  "jwt_secret": "Qk0kAa+d0iIEzXVHXbNbm+UaN3RNabmWtH8rDWZ5OPf+4GX8pBflAHodfpbipVMyrw1fsDanHsNBjhgbDeK9Jg=="
}
```

**风险**:
- JWT密钥暴露在配置文件中
- 如果配置文件泄露，攻击者可以伪造JWT token
- 所有用户会话可能被劫持

**影响**: 🔴 **严重** - 可能导致未授权访问

**修复建议**:
1. ✅ 使用环境变量存储JWT密钥
2. ✅ 如果未设置环境变量，自动生成随机密钥
3. ✅ 确保配置文件不提交到版本控制（已在.gitignore中）

**修复代码示例**:
```go
// main.go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    // 生成随机密钥
    secret := make([]byte, 64)
    rand.Read(secret)
    jwtSecret = base64.StdEncoding.EncodeToString(secret)
    log.Printf("⚠️  警告: JWT_SECRET未设置，已自动生成新密钥")
}
auth.SetJWTSecret(jwtSecret)
```

---

### 2. CORS配置允许所有来源

**位置**: `api/server.go:55`

```go
c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
```

**风险**:
- 允许任何网站访问API
- 可能导致CSRF攻击
- 在私有化部署中，应该限制为特定域名

**影响**: 🟠 **中等** - 可能导致CSRF攻击

**修复建议**:
1. ✅ 使用环境变量配置允许的源
2. ✅ 默认只允许localhost（开发环境）
3. ✅ 生产环境必须配置具体的域名

**修复代码示例**:
```go
// 从环境变量读取允许的源
allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
if allowedOrigins == "" {
    allowedOrigins = "http://localhost:3000" // 默认只允许本地
}

c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
```

---

## 🟠 中风险问题

### 3. 数据库未加密存储敏感信息

**位置**: `config/database.go`

**风险**:
- API密钥、私钥等敏感信息以明文存储在SQLite数据库中
- 如果数据库文件泄露，所有密钥都会暴露
- 没有字段级别的加密

**影响**: 🟠 **中等** - 数据库泄露会导致密钥暴露

**当前状态**:
- ✅ 密码使用bcrypt哈希存储（安全）
- ❌ API密钥明文存储
- ❌ 私钥明文存储

**修复建议**:
1. ✅ 对敏感字段进行AES-256加密
2. ✅ 使用环境变量或密钥管理服务存储加密密钥
3. ✅ 在读取时自动解密，写入时自动加密

**实现建议**:
```go
// 加密函数
func encryptAPIKey(key string) (string, error) {
    // 使用AES-256-GCM加密
    // 密钥从环境变量获取
}

// 解密函数
func decryptAPIKey(encrypted string) (string, error) {
    // 解密API密钥
}
```

---

### 4. 配置文件权限过宽

**当前状态**:
```bash
-rw-r--r--@ ez staff config.json
-rw-r--r--@ ez staff config.db
```

**风险**:
- 文件权限644，所有用户可读
- 如果系统被其他用户访问，可能读取敏感配置

**影响**: 🟡 **低-中** - 取决于部署环境

**修复建议**:
```bash
# 设置严格的文件权限
chmod 600 config.json
chmod 600 config.db
```

---

### 5. 管理员模式绕过认证

**位置**: `api/server.go:1332`

```go
if auth.IsAdminMode() {
    c.Set("user_id", "admin")
    c.Set("email", "admin@localhost")
    c.Next()
    return
}
```

**风险**:
- 管理员模式下完全绕过JWT认证
- 如果API暴露到公网，任何人都可以访问

**影响**: 🟠 **中等** - 取决于部署方式

**修复建议**:
1. ✅ 管理员模式应该只用于本地开发
2. ✅ 生产环境必须禁用管理员模式
3. ✅ 或者添加IP白名单限制

---

## ✅ 安全良好的方面

### 1. SQL注入防护

**状态**: ✅ **良好**

- 所有数据库查询都使用参数化查询
- 没有发现SQL注入漏洞
- 使用`?`占位符，Go的database/sql会自动转义

**示例**:
```go
d.db.Exec(`UPDATE exchanges SET enabled = ?, api_key = ? WHERE id = ?`, enabled, apiKey, id)
```

---

### 2. 密码存储

**状态**: ✅ **良好**

- 使用bcrypt进行密码哈希
- 默认cost为10（足够安全）
- 密码不会以明文存储

**代码**:
```go
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}
```

---

### 3. JWT Token安全

**状态**: ✅ **良好**

- 使用HS256签名算法
- Token 24小时过期
- 包含用户ID和邮箱信息

**改进建议**:
- 考虑添加refresh token机制
- 考虑添加token撤销功能

---

### 4. 敏感信息不输出到日志

**状态**: ✅ **良好**

- 检查了关键文件，没有发现API密钥或密码被输出到日志
- 日志中只包含用户ID等非敏感信息

---

## 🟡 低风险问题

### 6. 决策日志可能包含敏感信息

**位置**: `decision_logs/` 目录

**风险**:
- 决策日志可能包含账户余额、持仓信息等
- 如果日志文件泄露，可能暴露交易策略

**影响**: 🟡 **低** - 取决于日志内容

**修复建议**:
1. ✅ 确保日志目录权限正确（已在.gitignore中）
2. ✅ 定期清理旧日志
3. ✅ 考虑加密敏感日志字段

---

### 7. 环境变量未使用

**当前状态**:
- 大部分配置从`config.json`读取
- 没有使用环境变量存储敏感信息

**修复建议**:
1. ✅ 优先使用环境变量
2. ✅ 配置文件作为fallback
3. ✅ 敏感信息必须从环境变量读取

---

## 📝 私有化部署安全检查清单

### 部署前必须完成

- [ ] **修改JWT密钥**: 使用环境变量或生成新的随机密钥
- [ ] **配置CORS**: 限制允许的源，不要使用`*`
- [ ] **设置文件权限**: `chmod 600 config.json config.db`
- [ ] **禁用管理员模式**: 生产环境必须禁用（`admin_mode: false`）
- [ ] **配置防火墙**: 限制API端口访问（只允许本地或特定IP）
- [ ] **使用HTTPS**: 如果通过公网访问，必须使用HTTPS
- [ ] **定期备份数据库**: 但确保备份文件加密存储

### 建议完成

- [ ] **数据库加密**: 对敏感字段进行加密
- [ ] **日志审计**: 记录所有敏感操作
- [ ] **IP白名单**: 限制Web界面访问来源
- [ ] **定期更新**: 保持依赖项和系统更新
- [ ] **监控告警**: 设置异常访问告警

### 可选增强

- [ ] **2FA强制启用**: 要求所有用户启用2FA
- [ ] **API限流**: 防止暴力破解
- [ ] **安全审计日志**: 记录所有API调用
- [ ] **密钥轮换**: 定期更换API密钥

---

## 🔧 快速修复指南

### 1. 修复JWT密钥问题

创建 `.env` 文件：
```bash
JWT_SECRET=your_random_secret_key_here_min_64_chars
```

修改 `main.go`:
```go
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    // 生成随机密钥
    secret := make([]byte, 64)
    rand.Read(secret)
    jwtSecret = base64.StdEncoding.EncodeToString(secret)
    log.Printf("⚠️  警告: JWT_SECRET未设置，已自动生成")
}
auth.SetJWTSecret(jwtSecret)
```

### 2. 修复CORS配置

修改 `api/server.go`:
```go
allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
if allowedOrigin == "" {
    allowedOrigin = "http://localhost:3000"
}
c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
```

### 3. 设置文件权限

```bash
chmod 600 config.json
chmod 600 config.db
chmod 700 decision_logs/
```

### 4. 配置防火墙（Linux）

```bash
# 只允许本地访问API
sudo ufw allow from 127.0.0.1 to any port 8080
sudo ufw deny 8080

# 只允许本地访问Web界面
sudo ufw allow from 127.0.0.1 to any port 3000
sudo ufw deny 3000
```

---

## 📊 风险评估总结

### 私有化部署可行性

✅ **可以私有化部署**，但需要：

1. **必须修复**:
   - JWT密钥使用环境变量
   - CORS配置限制来源
   - 文件权限设置

2. **强烈建议**:
   - 禁用管理员模式（生产环境）
   - 配置防火墙
   - 使用HTTPS（如果公网访问）

3. **可选增强**:
   - 数据库字段加密
   - API限流
   - 安全审计日志

### 部署场景评估

#### 场景1: 本地部署（localhost only）
- **风险等级**: 🟢 **低**
- **需要修复**: JWT密钥、文件权限
- **建议**: CORS可以保持宽松（仅本地访问）

#### 场景2: 内网部署（局域网）
- **风险等级**: 🟡 **中**
- **需要修复**: JWT密钥、CORS、文件权限、防火墙
- **建议**: 禁用管理员模式，使用IP白名单

#### 场景3: 公网部署（互联网）
- **风险等级**: 🔴 **高**
- **必须修复**: 所有高风险问题
- **必须完成**: HTTPS、强认证、IP白名单、API限流
- **强烈建议**: 数据库加密、安全审计

---

## 🛡️ 安全最佳实践建议

### 1. 密钥管理

```bash
# 使用环境变量
export JWT_SECRET="your_secret"
export BINANCE_API_KEY="your_key"
export BINANCE_SECRET_KEY="your_secret"

# 或使用密钥管理服务
# AWS Secrets Manager, HashiCorp Vault等
```

### 2. 数据库安全

```bash
# 设置数据库文件权限
chmod 600 config.db

# 定期备份（加密）
tar czf backup-$(date +%Y%m%d).tar.gz config.db
gpg --encrypt --recipient your@email.com backup-*.tar.gz
```

### 3. 网络安全

```bash
# 使用nginx反向代理 + HTTPS
# 配置SSL证书
# 限制访问IP
```

### 4. 监控和审计

```bash
# 监控异常访问
# 记录所有API调用
# 设置告警规则
```

---

## 📚 相关资源

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go安全最佳实践](https://go.dev/doc/security/best-practices)
- [SQLite安全指南](https://www.sqlite.org/security.html)
- [JWT安全最佳实践](https://datatracker.ietf.org/doc/html/rfc8725)

---

## ✅ 结论

**总体评估**: 系统**可以私有化部署**，但需要修复几个关键安全问题。

**优先级修复**:
1. 🔴 JWT密钥硬编码（必须修复）
2. 🟠 CORS配置过宽（必须修复）
3. 🟡 文件权限设置（建议修复）

**修复后风险等级**: 🟢 **低-中**（取决于部署场景）

修复这些问题后，系统可以安全地进行私有化部署。

---

**报告生成时间**: 2025-11-06  
**审计人员**: AI Security Auditor


