# 如何配置外部数据源

## 🎯 配置方法

系统支持**两种方式**配置外部数据源：

### 方法1: Web界面配置（推荐）

1. **登录系统**（如果未启用管理员模式）
2. **进入"信号源配置"页面**
3. **填写API URL**:
   - Coin Pool API URL: `https://your-api.com/coin-pool`
   - OI Top API URL: `https://your-api.com/oi-top`
4. **保存配置**

### 方法2: 配置文件（config.json）

编辑 `config.json`:
```json
{
  "coin_pool_api_url": "https://your-api.com/coin-pool",
  "oi_top_api_url": "https://your-api.com/oi-top"
}
```

---

## 📡 API数据格式要求

### Coin Pool API格式

**请求方式**: `GET`

**响应格式**:
```json
{
  "success": true,
  "data": {
    "coins": [
      {
        "pair": "BTCUSDT",
        "score": 85.5,
        "start_time": 1699123456,
        "start_price": 95000.0,
        "last_score": 85.5,
        "max_score": 90.0,
        "max_price": 98000.0,
        "increase_percent": 5.2
      },
      {
        "pair": "ETHUSDT",
        "score": 78.3,
        "start_time": 1699123456,
        "start_price": 3500.0,
        "last_score": 78.3,
        "max_score": 82.0,
        "max_price": 3600.0,
        "increase_percent": 3.5
      }
    ],
    "count": 2
  }
}
```

**字段说明**:
- `pair`: 交易对符号（如 "BTCUSDT"）
- `score`: 评分（0-100，越高越好）
- `start_time`: 开始时间（Unix时间戳）
- `start_price`: 开始价格
- `last_score`: 最新评分
- `max_score`: 最高评分
- `max_price`: 最高价格
- `increase_percent`: 涨幅百分比

---

### OI Top API格式

**请求方式**: `GET`

**响应格式**:
```json
{
  "success": true,
  "data": {
    "positions": [
      {
        "symbol": "BTCUSDT",
        "rank": 1,
        "oi_delta_percent": 15.5,
        "oi_delta_value": 1000000.0,
        "price_delta_percent": 3.2,
        "net_long": 500000.0,
        "net_short": 300000.0
      },
      {
        "symbol": "ETHUSDT",
        "rank": 2,
        "oi_delta_percent": 12.3,
        "oi_delta_value": 800000.0,
        "price_delta_percent": 2.5,
        "net_long": 400000.0,
        "net_short": 200000.0
      }
    ],
    "time_range": "1h"
  }
}
```

**字段说明**:
- `symbol`: 交易对符号（如 "BTCUSDT"）
- `rank`: OI Top排名（1-20）
- `oi_delta_percent`: 持仓量变化百分比（1小时）
- `oi_delta_value`: 持仓量变化价值（USD）
- `price_delta_percent`: 价格变化百分比
- `net_long`: 净多仓（USD）
- `net_short`: 净空仓（USD）

---

## 🔍 从哪里获取数据源？

### 选项1: 使用默认币种列表（最简单）

**无需配置任何API**，系统会使用默认主流币种：
```
BTCUSDT, ETHUSDT, SOLUSDT, BNBUSDT, 
XRPUSDT, DOGEUSDT, ADAUSDT, HYPEUSDT
```

**配置**:
```json
{
  "use_default_coins": true
}
```

**优点**: 
- ✅ 无需外部API
- ✅ 稳定可靠
- ✅ 零配置

**缺点**:
- ❌ 币种固定，无法动态发现新机会

---

### 选项2: 自建数据源API（推荐）

#### 方案A: 使用Python Flask/FastAPI

**Coin Pool API示例**:
```python
from flask import Flask, jsonify
import requests

app = Flask(__name__)

@app.route('/coin-pool', methods=['GET'])
def get_coin_pool():
    # 从交易所API获取数据
    # 计算评分
    # 返回格式化的数据
    
    coins = [
        {
            "pair": "BTCUSDT",
            "score": 85.5,
            "start_time": 1699123456,
            "start_price": 95000.0,
            "last_score": 85.5,
            "max_score": 90.0,
            "max_price": 98000.0,
            "increase_percent": 5.2
        }
    ]
    
    return jsonify({
        "success": True,
        "data": {
            "coins": coins,
            "count": len(coins)
        }
    })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
```

**OI Top API示例**:
```python
@app.route('/oi-top', methods=['GET'])
def get_oi_top():
    # 从交易所API获取持仓量数据
    # 计算持仓量变化
    # 返回Top20
    
    positions = [
        {
            "symbol": "BTCUSDT",
            "rank": 1,
            "oi_delta_percent": 15.5,
            "oi_delta_value": 1000000.0,
            "price_delta_percent": 3.2,
            "net_long": 500000.0,
            "net_short": 300000.0
        }
    ]
    
    return jsonify({
        "success": True,
        "data": {
            "positions": positions,
            "time_range": "1h"
        }
    })
```

#### 方案B: 使用Node.js Express

```javascript
const express = require('express');
const app = express();

app.get('/coin-pool', (req, res) => {
  // 获取币种池数据
  res.json({
    success: true,
    data: {
      coins: [...],
      count: 10
    }
  });
});

app.get('/oi-top', (req, res) => {
  // 获取OI Top数据
  res.json({
    success: true,
    data: {
      positions: [...],
      time_range: "1h"
    }
  });
});

app.listen(5000, () => {
  console.log('API server running on port 5000');
});
```

---

### 选项3: 使用第三方数据服务

#### Coin Pool数据源

可以从以下来源获取币种评分数据：

