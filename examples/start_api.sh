#!/bin/bash
# NOFX信号源API启动脚本

cd "$(dirname "$0")"

# 检查虚拟环境
if [ ! -d "venv" ]; then
    echo "创建虚拟环境..."
    python3 -m venv venv
fi

# 激活虚拟环境
source venv/bin/activate

# 安装依赖
echo "检查依赖..."
pip install -q flask flask-cors requests 2>/dev/null || {
    echo "安装依赖..."
    pip install flask flask-cors requests
}

# 检查端口是否被占用
if lsof -Pi :5000 -sTCP:LISTEN -t >/dev/null ; then
    echo "⚠️  端口5000已被占用"
    echo "正在停止旧进程..."
    pkill -f production_signal_api.py
    sleep 2
fi

# 启动服务
echo "启动API服务器..."
python3 production_signal_api.py > /tmp/signal_api.log 2>&1 &
API_PID=$!

echo "API服务器已启动，PID: $API_PID"
echo "日志文件: /tmp/signal_api.log"
echo ""
echo "等待服务就绪..."
sleep 5

# 健康检查
if curl -s http://localhost:5000/health > /dev/null; then
    echo "✓ API服务器运行正常"
    echo ""
    echo "API端点:"
    echo "  - Coin Pool: http://localhost:5000/coin-pool"
    echo "  - OI Top:    http://localhost:5000/oi-top"
    echo "  - Health:    http://localhost:5000/health"
    echo "  - Stats:     http://localhost:5000/stats"
    echo ""
    echo "停止服务: pkill -f production_signal_api.py"
else
    echo "✗ API服务器启动失败，查看日志: tail -f /tmp/signal_api.log"
    exit 1
fi


