# 信号源API示例

## 🚀 快速开始

### 1. 安装依赖

```bash
pip install flask requests
```

### 2. 启动API服务器

```bash
python signal_source_api.py
```

服务器将在 `http://localhost:5000` 启动

### 3. 测试API

```bash
# 测试Coin Pool API
curl http://localhost:5000/coin-pool

# 测试OI Top API
curl http://localhost:5000/oi-top

# 健康检查
curl http://localhost:5000/health
```

### 4. 配置NOFX

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

### 5. 重启NOFX

```bash
./nofx
```

---

## 📝 说明

这个示例API服务器：
- ✅ 使用Binance公开API获取数据
- ✅ 无需API密钥
- ✅ 自动计算币种评分
- ✅ 自动计算持仓量变化
- ✅ 返回符合NOFX格式的数据

**注意**: 这是简化版本，生产环境建议：
- 添加数据缓存
- 添加错误处理
- 添加认证机制
- 优化算法

---

## 🔧 自定义

您可以修改 `signal_source_api.py` 来自定义：
- 评分算法
- 筛选条件
- 数据来源
- 返回格式

---

详细文档请参考: `../docs/HOW_TO_CONFIGURE_SIGNAL_SOURCES.md`