1. **交易所公开API**:
   - Binance API: `https://api.binance.com/api/v3/ticker/24hr`
   - 计算24小时涨幅、成交量等指标
   - 根据涨幅、成交量计算评分

2. **CoinGecko API**:
   - `https://api.coingecko.com/api/v3/coins/markets`
   - 提供市值、价格变化等数据

3. **CoinMarketCap API**:
   - 需要API密钥
   - 提供市场数据

#### OI Top数据源

可以从以下来源获取持仓量数据：

1. **Binance Futures API**:
   - `https://fapi.binance.com/fapi/v1/openInterest`
   - 获取各币种的持仓量数据
   - 计算1小时持仓量变化

2. **Bybit API**:
   - `https://api.bybit.com/v2/public/open-interest`
   - 获取持仓量数据

3. **自建数据采集**:
   - 定期从多个交易所采集持仓量数据
   - 计算持仓量变化百分比
   - 返回Top20

---

## 🛠️ 完整示例：自建Coin Pool API

### 使用Binance API构建Coin Pool

```python
from flask import Flask, jsonify
import requests
import time

app = Flask(__name__)

def get_binance_tickers():
    """从Binance获取24小时行情数据"""
    url = "https://fapi.binance.com/fapi/v1/ticker/24hr"
    response = requests.get(url)
    return response.json()

def calculate_score(ticker):
    """计算币种评分"""
    price_change = float(ticker['priceChangePercent'])
    volume = float(ticker['quoteVolume'])
    
    # 评分算法（示例）
    score = 0
    
    # 涨幅权重 40%
    if price_change > 0:
        score += min(price_change * 0.4, 40)
    
    # 成交量权重 30%
    volume_score = min(volume / 1000000000 * 30, 30)  # 假设10亿为满分
    score += volume_score
    
    # 价格变化权重 30%
    price_volatility = abs(float(ticker['highPrice']) - float(ticker['lowPrice'])) / float(ticker['lastPrice'])
    score += min(price_volatility * 100 * 0.3, 30)
    
    return min(score, 100)

@app.route('/coin-pool', methods=['GET'])
def coin_pool():
    try:
        tickers = get_binance_tickers()
        
        coins = []
        for ticker in tickers:
            symbol = ticker['symbol']
            
            # 只处理USDT交易对
            if not symbol.endswith('USDT'):
                continue
            
            score = calculate_score(ticker)
            
            # 只返回评分>50的币种
            if score < 50:
                continue
            
            coin = {
                "pair": symbol,
                "score": round(score, 2),
                "start_time": int(time.time()) - 86400,  # 24小时前
                "start_price": float(ticker['openPrice']),
                "last_score": round(score, 2),
                "max_score": round(score, 2),
                "max_price": float(ticker['highPrice']),
                "increase_percent": float(ticker['priceChangePercent'])
            }
            coins.append(coin)
        
        # 按评分排序
        coins.sort(key=lambda x: x['score'], reverse=True)
        
        # 只返回Top30
        coins = coins[:30]
        
        return jsonify({
            "success": True,
            "data": {
                "coins": coins,
                "count": len(coins)
            }
        })
    except Exception as e:
        return jsonify({
            "success": False,
            "error": str(e)
        }), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
```

---

## 🚀 快速开始

### 步骤1: 部署API服务

```bash
# 使用上面的Python示例
pip install flask requests
python api_server.py
```

### 步骤2: 配置NOFX

**方法A: Web界面**
- 进入"信号源配置"
- Coin Pool URL: `http://localhost:5000/coin-pool`
- OI Top URL: `http://localhost:5000/oi-top`

**方法B: config.json**
```json
{
  "coin_pool_api_url": "http://localhost:5000/coin-pool",
  "oi_top_api_url": "http://localhost:5000/oi-top"
}
```

### 步骤3: 重启NOFX

```bash
./nofx
```

---

## ⚠️ 注意事项

### 1. API响应时间

- **建议**: API响应时间 < 5秒
- **超时设置**: 系统默认30秒超时
- **重试机制**: 系统会自动重试3次

### 2. API可用性

- 如果API不可用，系统会使用缓存数据
- 如果缓存也不可用，会使用默认币种列表
- 建议API服务保持高可用

### 3. 数据更新频率

- **Coin Pool**: 建议每5-10分钟更新一次
- **OI Top**: 建议每1小时更新一次
- 系统会缓存数据，避免频繁请求

### 4. 数据格式验证

- 确保返回的JSON格式正确
- 确保`success: true`
- 确保必需字段都存在

---

## 📝 测试API

### 测试Coin Pool API

```bash
curl http://localhost:5000/coin-pool
```

**期望响应**:
```json
{
  "success": true,
  "data": {
    "coins": [...],
    "count": 10
  }
}
```

### 测试OI Top API

```bash
curl http://localhost:5000/oi-top
```

**期望响应**:
```json
{
  "success": true,
  "data": {
    "positions": [...],
    "time_range": "1h"
  }
}
```

---

## 🎯 推荐方案

### 方案1: 最简单（推荐新手）

**使用默认币种列表**:
```json
{
  "use_default_coins": true
}
```

### 方案2: 自建简单API（推荐进阶）

**使用Binance公开API构建**:
- 无需API密钥
- 数据实时更新
- 完全可控

### 方案3: 专业数据服务（推荐生产环境）

**使用专业数据提供商**:
- 数据质量高
- 稳定性好
- 需要付费

---

## 📚 相关文档

- [信号源配置说明](./SIGNAL_SOURCE_GUIDE.md)
- [API接口文档](../README.md)
- [部署指南](./DEPLOYMENT_SECURITY_GUIDE.md)

---

**最后更新**: 2025-11-06


