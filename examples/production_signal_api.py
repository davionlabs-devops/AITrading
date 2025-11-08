#!/usr/bin/env python3
"""
NOFX 生产级信号源API服务器
提供Coin Pool和OI Top数据源

特性:
- 数据缓存机制
- 错误处理和重试
- 性能优化
- 数据持久化
- 监控和日志
- 多数据源支持

使用方法:
    pip install flask requests redis sqlalchemy python-dotenv
    python production_signal_api.py
"""

from flask import Flask, jsonify, request
from flask_cors import CORS
import requests
import time
import json
import os
from datetime import datetime, timedelta
from typing import List, Dict, Optional
import logging
from functools import wraps
import threading
from collections import defaultdict

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)  # 允许跨域请求

# 配置
CONFIG = {
    'BINANCE_FUTURES_API': 'https://fapi.binance.com/fapi/v1',
    'CACHE_TTL': 300,  # 缓存5分钟
    'REQUEST_TIMEOUT': 10,
    'MAX_RETRIES': 3,
    'MIN_VOLUME': 1000000,  # 最小成交量（USDT）
    'MIN_OI': 1000000,  # 最小持仓量（USDT）
    'TOP_COINS': 30,  # 返回Top30币种
    'TOP_OI': 20,  # 返回Top20持仓量
}

# 内存缓存
cache = {
    'coin_pool': {
        'data': None,
        'timestamp': 0,
        'lock': threading.Lock()
    },
    'oi_top': {
        'data': None,
        'timestamp': 0,
        'lock': threading.Lock()
    }
}

# 历史数据存储（用于计算变化）
history_store = {
    'oi_history': defaultdict(dict),  # {symbol: {timestamp: oi_value}}
    'price_history': defaultdict(dict)  # {symbol: {timestamp: price}}
}


def retry_on_failure(max_retries=3, delay=1):
    """重试装饰器"""
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            last_exception = None
            for attempt in range(max_retries):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    last_exception = e
                    if attempt < max_retries - 1:
                        logger.warning(f"尝试 {attempt + 1}/{max_retries} 失败: {e}, {delay}秒后重试...")
                        time.sleep(delay * (attempt + 1))
                    else:
                        logger.error(f"所有重试失败: {e}")
            raise last_exception
        return wrapper
    return decorator


@retry_on_failure(max_retries=CONFIG['MAX_RETRIES'])
def fetch_binance_tickers() -> List[Dict]:
    """从Binance获取24小时行情数据"""
    url = f"{CONFIG['BINANCE_FUTURES_API']}/ticker/24hr"
    response = requests.get(url, timeout=CONFIG['REQUEST_TIMEOUT'])
    response.raise_for_status()
    return response.json()


@retry_on_failure(max_retries=CONFIG['MAX_RETRIES'])
def fetch_binance_open_interest(symbol: Optional[str] = None) -> List[Dict]:
    """从Binance获取持仓量数据"""
    if symbol:
        url = f"{CONFIG['BINANCE_FUTURES_API']}/openInterest"
        params = {'symbol': symbol}
    else:
        # 获取所有币种的持仓量（需要逐个请求）
        url = f"{CONFIG['BINANCE_FUTURES_API']}/openInterest"
        params = {}
    
    response = requests.get(url, params=params, timeout=CONFIG['REQUEST_TIMEOUT'])
    response.raise_for_status()
    return response.json()


@retry_on_failure(max_retries=CONFIG['MAX_RETRIES'])
def fetch_binance_top_long_short_account_ratio(symbol: str, period: str = '5m') -> Dict:
    """获取大户持仓量多空比"""
    url = f"{CONFIG['BINANCE_FUTURES_API']}/topLongShortAccountRatio"
    params = {'symbol': symbol, 'period': period}
    response = requests.get(url, params=params, timeout=CONFIG['REQUEST_TIMEOUT'])
    response.raise_for_status()
    data = response.json()
    return data[-1] if isinstance(data, list) else data


def calculate_coin_score(ticker: Dict) -> float:
    """
    计算币种评分（0-100）
    综合多个指标：涨幅、成交量、波动率、市场深度
    """
    score = 0.0
    
    # 1. 涨幅权重 35%
    price_change = float(ticker.get('priceChangePercent', 0))
    if price_change > 0:
        # 涨幅越大，分数越高，但设置上限
        score += min(price_change * 0.35, 35)
    elif price_change < -10:
        # 跌幅过大，扣分
        score -= min(abs(price_change) * 0.1, 10)
    
    # 2. 成交量权重 25%
    volume = float(ticker.get('quoteVolume', 0))
    if volume > 0:
        # 使用对数缩放，避免超大成交量占主导
        import math
        volume_score = min(math.log10(volume / 1000000 + 1) * 5, 25)
        score += volume_score
    
    # 3. 价格波动权重 20%
    high = float(ticker.get('highPrice', 0))
    low = float(ticker.get('lowPrice', 0))
    last = float(ticker.get('lastPrice', 0))
    if last > 0:
        volatility = abs(high - low) / last
        # 适度波动加分，过度波动扣分
        if 0.02 < volatility < 0.1:
            score += volatility * 200
        elif volatility > 0.15:
            score -= (volatility - 0.1) * 50
    
    # 4. 买卖盘深度权重 10%
    bid_qty = float(ticker.get('bidQty', 0))
    ask_qty = float(ticker.get('askQty', 0))
    if bid_qty > 0 and ask_qty > 0:
        depth_ratio = min(bid_qty, ask_qty) / max(bid_qty, ask_qty)
        score += depth_ratio * 10
    
    # 5. 交易次数权重 10%
    count = float(ticker.get('count', 0))
    if count > 0:
        count_score = min(math.log10(count / 1000 + 1) * 2, 10)
        score += count_score
    
    return max(0, min(score, 100))


