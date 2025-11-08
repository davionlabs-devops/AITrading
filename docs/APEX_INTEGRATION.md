# APEX交易所集成指南

## ✅ 已完成的工作

### 1. 后端实现
- ✅ 创建了 `trader/apex_trader.go` - APEX交易员实现
- ✅ 实现了所有必需的接口方法
- ✅ 更新了数据库结构，添加APEX字段
- ✅ 更新了 `auto_trader.go` 支持APEX
- ✅ 更新了 `manager/trader_manager.go` 支持APEX配置
- ✅ 更新了 `api/server.go` 支持APEX API

### 2. 前端实现
- ✅ 更新了 `types.ts` 添加APEX类型定义
- ✅ 更新了 `AITradersPage.tsx` 添加APEX配置表单
- ✅ 更新了 `ExchangeIcons.tsx` 添加APEX图标

---

## 📋 如何配置APEX交易所

### 步骤1: 在Web界面添加APEX交易所

1. **打开NOFX Web界面**
   ```
   http://localhost:3000
   ```

2. **进入"交易所配置"页面**

3. **点击"添加交易所"**

4. **选择"APEX Pro (APEX)"**

5. **填写以下信息**:
   - **Account ID**: `642292042821533803`
   - **API Key**: `a5230315-7bef-deda-7e93-82e30befb31e`
   - **API Secret**: `daYFa-16FVy63x5l9WoTSQKOrgYUxdKwbA5ajX3d`
   - **Passphrase**: `10BDWO5QYkWbLKmzbv5y`
   - **Omni Key Seed** (可选): `0x819d170286e105d1ae3b7fc1753012ae8705347f36468fef6d1b1d65c2f2f1c115505c6b8f2b2c192f0f8f826a2060b9c94feac32cec249922c435b2bd0c58381b`

6. **保存配置**

---

## 🔧 API实现说明

### 认证方式

APEX使用HMAC-SHA256签名认证：

```go
// 签名生成
message = timestamp + method + path + body
signature = HMAC-SHA256(apiSecret, message)
```

### 请求头

```
APEX-API-KEY: {apiKey}
APEX-SIGNATURE: {signature}
APEX-TIMESTAMP: {timestamp}
APEX-PASSPHRASE: {passphrase}
```

### API端点

- **账户信息**: `GET /v1/account?accountId={accountId}`
- **持仓信息**: `GET /v1/positions?accountId={accountId}`
- **下单**: `POST /v1/order`
- **设置杠杆**: `POST /v1/leverage`
- **设置仓位模式**: `POST /v1/margin-mode`

---

## ⚠️ 注意事项

### 1. API签名格式

当前实现使用标准的HMAC-SHA256签名。如果APEX API文档要求不同的签名格式，可能需要调整 `generateSignature` 函数。

### 2. API端点URL

当前使用：
- **生产环境**: `https://api.pro.apex.exchange`
- **测试环境**: `https://api-testnet.pro.apex.exchange`

请根据APEX官方文档确认正确的API地址。

### 3. 响应格式

当前实现假设APEX API返回格式为：
```json
{
  "code": 0,
  "data": {...},
  "message": "..."
}
```

如果实际格式不同，需要调整解析逻辑。

### 4. 订单类型

当前实现使用 `MARKET` 订单类型。如果需要使用限价单，可以修改 `OpenLong` 和 `OpenShort` 方法。

---

## 🧪 测试步骤

### 1. 测试账户余额

```bash
# 启动NOFX
./nofx

# 查看日志，确认APEX连接成功
tail -f /tmp/nofx.log | grep -i apex
```

### 2. 创建交易员

1. 在Web界面创建新交易员
2. 选择APEX交易所
3. 选择AI模型
4. 设置初始余额
5. 启动交易员

### 3. 验证功能

- ✅ 获取账户余额
- ✅ 获取持仓信息
- ✅ 开仓/平仓
- ✅ 设置杠杆
- ✅ 设置止损/止盈

---

## 🔍 故障排查

### 问题1: "API返回错误"

**可能原因**:
- API密钥错误
- 签名算法不正确
- API端点URL错误

**解决方案**:
1. 检查API密钥是否正确
2. 查看APEX API文档，确认签名格式
3. 检查API端点URL

### 问题2: "获取账户信息失败"

**可能原因**:
- Account ID错误
- API权限不足

**解决方案**:
1. 确认Account ID正确
2. 检查API密钥权限（需要读取账户权限）

### 问题3: "下单失败"

**可能原因**:
- 余额不足
- 交易对不存在
- 杠杆设置错误

**解决方案**:
1. 检查账户余额
2. 确认交易对符号格式（如 `BTCUSDT`）
3. 检查杠杆设置

---

## 📝 API文档参考

请参考APEX官方API文档：
- **文档地址**: https://api-docs.omni.apex.exchange/
- **认证方式**: HMAC-SHA256
- **Base URL**: `https://api.pro.apex.exchange`

---

## 🔄 后续优化

如果遇到API格式不匹配，可能需要调整：

1. **签名算法**: 根据APEX文档调整 `generateSignature` 函数
2. **请求头格式**: 根据APEX文档调整请求头
3. **响应解析**: 根据实际响应格式调整解析逻辑
4. **错误处理**: 根据APEX错误码优化错误处理

---

## ✅ 配置检查清单

- [ ] APEX交易所已添加到数据库
- [ ] API密钥已正确配置
- [ ] Account ID已设置
- [ ] 交易员已创建并选择APEX
- [ ] 测试获取账户余额成功
- [ ] 测试获取持仓信息成功

---

**最后更新**: 2025-11-06


