# 外部API安全分析报告

## 🎯 概述

本报告分析 NOFX 系统中所有外部API调用，评估是否存在信息泄露风险。

---

## 📡 外部API调用清单

### 1. AI模型API（DeepSeek / Qwen / Custom）

**调用位置**: `mcp/client.go`

**发送的数据**:
```json
{
  "model": "deepseek-chat",
  "messages": [
    {
      "role": "system",
      "content": "[系统提示词，包含交易策略]"
    },
    {
      "role": "user",
      "content": "[用户提示词，包含以下敏感信息]"
    }
  ]
}
```

**⚠️ 敏感信息泄露风险**:

#### 高风险数据（会发送给AI服务商）:

1. **账户信息**:
   - ✅ 账户净值（Total Equity）
   - ✅ 可用余额（Available Balance）
   - ✅ 总盈亏（Total PnL）
   - ✅ 保证金使用率（Margin Used %）

2. **持仓信息**:
   - ✅ 所有持仓的详细信息
   - ✅ 持仓价格（Entry Price）
   - ✅ 当前价格（Mark Price）
   - ✅ 未实现盈亏（Unrealized PnL）
   - ✅ 杠杆倍数（Leverage）
   - ✅ 清算价格（Liquidation Price）

3. **交易策略**:
   - ✅ 自定义交易策略（Custom Prompt）
   - ✅ 风险控制参数
   - ✅ 交易偏好设置

4. **市场数据**:
   - ✅ 关注的币种列表
   - ✅ 技术指标（MACD, RSI等）
   - ✅ 价格变化趋势

**示例数据**（实际发送给AI的内容）:
```
账户: 净值10000.00 | 余额8500.00 (85.0%) | 盈亏+5.2% | 保证金15.0% | 持仓2个

持仓:
1. BTCUSDT 多仓: 入场价95000 | 当前价98000 | 数量0.1 | 杠杆5x | 盈亏+300 | 清算价90000
2. ETHUSDT 空仓: 入场价3500 | 当前价3400 | 数量2.0 | 杠杆3x | 盈亏+200 | 清算价3800

候选币种: SOLUSDT, BNBUSDT, XRPUSDT...
```

**风险等级**: 🔴 **高风险**

**影响**:
- AI服务提供商（DeepSeek/Qwen/OpenAI等）可以看到您的：
  - 账户余额
  - 持仓情况
  - 交易策略
  - 风险偏好
- 这些数据可能被用于：
  - 数据分析
  - 模型训练（取决于服务商政策）
  - 商业分析

**缓解措施**:

1. **使用自托管AI模型**（推荐）:
   - 部署本地AI模型（如Ollama）
   - 使用私有化部署的AI服务
   - 确保数据不出本地网络

2. **数据脱敏**（部分缓解）:
   - 将余额转换为相对值（如"账户的80%"而非具体金额）
   - 隐藏具体持仓数量
   - 使用范围而非精确值

3. **审查AI服务商隐私政策**:
   - 确认数据是否用于训练
   - 确认数据保留期限
   - 确认数据共享政策

---

### 2. 交易所API（Binance / Hyperliquid / Aster）

**调用位置**: 
- `trader/binance_futures.go`
- `trader/hyperliquid_trader.go`
- `trader/aster_trader.go`

**发送的数据**:

#### Binance API:
- API密钥（在请求头中）
- 交易请求（开仓、平仓、设置止损等）
- 查询请求（余额、持仓等）

#### Hyperliquid API:
- 钱包地址
- 私钥签名
- 交易请求

#### Aster API:
- 用户地址
- 签名者地址
- 私钥签名
- 交易请求

**风险等级**: 🟡 **中等风险**

**影响**:
- API密钥在请求头中传输（这是正常的认证方式）
- 交易请求包含交易对、数量、价格等信息
- 这些信息是交易所API正常工作所必需的

**缓解措施**:
- ✅ 使用子账户（限制权限）
- ✅ 设置IP白名单
- ✅ 限制API权限（只允许交易，不允许提现）
- ✅ 定期轮换API密钥

---

### 3. 币种池API（Coin Pool API）

**调用位置**: `pool/coin_pool.go`

**发送的数据**:
- 仅GET请求
- 不发送任何敏感信息
- 只接收币种列表数据

**风险等级**: 🟢 **低风险**

**影响**: 无敏感信息泄露

---

### 4. OI Top API（持仓量Top数据）

**调用位置**: `pool/coin_pool.go`

**发送的数据**:
- 仅GET请求
- 不发送任何敏感信息