def calculate_oi_change(symbol: str, current_oi: float) -> Dict:
    """计算持仓量变化"""
    now = int(time.time())
    one_hour_ago = now - 3600
    
    # 获取历史数据
    oi_history = history_store['oi_history'][symbol]
    price_history = history_store['price_history'][symbol]
    
    # 找到1小时前最近的数据
    historical_oi = None
    historical_price = None
    
    for timestamp, oi_value in sorted(oi_history.items(), reverse=True):
        if timestamp <= one_hour_ago:
            historical_oi = oi_value
            break
    
    for timestamp, price_value in sorted(price_history.items(), reverse=True):
        if timestamp <= one_hour_ago:
            historical_price = price_value
            break
    
    # 计算变化
    if historical_oi and historical_oi > 0:
        oi_delta = current_oi - historical_oi
        oi_delta_percent = (oi_delta / historical_oi) * 100
        oi_delta_value = oi_delta * historical_price if historical_price else 0
    else:
        oi_delta = 0
        oi_delta_percent = 0
        oi_delta_value = 0
    
    # 保存当前数据
    history_store['oi_history'][symbol][now] = current_oi
    if historical_price:
        history_store['price_history'][symbol][now] = historical_price
    
    return {
        'oi_delta': oi_delta,
        'oi_delta_percent': oi_delta_percent,
        'oi_delta_value': oi_delta_value
    }


def get_net_long_short(symbol: str) -> Dict:
    """获取净多空仓数据"""
    try:
        ratio_data = fetch_binance_top_long_short_account_ratio(symbol)
        long_short_ratio = float(ratio_data.get('longShortRatio', 1.0))
        
        # 获取当前持仓量
        oi_data = fetch_binance_open_interest(symbol)
        total_oi = float(oi_data.get('openInterest', 0))
        
        # 计算净多空仓（简化计算）
        if long_short_ratio > 1:
            # 多头占优
            net_long = total_oi * (long_short_ratio - 1) / (long_short_ratio + 1)
            net_short = total_oi - net_long
        else:
            # 空头占优
            net_short = total_oi * (1 - long_short_ratio) / (long_short_ratio + 1)
            net_long = total_oi - net_short
        
        return {
            'net_long': max(0, net_long),
            'net_short': max(0, net_short)
        }
    except Exception as e:
        logger.warning(f"获取 {symbol} 多空比失败: {e}")
        # 返回默认值
        return {
            'net_long': 0,
            'net_short': 0
        }


def build_coin_pool_data() -> Dict:
    """构建Coin Pool数据"""
    logger.info("开始构建Coin Pool数据...")
    
    tickers = fetch_binance_tickers()
    
    coins = []
    for ticker in tickers:
        symbol = ticker['symbol']
        
        # 只处理USDT交易对
        if not symbol.endswith('USDT'):
            continue
        
        # 过滤低成交量币种
        volume = float(ticker.get('quoteVolume', 0))
        if volume < CONFIG['MIN_VOLUME']:
            continue
        
        # 计算评分
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
    
    # 只返回Top N
    coins = coins[:CONFIG['TOP_COINS']]
    
    logger.info(f"Coin Pool数据构建完成，共 {len(coins)} 个币种")
    
    return {
        "success": True,
        "data": {
            "coins": coins,
            "count": len(coins)
        }
    }


def build_oi_top_data() -> Dict:
    """构建OI Top数据"""
    logger.info("开始构建OI Top数据...")
    
    # 获取所有活跃币种的持仓量
    tickers = fetch_binance_tickers()
    
    positions = []
    for ticker in tickers:
        symbol = ticker['symbol']
        
        if not symbol.endswith('USDT'):
            continue
        
        # 获取持仓量
        try:
            oi_data = fetch_binance_open_interest(symbol)
            current_oi = float(oi_data.get('openInterest', 0))
            
            if current_oi < CONFIG['MIN_OI']:
                continue
            
            # 计算持仓量变化
            oi_change = calculate_oi_change(symbol, current_oi)
            
            # 获取价格变化
            price_change = float(ticker.get('priceChangePercent', 0))
            
            # 获取净多空仓
            net_positions = get_net_long_short(symbol)
            
            position = {
                "symbol": symbol,
                "rank": 0,  # 稍后排序
                "oi_delta_percent": round(oi_change['oi_delta_percent'], 2),
                "oi_delta_value": round(oi_change['oi_delta_value'], 2),
                "price_delta_percent": round(price_change, 2),
                "net_long": round(net_positions['net_long'], 2),
                "net_short": round(net_positions['net_short'], 2)
            }
            positions.append(position)
            
            # 避免请求过快
            time.sleep(0.1)
            
        except Exception as e:
            logger.warning(f"处理 {symbol} 失败: {e}")
            continue
    
    # 按持仓量变化百分比排序
    positions.sort(key=lambda x: x['oi_delta_percent'], reverse=True)
    
    # 设置排名
    for i, pos in enumerate(positions[:CONFIG['TOP_OI']], 1):
        pos['rank'] = i
    
    logger.info(f"OI Top数据构建完成，共 {len(positions[:CONFIG['TOP_OI']])} 个币种")
    
    return {
        "success": True,
        "data": {
            "positions": positions[:CONFIG['TOP_OI']],
            "time_range": "1h"
        }
    }


