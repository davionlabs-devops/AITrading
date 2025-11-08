#!/usr/bin/env python3
"""
NOFX 信号源API示例服务器
提供Coin Pool和OI Top数据源

使用方法:
    pip install flask requests
    python signal_source_api.py
"""

from flask import Flask, jsonify
import requests
import time
from datetime import datetime, timedelta

app = Flask(__name__)

# Binance API基础URL
BINANCE_FUTURES_API = "https://fapi.binance.com/fapi/v1"


def get_binance_tickers():
    """从Binance获取24小时行情数据"""
    try:
        url = f"{BINANCE_FUTURES_API}/ticker/24hr"
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        return response.json()
    except Exception as e:
        print(f"获取Binance数据失败: {e}")
        return []


def calculate_coin_score(ticker):
    """计算币种评分（0-100）"""
    score = 0
    
    # 涨幅权重 40%
    price_change = float(ticker.get('priceChangePercent', 0))
    if price_change > 0:
        score += min(price_change * 0.4, 40)
    
    # 成交量权重 30%
    volume = float(ticker.get('quoteVolume', 0))
    # 假设100亿为满分
    volume_score = min(volume / 10000000000 * 30, 30)
    score += volume_score
    
    # 价格波动权重 30%
    high = float(ticker.get('highPrice', 0))
    low = float(ticker.get('lowPrice', 0))
    last = float(ticker.get('lastPrice', 0))
    if last > 0:
        volatility = abs(high - low) / last
        score += min(volatility * 100 * 0.3, 30)
    
    return min(score, 100)


@app.route('/coin-pool', methods=['GET'])
def coin_pool():
    """Coin Pool API - 返回评分最高的币种列表"""
    try:
        tickers = get_binance_tickers()
        
        if not tickers:
            return jsonify({
                "success": False,
                "error": "无法获取市场数据"
            }), 500
        
        coins = []
        for ticker in tickers:
            symbol = ticker['symbol']
            
            # 只处理USDT交易对
            if not symbol.endswith('USDT'):
                continue
            
            # 过滤掉一些不活跃的币种
            volume = float(ticker.get('quoteVolume', 0))
            if volume < 1000000:  # 成交量小于100万USDT的币种跳过
                continue
            
            score = calculate_coin_score(ticker)
            
            # 只返回评分>50的币种
            if score < 50:
                continue
            
            coin = {
                "pair": symbol,
                "score": round(score, 2),
                "start_time": int(time.time()) - 86400,  # 24小时前
                "start_price": float(ticker.get('openPrice', 0)),
                "last_score": round(score, 2),
                "max_score": round(score, 2),
                "max_price": float(ticker.get('highPrice', 0)),
                "increase_percent": float(ticker.get('priceChangePercent', 0))
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


def get_binance_open_interest():
    """从Binance获取持仓量数据"""
    try:
        url = f"{BINANCE_FUTURES_API}/openInterest"
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        return response.json()
    except Exception as e:
        print(f"获取持仓量数据失败: {e}")
        return []


@app.route('/oi-top', methods=['GET'])
def oi_top():
    """OI Top API - 返回持仓量增长最快的币种"""
    try:
        # 获取当前持仓量
        current_oi = {}
        oi_data = get_binance_open_interest()
        
        for item in oi_data:
            symbol = item.get('symbol', '')
            if symbol.endswith('USDT'):
                current_oi[symbol] = float(item.get('openInterest', 0))
        
        # 获取24小时价格变化
        tickers = get_binance_tickers()
        price_changes = {}
        for ticker in tickers:
            symbol = ticker['symbol']
            if symbol.endswith('USDT'):
                price_changes[symbol] = float(ticker.get('priceChangePercent', 0))
        
        # 计算持仓量变化（简化版：使用当前持仓量作为参考）
        # 注意：实际应该对比1小时前的持仓量，这里使用简化算法
        positions = []
        for symbol, oi in current_oi.items():
            if oi < 1000000:  # 持仓量小于100万的币种跳过
                continue
            
            # 简化算法：使用持仓量大小和价格变化计算排名
            price_change = price_changes.get(symbol, 0)
            
            # 计算持仓量变化百分比（简化：使用持仓量大小作为指标）
            oi_delta_percent = min(oi / 10000000 * 10, 20)  # 简化计算
            
            position = {
                "symbol": symbol,
                "rank": 0,  # 稍后排序
                "oi_delta_percent": round(oi_delta_percent, 2),
                "oi_delta_value": round(oi * 0.1, 2),  # 简化计算
                "price_delta_percent": round(price_change, 2),
                "net_long": round(oi * 0.6, 2),  # 简化：假设60%多仓
                "net_short": round(oi * 0.4, 2)   # 简化：假设40%空仓
            }
            positions.append(position)
        
        # 按持仓量变化百分比排序
        positions.sort(key=lambda x: x['oi_delta_percent'], reverse=True)
        
        # 设置排名
        for i, pos in enumerate(positions[:20], 1):  # 只返回Top20
            pos['rank'] = i
        
        return jsonify({
            "success": True,
            "data": {
                "positions": positions[:20],
                "time_range": "1h"
            }
        })
    except Exception as e:
        return jsonify({
            "success": False,
            "error": str(e)
        }), 500


@app.route('/health', methods=['GET'])
def health():
    """健康检查"""
    return jsonify({
        "status": "ok",
        "service": "NOFX Signal Source API",
        "timestamp": int(time.time())
    })


if __name__ == '__main__':
    print("=" * 50)
    print("NOFX 信号源API服务器")
    print("=" * 50)
    print("Coin Pool API: http://localhost:5000/coin-pool")
    print("OI Top API: http://localhost:5000/oi-top")
    print("健康检查: http://localhost:5000/health")
    print("=" * 50)
    print("按 Ctrl+C 停止服务器")
    print()
    
    app.run(host='0.0.0.0', port=5000, debug=False)