**风险等级**: 🟢 **低风险**

**影响**: 无敏感信息泄露

---

## 🔍 详细分析

### AI模型API数据泄露分析

#### 发送给AI的完整数据示例:

```json
{
  "system": "你是专业的加密货币交易AI...",
  "user": "时间: 2025-11-06 10:30:00 | 周期: #74 | 运行: 120分钟\n\nBTC: 98000.00 (1h: +2.5%, 4h: +5.2%) | MACD: 0.0123 | RSI: 65.5\n\n账户: 净值10000.00 | 余额8500.00 (85.0%) | 盈亏+5.2% | 保证金15.0% | 持仓2个\n\n持仓:\n1. BTCUSDT 多仓: 入场95000 | 当前98000 | 数量0.1 | 杠杆5x | 盈亏+300 | 清算90000\n2. ETHUSDT 空仓: 入场3500 | 当前3400 | 数量2.0 | 杠杆3x | 盈亏+200 | 清算3800\n\n候选币种: SOLUSDT, BNBUSDT...\n\n请做出交易决策..."
}
```

**泄露的信息包括**:
1. ✅ 账户总资产（10000.00）
2. ✅ 可用余额（8500.00）
3. ✅ 盈亏情况（+5.2%）
4. ✅ 持仓详情（币种、方向、价格、数量、杠杆）
5. ✅ 交易策略和偏好
6. ✅ 关注的币种

---

## 🛡️ 安全建议

### 立即行动（必须）

1. **审查AI服务商隐私政策**
   - 确认数据是否用于训练
   - 确认数据保留期限
   - 如果担心隐私，考虑自托管方案

2. **使用子账户和API限制**
   - 交易所API使用子账户
   - 限制API权限（禁止提现）
   - 设置IP白名单

### 强烈建议

1. **自托管AI模型**（最佳方案）
   ```bash
   # 使用Ollama部署本地模型
   docker run -d -v ollama:/root/.ollama -p 11434:11434 --name ollama ollama/ollama
   ollama pull qwen2.5:7b
   
   # 配置NOFX使用本地模型
   # custom_api_url: "http://localhost:11434/v1"
   # custom_model_name: "qwen2.5:7b"
   ```

2. **数据脱敏**（部分缓解）
   - 修改 `decision/engine.go` 中的 `buildUserPrompt` 函数
   - 将精确数值转换为相对值或范围

3. **使用代理/网关**
   - 通过私有代理转发AI API请求
   - 在代理层进行数据脱敏

### 可选增强

1. **加密敏感字段**
   - 对发送给AI的敏感数据进行加密
   - 使用AI服务商提供的加密传输选项

2. **数据最小化**
   - 只发送AI决策所需的最小数据
   - 移除不必要的详细信息

---

## 📊 风险总结

| API类型 | 风险等级 | 泄露数据 | 缓解难度 |
|---------|---------|---------|---------|
| **AI模型API** | 🔴 高 | 账户余额、持仓、策略 | 中等 |
| **交易所API** | 🟡 中 | API密钥、交易请求 | 低 |
| **币种池API** | 🟢 低 | 无 | 无 |
| **OI Top API** | 🟢 低 | 无 | 无 |

---

## 🔧 实施数据脱敏（示例代码）

如果需要减少发送给AI的敏感信息，可以修改 `decision/engine.go`:

```go
// 数据脱敏函数
func sanitizeAccountInfo(account AccountInfo) string {
    // 将精确余额转换为范围
    equityRange := getEquityRange(account.TotalEquity)
    return fmt.Sprintf("账户: 净值约%s | 余额约%.0f%% | 盈亏%+.1f%%", 
        equityRange, account.AvailableBalance/account.TotalEquity*100, account.TotalPnLPct)
}

func getEquityRange(equity float64) string {
    if equity < 1000 {
        return "<1K"
    } else if equity < 10000 {
        return "1K-10K"
    } else if equity < 100000 {
        return "10K-100K"
    } else {
        return ">100K"
    }
}
```

---

## ✅ 结论

**主要风险**: AI模型API会发送完整的账户和持仓信息给第三方服务商。

**建议**:
1. 🔴 **高风险用户**: 使用自托管AI模型（Ollama等）
2. 🟡 **中等风险用户**: 审查AI服务商隐私政策，考虑数据脱敏
3. 🟢 **低风险用户**: 如果隐私政策可接受，可以继续使用

**交易所API风险**: 正常，通过使用子账户和API限制可以缓解。

---

**最后更新**: 2025-11-06