def get_cached_data(cache_key: str, builder_func):
    """获取缓存数据或重新构建"""
    cache_item = cache[cache_key]
    
    with cache_item['lock']:
        now = time.time()
        
        # 检查缓存是否有效
        if cache_item['data'] and (now - cache_item['timestamp']) < CONFIG['CACHE_TTL']:
            logger.debug(f"使用缓存数据: {cache_key}")
            return cache_item['data']
        
        # 重新构建数据
        logger.info(f"缓存过期，重新构建数据: {cache_key}")
        try:
            data = builder_func()
            cache_item['data'] = data
            cache_item['timestamp'] = now
            return data
        except Exception as e:
            logger.error(f"构建数据失败: {e}")
            # 如果构建失败，返回旧缓存（如果有）
            if cache_item['data']:
                logger.warning("使用过期缓存数据")
                return cache_item['data']
            raise


@app.route('/coin-pool', methods=['GET'])
def coin_pool():
    """Coin Pool API"""
    try:
        data = get_cached_data('coin_pool', build_coin_pool_data)
        return jsonify(data)
    except Exception as e:
        logger.error(f"Coin Pool API错误: {e}")
        return jsonify({
            "success": False,
            "error": str(e)
        }), 500


@app.route('/oi-top', methods=['GET'])
def oi_top():
    """OI Top API"""
    try:
        data = get_cached_data('oi_top', build_oi_top_data)
        return jsonify(data)
    except Exception as e:
        logger.error(f"OI Top API错误: {e}")
        return jsonify({
            "success": False,
            "error": str(e)
        }), 500


@app.route('/health', methods=['GET'])
def health():
    """健康检查"""
    return jsonify({
        "status": "ok",
        "service": "NOFX Production Signal Source API",
        "timestamp": int(time.time()),
        "cache": {
            "coin_pool": {
                "cached": cache['coin_pool']['data'] is not None,
                "age": int(time.time() - cache['coin_pool']['timestamp']) if cache['coin_pool']['timestamp'] else None
            },
            "oi_top": {
                "cached": cache['oi_top']['data'] is not None,
                "age": int(time.time() - cache['oi_top']['timestamp']) if cache['oi_top']['timestamp'] else None
            }
        }
    })


@app.route('/stats', methods=['GET'])
def stats():
    """统计信息"""
    return jsonify({
        "config": CONFIG,
        "cache_status": {
            "coin_pool": {
                "has_data": cache['coin_pool']['data'] is not None,
                "last_update": cache['coin_pool']['timestamp'],
                "age_seconds": int(time.time() - cache['coin_pool']['timestamp']) if cache['coin_pool']['timestamp'] else None
            },
            "oi_top": {
                "has_data": cache['oi_top']['data'] is not None,
                "last_update": cache['oi_top']['timestamp'],
                "age_seconds": int(time.time() - cache['oi_top']['timestamp']) if cache['oi_top']['timestamp'] else None
            }
        }
    })


if __name__ == '__main__':
    print("=" * 60)
    print("NOFX 生产级信号源API服务器")
    print("=" * 60)
    port = int(os.getenv('PORT', 5001))
    print(f"Coin Pool API: http://localhost:{port}/coin-pool")
    print(f"OI Top API: http://localhost:{port}/oi-top")
    print(f"健康检查: http://localhost:{port}/health")
    print(f"统计信息: http://localhost:{port}/stats")
    print("=" * 60)
    print("特性:")
    print("  - 数据缓存机制（5分钟TTL）")
    print("  - 自动重试机制")
    print("  - 多线程安全")
    print("  - 历史数据追踪")
    print("  - 完整的错误处理")
    print("=" * 60)
    print("按 Ctrl+C 停止服务器")
    print()
    
    # 预热缓存
    logger.info("预热缓存...")
    try:
        get_cached_data('coin_pool', build_coin_pool_data)
        logger.info("Coin Pool缓存预热完成")
    except Exception as e:
        logger.warning(f"Coin Pool缓存预热失败: {e}")
    
    port = int(os.getenv('PORT', 5001))  # 默认使用5001端口，避免与macOS AirPlay冲突
    app.run(host='0.0.0.0', port=port, debug=False, threaded=True)

