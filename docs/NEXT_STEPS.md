# 下一步操作指南

## ✅ 当前状态

### 1. 信号源API服务器
- **状态**: ✅ 已启动
- **端口**: 5001
- **健康检查**: http://localhost:5001/health

### 2. NOFX系统
- **后端**: 运行在 8080
- **前端**: 运行在 3000

---

## 🎯 下一步操作

### 步骤1: 配置NOFX使用信号源API

#### 方法A: 通过Web界面配置（推荐）

1. **打开浏览器**
   ```
   http://localhost:3000
   ```

2. **进入信号源配置页面**
   - 登录系统（如果未启用管理员模式）
   - 找到"信号源配置"菜单

3. **填写API地址**
   - **Coin Pool URL**: `http://localhost:5001/coin-pool`
   - **OI Top URL**: `http://localhost:5001/oi-top`

4. **保存配置**

#### 方法B: 通过config.json配置

编辑 `config.json`，添加：

```json
{
  "coin_pool_api_url": "http://localhost:5001/coin-pool",
  "oi_top_api_url": "http://localhost:5001/oi-top",
  "use_default_coins": false
}
```

**注意**: 如果设置了 `use_default_coins: false`，必须提供 `coin_pool_api_url`。

---

### 步骤2: 重启NOFX（如果修改了config.json）

```bash
# 停止NOFX
pkill -f nofx

# 重新启动
cd /Users/ez/Downloads/ez1/nofx-dev
./nofx
```

---

### 步骤3: 创建/编辑交易员并启用信号源

1. **进入交易员管理页面**
   - 创建新交易员或编辑现有交易员

2. **启用信号源**
   - ✅ 勾选 "使用 Coin Pool 信号"
   - ✅ 勾选 "使用 OI Top 信号"

3. **保存并启动交易员**

---

### 步骤4: 验证配置

#### 检查API是否正常工作

```bash
# 测试Coin Pool API
curl http://localhost:5001/coin-pool | python3 -m json.tool

# 测试OI Top API
curl http://localhost:5001/oi-top | python3 -m json.tool

# 健康检查
curl http://localhost:5001/health | python3 -m json.tool
```

#### 检查NOFX日志

```bash
# 查看NOFX日志，确认是否成功获取信号源数据
tail -f /tmp/nofx.log | grep -E "币种池|OI Top|信号源"
```

---

## 🔧 故障排查

### 问题1: Coin Pool返回0个币种

**原因**: 评分算法可能太严格，所有币种评分都低于50

**解决方案**:
1. 修改 `production_signal_api.py` 中的评分阈值
2. 或者降低 `MIN_VOLUME` 配置

编辑 `examples/production_signal_api.py`:
```python
# 修改第50行左右
if score < 50:  # 改为 30 或更低
    continue
```

### 问题2: OI Top API响应慢

**原因**: OI Top需要逐个请求每个币种的持仓量数据，比较慢

**解决方案**:
1. 这是正常的，首次请求需要时间
2. 后续请求会使用缓存（5分钟TTL）
3. 可以增加缓存时间

### 问题3: API服务器无法访问

**检查**:
```bash
# 检查进程是否运行
ps aux | grep production_signal_api

# 检查端口是否监听
lsof -i :5001

# 查看日志
tail -f /tmp/signal_api.log
```

**重启API服务器**:
```bash
cd /Users/ez/Downloads/ez1/nofx-dev/examples
./start_api.sh
```

---

## 📊 监控和测试

### 测试信号源API

```bash
# 1. 健康检查
curl http://localhost:5001/health

# 2. 统计信息
curl http://localhost:5001/stats

# 3. Coin Pool数据
curl http://localhost:5001/coin-pool | python3 -m json.tool | head -30

# 4. OI Top数据
curl http://localhost:5001/oi-top | python3 -m json.tool | head -30
```

### 查看NOFX日志

```bash
# 实时查看日志
tail -f /tmp/nofx.log

# 查看信号源相关日志
grep -E "币种池|OI Top|信号源|Coin Pool" /tmp/nofx.log
```

---

## 🚀 快速启动脚本

已创建启动脚本，方便管理：

```bash
# 启动API服务器
cd /Users/ez/Downloads/ez1/nofx-dev/examples
./start_api.sh

# 停止API服务器
pkill -f production_signal_api.py

# 查看日志
tail -f /tmp/signal_api.log
```

---

## 📝 配置检查清单

- [ ] API服务器已启动（端口5001）
- [ ] 健康检查通过
- [ ] Coin Pool API返回数据
- [ ] OI Top API返回数据
- [ ] NOFX已配置信号源URL
- [ ] 交易员已启用信号源
- [ ] NOFX日志显示成功获取数据

---

## 🎯 推荐配置

### 方案1: 使用默认币种（最简单）

```json
{
  "use_default_coins": true
}
```

**优点**: 无需外部API，稳定可靠

### 方案2: 使用信号源API（推荐）

```json
{
  "use_default_coins": false,
  "coin_pool_api_url": "http://localhost:5001/coin-pool",
  "oi_top_api_url": "http://localhost:5001/oi-top"
}
```

**优点**: 动态发现交易机会，更灵活

### 方案3: 混合模式

```json
{
  "use_default_coins": true,
  "coin_pool_api_url": "http://localhost:5001/coin-pool",
  "oi_top_api_url": "http://localhost:5001/oi-top"
}
```

**优点**: 如果API失败，自动回退到默认币种

---

## 📞 需要帮助？

如果遇到问题：

1. **查看日志**: `/tmp/signal_api.log` 和 `/tmp/nofx.log`
2. **检查API**: `curl http://localhost:5001/health`
3. **重启服务**: 使用 `start_api.sh` 脚本

---

**最后更新**: 2025-11-06


