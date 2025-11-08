# 生产环境部署指南

## 🎯 部署方案

### 方案1: Docker部署（推荐）

#### 1. 构建镜像

```bash
cd examples
docker build -t nofx-signal-api:latest .
```

#### 2. 启动服务

```bash
docker-compose up -d
```

#### 3. 查看日志

```bash
docker-compose logs -f signal-api
```

#### 4. 健康检查

```bash
curl http://localhost:5000/health
```

---

### 方案2: 直接运行

#### 1. 安装依赖

```bash
pip install -r requirements.txt
```

#### 2. 启动服务

```bash
python production_signal_api.py
```

#### 3. 使用systemd管理（Linux）

创建服务文件 `/etc/systemd/system/nofx-signal-api.service`:

```ini
[Unit]
Description=NOFX Signal Source API
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/nofx-dev/examples
ExecStart=/usr/bin/python3 production_signal_api.py
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务:

```bash
sudo systemctl enable nofx-signal-api
sudo systemctl start nofx-signal-api
sudo systemctl status nofx-signal-api
```

---

### 方案3: 使用Nginx反向代理

#### Nginx配置

```nginx
upstream signal_api {
    server 127.0.0.1:5000;
}

server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://signal_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

---

## 🔧 配置优化

### 环境变量配置

创建 `.env` 文件:

```bash
# API配置
BINANCE_API_TIMEOUT=10
CACHE_TTL=300
MIN_VOLUME=1000000
MIN_OI=1000000
TOP_COINS=30
TOP_OI=20

# 服务器配置
FLASK_ENV=production
FLASK_HOST=0.0.0.0
FLASK_PORT=5000

# 可选：Redis配置
# REDIS_HOST=localhost
# REDIS_PORT=6379
# REDIS_DB=0
```

修改代码加载环境变量:

```python
from dotenv import load_dotenv
load_dotenv()

CONFIG['CACHE_TTL'] = int(os.getenv('CACHE_TTL', 300))
# ...
```

---

## 📊 监控和日志

### 1. 日志配置

修改日志级别和输出:

```python
import logging
from logging.handlers import RotatingFileHandler

# 文件日志
file_handler = RotatingFileHandler(
    'signal_api.log',
    maxBytes=10*1024*1024,  # 10MB
    backupCount=5
)
file_handler.setLevel(logging.INFO)
file_handler.setFormatter(logging.Formatter(
    '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
))

logger.addHandler(file_handler)
```

### 2. 监控指标

添加Prometheus监控（可选）:

```python
from prometheus_client import Counter, Histogram, generate_latest

REQUEST_COUNT = Counter('api_requests_total', 'Total API requests', ['endpoint'])
REQUEST_DURATION = Histogram('api_request_duration_seconds', 'API request duration', ['endpoint'])

@app.route('/metrics')
def metrics():
    return generate_latest()
```

### 3. 健康检查增强

```python
@app.route('/health')
def health():
    health_status = {
        "status": "ok",
        "timestamp": int(time.time()),
        "checks": {
            "binance_api": check_binance_api(),
            "cache": check_cache(),
            "memory": check_memory()
        }
    }
    
    # 如果任何检查失败，返回503
    if not all(health_status["checks"].values()):
        return jsonify(health_status), 503
    
    return jsonify(health_status)
```

---

## 🚀 性能优化

### 1. 使用Redis缓存

```python
import redis

redis_client = redis.Redis(
    host=os.getenv('REDIS_HOST', 'localhost'),
    port=int(os.getenv('REDIS_PORT', 6379)),
    db=int(os.getenv('REDIS_DB', 0))
)

def get_cached_data_redis(key: str, builder_func, ttl: int = 300):
    """使用Redis缓存"""
    cached = redis_client.get(key)
    if cached:
        return json.loads(cached)
    
    data = builder_func()
    redis_client.setex(key, ttl, json.dumps(data))
    return data
```

### 2. 异步处理

使用Celery处理耗时任务:

```python
from celery import Celery

celery = Celery('signal_api')

@celery.task
def build_coin_pool_async():
    return build_coin_pool_data()
```

### 3. 数据库持久化

保存历史数据到数据库:

```python
from sqlalchemy import create_engine, Column, String, Float, Integer, DateTime
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker

Base = declarative_base()

class CoinPoolHistory(Base):
    __tablename__ = 'coin_pool_history'
    
    id = Column(Integer, primary_key=True)
    symbol = Column(String(20))
    score = Column(Float)
    timestamp = Column(DateTime)
    # ...

# 保存历史数据
def save_to_database(data):
    session = Session()
    # 保存逻辑
    session.commit()
```

---

## 🔒 安全加固

### 1. API认证

```python
from functools import wraps
from flask import request

API_KEYS = os.getenv('API_KEYS', '').split(',')

def require_api_key(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        api_key = request.headers.get('X-API-Key')
        if api_key not in API_KEYS:
            return jsonify({'error': 'Invalid API key'}), 401
        return f(*args, **kwargs)
    return decorated_function

@app.route('/coin-pool')
@require_api_key
def coin_pool():
    # ...
```

### 2. 速率限制

```python
from flask_limiter import Limiter
from flask_limiter.util import get_remote_address

limiter = Limiter(
    app=app,
    key_func=get_remote_address,
    default_limits=["100 per hour"]
)

@app.route('/coin-pool')
@limiter.limit("10 per minute")
def coin_pool():
    # ...
```

### 3. HTTPS配置

使用gunicorn + SSL:

```bash
pip install gunicorn

gunicorn --bind 0.0.0.0:5000 \
    --certfile cert.pem \
    --keyfile key.pem \
    production_signal_api:app
```

---

## 📈 扩展方案

### 1. 多数据源支持

```python
DATA_SOURCES = {
    'binance': BinanceDataSource(),
    'bybit': BybitDataSource(),
    'okx': OKXDataSource()
}

def build_coin_pool_multi_source():
    """从多个数据源聚合数据"""
    all_coins = []
    for source_name, source in DATA_SOURCES.items():
        coins = source.get_coins()
        all_coins.extend(coins)
    
    # 去重和合并
    # ...
```

### 2. 实时数据推送

使用WebSocket:

```python
from flask_socketio import SocketIO, emit

socketio = SocketIO(app, cors_allowed_origins="*")

@socketio.on('subscribe_coin_pool')
def handle_subscription():
    emit('coin_pool_update', get_cached_data('coin_pool', build_coin_pool_data))
```

### 3. 分布式部署

使用Kubernetes:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nofx-signal-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nofx-signal-api
  template:
    metadata:
      labels:
        app: nofx-signal-api
    spec:
      containers:
      - name: signal-api
        image: nofx-signal-api:latest
        ports:
        - containerPort: 5000
```

---

## ✅ 部署检查清单

- [ ] 代码已测试
- [ ] 依赖已安装
- [ ] 环境变量已配置
- [ ] 日志配置完成
- [ ] 监控已设置
- [ ] 健康检查正常
- [ ] 防火墙规则已配置
- [ ] SSL证书已配置（如需要）
- [ ] 备份策略已制定
- [ ] 文档已更新

---

## 📞 故障排查

### 常见问题

1. **API响应慢**
   - 检查网络连接
   - 增加缓存时间
   - 优化数据源请求

2. **内存占用高**
   - 清理历史数据
   - 使用Redis缓存
   - 限制并发请求

3. **数据不准确**
   - 检查数据源API
   - 验证计算逻辑
   - 查看日志错误

---

**最后更新**: 2025-11-06


