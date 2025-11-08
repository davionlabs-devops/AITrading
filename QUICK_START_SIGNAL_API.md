# 🚀 快速开始 - 信号源API

## 当前状态
✅ API服务器已启动在端口 5001

## 立即操作

### 1. 配置NOFX（选择一种方式）

**方式A: Web界面（推荐）**
1. 访问 http://localhost:3000
2. 进入"信号源配置"
3. 填写:
   - Coin Pool URL: `http://localhost:5001/coin-pool`
   - OI Top URL: `http://localhost:5001/oi-top`
4. 保存

**方式B: 配置文件**
编辑 `config.json`:
```json
{
  "coin_pool_api_url": "http://localhost:5001/coin-pool",
  "oi_top_api_url": "http://localhost:5001/oi-top"
}
```
然后重启NOFX: `./nofx`

### 2. 启用交易员的信号源
1. 创建/编辑交易员
2. 勾选 "使用 Coin Pool 信号"
3. 勾选 "使用 OI Top 信号"
4. 保存并启动

### 3. 验证
```bash
# 测试API
curl http://localhost:5001/health

# 查看NOFX日志
tail -f /tmp/nofx.log | grep -E "币种池|信号源"
```

## 管理命令

```bash
# 启动API服务器
cd examples && ./start_api.sh

# 停止API服务器
pkill -f production_signal_api.py

# 查看日志
tail -f /tmp/signal_api.log
```

## 问题排查

**Coin Pool返回0个币种?**
- 这是正常的，可能是评分算法太严格
- 可以修改 `examples/production_signal_api.py` 降低评分阈值

**API无法访问?**
- 检查: `curl http://localhost:5001/health`
- 重启: `cd examples && ./start_api.sh`
