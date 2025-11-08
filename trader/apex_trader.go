package trader

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// ApexTrader APEX交易平台实现
type ApexTrader struct {
	accountID    string // Account ID
	apiKey       string // API Key
	apiSecret    string // API Secret
	passphrase   string // API Key Passphrase
	omniKeySeed  string // Omni Key Seed (可选，用于高级功能)
	client       *http.Client
	baseURL      string
	testnet      bool

	// 余额缓存
	cachedBalance     map[string]interface{}
	balanceCacheTime  time.Time
	balanceCacheMutex sync.RWMutex

	// 持仓缓存
	cachedPositions     []map[string]interface{}
	positionsCacheTime  time.Time
	positionsCacheMutex sync.RWMutex

	// 缓存有效期（15秒）
	cacheDuration time.Duration

	// 交易对精度缓存
	symbolPrecision map[string]SymbolPrecision
	precisionMutex  sync.RWMutex
}

// NewApexTrader 创建APEX Omni交易器
func NewApexTrader(accountID, apiKey, apiSecret, passphrase, omniKeySeed string, testnet bool) (*ApexTrader, error) {
	// Apex Omni DEX API端点（参考项目使用 pro.apex.pro）
	baseURL := "https://pro.apex.pro"
	if testnet {
		baseURL = "https://hk.20240231.xyz" // 测试网地址
	}

	return &ApexTrader{
		accountID:      accountID,
		apiKey:         apiKey,
		apiSecret:      apiSecret,
		passphrase:     passphrase,
		omniKeySeed:    omniKeySeed,
		testnet:        testnet,
		baseURL:        baseURL,
		cacheDuration:  15 * time.Second,
		symbolPrecision: make(map[string]SymbolPrecision),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}, nil
}

// generateSignature 生成APEX API签名
func (t *ApexTrader) generateSignature(method, path, body, timestamp string) string {
	message := timestamp + method + path + body
	mac := hmac.New(sha256.New, []byte(t.apiSecret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// makeRequest 发送API请求
func (t *ApexTrader) makeRequest(method, endpoint string, params map[string]string, body interface{}) ([]byte, error) {
	path := endpoint
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		path += "?" + values.Encode()
	}

	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
	}

	// 使用毫秒时间戳（与参考项目一致）
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signature := t.generateSignature(method, path, string(bodyBytes), timestamp)

	req, err := http.NewRequest(method, t.baseURL+path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 使用正确的请求头格式（APEX-ACCESS-*）
	req.Header.Set("APEX-ACCESS-KEY", t.apiKey)
	req.Header.Set("APEX-ACCESS-SIGNATURE", signature)
	req.Header.Set("APEX-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("APEX-ACCESS-PASSPHRASE", t.passphrase)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API返回错误 (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetBalance 获取账户余额
func (t *ApexTrader) GetBalance() (map[string]interface{}, error) {
	// 检查缓存
	t.balanceCacheMutex.RLock()
	if t.cachedBalance != nil && time.Since(t.balanceCacheTime) < t.cacheDuration {
		cacheAge := time.Since(t.balanceCacheTime)
		t.balanceCacheMutex.RUnlock()
		log.Printf("✓ 使用缓存的账户余额（缓存时间: %.1f秒前）", cacheAge.Seconds())
		return t.cachedBalance, nil
	}
	t.balanceCacheMutex.RUnlock()

	// 调用API
	log.Printf("🔄 缓存过期，正在调用APEX Omni API获取账户余额...")
	
	// APEX Omni API: 尝试使用 v1 API (参考 jubi_apex 项目)
	// 先尝试 v1 API
	respBody, err := t.makeRequest("GET", "/api/v1/account/balance", nil, nil)
	if err != nil {
		// 如果 v1 失败，尝试 v3 API
		log.Printf("⚠️ v1 API 失败，尝试 v3 API: %v", err)
		respBody, err = t.makeRequest("GET", "/api/v3/account", nil, nil)
	}
	if err != nil {
		log.Printf("❌ APEX API调用失败: %v", err)
		return nil, fmt.Errorf("获取账户信息失败: %w", err)
	}

	// APEX Omni返回格式可能不同，需要适配
	// 支持 v1 和 v3 两种格式
	var accountResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			// v3 格式
			TotalEquity      string `json:"totalEquity"`
			AvailableBalance string `json:"availableBalance"`
			UnrealizedPnL    string `json:"unrealizedPnL"`
			// v1 格式
			TotalEquityValue string `json:"totalEquityValue"`
			AvailableBalanceValue string `json:"availableBalanceValue"`
			UnrealizedPnl    string `json:"unrealizedPnl"`
			// 或者可能是contractWallets格式
			ContractWallets []struct {
				Token   string `json:"token"`
				Balance string `json:"balance"`
			} `json:"contractWallets"`
		} `json:"data"`
		// 直接返回格式（v1可能直接返回数据）
		TotalEquity      string `json:"totalEquity"`
		AvailableBalance string `json:"availableBalance"`
		UnrealizedPnL    string `json:"unrealizedPnL"`
	}

	if err := json.Unmarshal(respBody, &accountResp); err != nil {
		return nil, fmt.Errorf("解析账户响应失败: %w", err)
	}

	// 检查是否有错误（v1和v3格式可能不同）
	if accountResp.Code != "" && accountResp.Code != "0" {
		return nil, fmt.Errorf("API返回错误: %s", accountResp.Message)
	}

	result := make(map[string]interface{})
	
	// 尝试解析不同格式的响应（优先使用Data字段，然后是直接字段）
	var totalEquity, availableBalance, unrealizedPnL string
	
	if accountResp.Data.TotalEquity != "" {
		totalEquity = accountResp.Data.TotalEquity
		availableBalance = accountResp.Data.AvailableBalance
		unrealizedPnL = accountResp.Data.UnrealizedPnL
	} else if accountResp.Data.TotalEquityValue != "" {
		// v1 格式
		totalEquity = accountResp.Data.TotalEquityValue
		availableBalance = accountResp.Data.AvailableBalanceValue
		unrealizedPnL = accountResp.Data.UnrealizedPnl
	} else if accountResp.TotalEquity != "" {
		// 直接返回格式
		totalEquity = accountResp.TotalEquity
		availableBalance = accountResp.AvailableBalance
		unrealizedPnL = accountResp.UnrealizedPnL
	}
	
	if totalEquity != "" {
		result["totalWalletBalance"], _ = strconv.ParseFloat(totalEquity, 64)
		result["availableBalance"], _ = strconv.ParseFloat(availableBalance, 64)
		result["totalUnrealizedProfit"], _ = strconv.ParseFloat(unrealizedPnL, 64)
	} else if len(accountResp.Data.ContractWallets) > 0 {
		// 从contractWallets中提取USDT余额
		var totalBalance float64
		for _, wallet := range accountResp.Data.ContractWallets {
			if wallet.Token == "USDT" {
				balance, _ := strconv.ParseFloat(wallet.Balance, 64)
				totalBalance += balance
			}
		}
		result["totalWalletBalance"] = totalBalance
		result["availableBalance"] = totalBalance
		result["totalUnrealizedProfit"] = 0.0
	}

	log.Printf("✓ APEX API返回: 总余额=%.2f, 可用=%.2f, 未实现盈亏=%.2f",
		result["totalWalletBalance"], result["availableBalance"], result["totalUnrealizedProfit"])

	// 更新缓存
	t.balanceCacheMutex.Lock()
	t.cachedBalance = result
	t.balanceCacheTime = time.Now()
	t.balanceCacheMutex.Unlock()

	return result, nil
}

// GetPositions 获取所有持仓
func (t *ApexTrader) GetPositions() ([]map[string]interface{}, error) {
	// 检查缓存
	t.positionsCacheMutex.RLock()
	if t.cachedPositions != nil && time.Since(t.positionsCacheTime) < t.cacheDuration {
		cacheAge := time.Since(t.positionsCacheTime)
		t.positionsCacheMutex.RUnlock()
		log.Printf("✓ 使用缓存的持仓信息（缓存时间: %.1f秒前）", cacheAge.Seconds())
		return t.cachedPositions, nil
	}
	t.positionsCacheMutex.RUnlock()

	// 调用API
	log.Printf("🔄 缓存过期，正在调用APEX Omni API获取持仓信息...")

	// APEX Omni API: 尝试使用 v1 API (参考 jubi_apex 项目)
	// 先尝试 v1 API
	respBody, err := t.makeRequest("GET", "/api/v1/account/positions", nil, nil)
	if err != nil {
		// 如果 v1 失败，尝试 v3 API
		log.Printf("⚠️ v1 API 失败，尝试 v3 API: %v", err)
		respBody, err = t.makeRequest("GET", "/api/v3/accounts/positions", nil, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	var positionsResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    []struct {
			Symbol          string `json:"symbol"`
			Side            string `json:"side"` // "LONG" or "SHORT"
			Size            string `json:"size"`
			EntryPrice      string `json:"entryPrice"`
			MarkPrice       string `json:"markPrice"`
			UnrealizedPnL   string `json:"unrealizedPnL"`
			Leverage        string `json:"leverage"`
			LiquidationPrice string `json:"liquidationPrice"`
			MarginUsed      string `json:"marginUsed"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &positionsResp); err != nil {
		return nil, fmt.Errorf("解析持仓响应失败: %w", err)
	}

	if positionsResp.Code != "0" && positionsResp.Code != "" {
		return nil, fmt.Errorf("API返回错误: %s", positionsResp.Message)
	}

	var result []map[string]interface{}
	for _, pos := range positionsResp.Data {
		size, _ := strconv.ParseFloat(pos.Size, 64)
		if math.Abs(size) < 0.0001 { // 忽略极小的持仓
			continue
		}

		entryPrice, _ := strconv.ParseFloat(pos.EntryPrice, 64)
		markPrice, _ := strconv.ParseFloat(pos.MarkPrice, 64)
		unrealizedPnL, _ := strconv.ParseFloat(pos.UnrealizedPnL, 64)
		leverage, _ := strconv.Atoi(pos.Leverage)
		liquidationPrice, _ := strconv.ParseFloat(pos.LiquidationPrice, 64)
		marginUsed, _ := strconv.ParseFloat(pos.MarginUsed, 64)

		side := "long"
		if pos.Side == "SHORT" || size < 0 {
			side = "short"
			size = math.Abs(size)
		}

		position := map[string]interface{}{
			"symbol":           pos.Symbol,
			"side":             side,
			"quantity":         size,
			"entry_price":      entryPrice,
			"mark_price":       markPrice,
			"unrealized_pnl":   unrealizedPnL,
			"leverage":         leverage,
			"liquidation_price": liquidationPrice,
			"margin_used":      marginUsed,
		}

		// 计算盈亏百分比
		if entryPrice > 0 {
			if side == "long" {
				position["unrealized_pnl_pct"] = ((markPrice - entryPrice) / entryPrice) * 100
			} else {
				position["unrealized_pnl_pct"] = ((entryPrice - markPrice) / entryPrice) * 100
			}
		}

		result = append(result, position)
	}

	// 更新缓存
	t.positionsCacheMutex.Lock()
	t.cachedPositions = result
	t.positionsCacheTime = time.Now()
	t.positionsCacheMutex.Unlock()

	return result, nil
}

// GetMarketPrice 获取市场价格
func (t *ApexTrader) GetMarketPrice(symbol string) (float64, error) {
	// APEX Omni API: GET /api/v1/ticker/{symbol}
	respBody, err := t.makeRequest("GET", "/api/v1/ticker/"+symbol, nil, nil)
	if err != nil {
		return 0, fmt.Errorf("获取市场价格失败: %w", err)
	}

	var tickerResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			LastPrice string `json:"lastPrice"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &tickerResp); err != nil {
		return 0, fmt.Errorf("解析价格响应失败: %w", err)
	}

	if tickerResp.Code != "0" && tickerResp.Code != "" {
		return 0, fmt.Errorf("API返回错误: %s", tickerResp.Message)
	}

	price, err := strconv.ParseFloat(tickerResp.Data.LastPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("解析价格失败: %w", err)
	}

	return price, nil
}

// getPrecision 获取交易对精度信息
func (t *ApexTrader) getPrecision(symbol string) (SymbolPrecision, error) {
	t.precisionMutex.RLock()
	if prec, ok := t.symbolPrecision[symbol]; ok {
		t.precisionMutex.RUnlock()
		return prec, nil
	}
	t.precisionMutex.RUnlock()

	// APEX Omni API: GET /api/v1/symbols
	respBody, err := t.makeRequest("GET", "/api/v1/symbols", nil, nil)
	if err != nil {
		return SymbolPrecision{}, err
	}

	var instrumentsResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    []struct {
			Symbol         string `json:"symbol"`
			TickSize       string `json:"tickSize"`
			StepSize       string `json:"stepSize"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &instrumentsResp); err != nil {
		return SymbolPrecision{}, err
	}

	for _, inst := range instrumentsResp.Data {
		if inst.Symbol == symbol {
			tickSize, _ := strconv.ParseFloat(inst.TickSize, 64)
			stepSize, _ := strconv.ParseFloat(inst.StepSize, 64)

			// 计算精度
			pricePrecision := 0
			if tickSize > 0 {
				pricePrecision = int(-math.Log10(tickSize))
			}

			quantityPrecision := 0
			if stepSize > 0 {
				quantityPrecision = int(-math.Log10(stepSize))
			}

			prec := SymbolPrecision{
				PricePrecision:    pricePrecision,
				QuantityPrecision: quantityPrecision,
				TickSize:          tickSize,
				StepSize:          stepSize,
			}

			t.precisionMutex.Lock()
			t.symbolPrecision[symbol] = prec
			t.precisionMutex.Unlock()

			return prec, nil
		}
	}

	return SymbolPrecision{}, fmt.Errorf("未找到交易对: %s", symbol)
}

// FormatQuantity 格式化数量到正确的精度
func (t *ApexTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	prec, err := t.getPrecision(symbol)
	if err != nil {
		// 如果获取精度失败，使用默认精度
		return fmt.Sprintf("%.8f", quantity), nil
	}

	if prec.StepSize > 0 {
		quantity = math.Floor(quantity/prec.StepSize) * prec.StepSize
	}

	return strconv.FormatFloat(quantity, 'f', prec.QuantityPrecision, 64), nil
}

// OpenLong 开多仓
func (t *ApexTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	// 先设置杠杆
	if err := t.SetLeverage(symbol, leverage); err != nil {
		log.Printf("⚠️ 设置杠杆失败，继续开仓: %v", err)
	}

	// 格式化数量
	qtyStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return nil, fmt.Errorf("格式化数量失败: %w", err)
	}

	// 获取当前价格（用于计算手续费）
	currentPrice, err := t.GetMarketPrice(symbol)
	if err != nil {
		log.Printf("⚠️ 获取市场价格失败，使用默认价格: %v", err)
		currentPrice = 0
	}
	priceStr := strconv.FormatFloat(currentPrice, 'f', 2, 64)

	// APEX Omni API: POST /api/v1/orders
	orderReq := map[string]interface{}{
		"symbol":     symbol,
		"side":       "BUY",
		"type":       "MARKET",
		"size":       qtyStr,
		"price":      priceStr, // 市价单也需要价格用于计算手续费
		"reduceOnly": false,
		"timeInForce": "GOOD_TIL_CANCEL",
	}

	respBody, err := t.makeRequest("POST", "/api/v1/orders", nil, orderReq)
	if err != nil {
		return nil, fmt.Errorf("开多仓失败: %w", err)
	}

	var orderResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("解析订单响应失败: %w", err)
	}

	if orderResp.Code != "0" && orderResp.Code != "" {
		return nil, fmt.Errorf("API返回错误: %s", orderResp.Message)
	}

	log.Printf("✓ 开多仓成功: %s 数量: %s", symbol, qtyStr)

	return map[string]interface{}{
		"order_id": orderResp.Data.OrderID,
		"symbol":   symbol,
		"side":     "long",
		"quantity": qtyStr,
	}, nil
}

// OpenShort 开空仓
func (t *ApexTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	// 先设置杠杆
	if err := t.SetLeverage(symbol, leverage); err != nil {
		log.Printf("⚠️ 设置杠杆失败，继续开仓: %v", err)
	}

	// 格式化数量
	qtyStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return nil, fmt.Errorf("格式化数量失败: %w", err)
	}

	// 获取当前价格（用于计算手续费）
	currentPrice, err := t.GetMarketPrice(symbol)
	if err != nil {
		log.Printf("⚠️ 获取市场价格失败，使用默认价格: %v", err)
		currentPrice = 0
	}
	priceStr := strconv.FormatFloat(currentPrice, 'f', 2, 64)

	// APEX Omni API: POST /api/v1/orders
	orderReq := map[string]interface{}{
		"symbol":     symbol,
		"side":       "SELL",
		"type":       "MARKET",
		"size":       qtyStr,
		"price":      priceStr, // 市价单也需要价格用于计算手续费
		"reduceOnly": false,
		"timeInForce": "GOOD_TIL_CANCEL",
	}

	respBody, err := t.makeRequest("POST", "/api/v1/orders", nil, orderReq)
	if err != nil {
		return nil, fmt.Errorf("开空仓失败: %w", err)
	}

	var orderResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("解析订单响应失败: %w", err)
	}

	if orderResp.Code != "0" && orderResp.Code != "" {
		return nil, fmt.Errorf("API返回错误: %s", orderResp.Message)
	}

	log.Printf("✓ 开空仓成功: %s 数量: %s", symbol, qtyStr)

	return map[string]interface{}{
		"order_id": orderResp.Data.OrderID,
		"symbol":   symbol,
		"side":     "short",
		"quantity": qtyStr,
	}, nil
}

// CloseLong 平多仓
func (t *ApexTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	// 获取当前持仓
	positions, err := t.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	var longPosition map[string]interface{}
	for _, pos := range positions {
		if pos["symbol"] == symbol && pos["side"] == "long" {
			longPosition = pos
			break
		}
	}

	if longPosition == nil {
		return nil, fmt.Errorf("未找到 %s 的多仓持仓", symbol)
	}

	// 如果quantity为0，平全部
	if quantity == 0 {
		quantity = longPosition["quantity"].(float64)
	}

	// 格式化数量
	qtyStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return nil, fmt.Errorf("格式化数量失败: %w", err)
	}

	// 获取当前价格（用于计算手续费）
	currentPrice, err := t.GetMarketPrice(symbol)
	if err != nil {
		log.Printf("⚠️ 获取市场价格失败，使用默认价格: %v", err)
		currentPrice = 0
	}
	priceStr := strconv.FormatFloat(currentPrice, 'f', 2, 64)

	// APEX Omni API: POST /api/v1/orders (平仓)
	orderReq := map[string]interface{}{
		"symbol":     symbol,
		"side":       "SELL", // 平多仓用SELL
		"type":       "MARKET",
		"size":       qtyStr,
		"price":      priceStr,
		"reduceOnly": true, // 平仓标志
		"timeInForce": "GOOD_TIL_CANCEL",
	}

	respBody, err := t.makeRequest("POST", "/api/v1/orders", nil, orderReq)
	if err != nil {
		return nil, fmt.Errorf("平多仓失败: %w", err)
	}

	var orderResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("解析订单响应失败: %w", err)
	}

	if orderResp.Code != "0" && orderResp.Code != "" {
		return nil, fmt.Errorf("API返回错误: %s", orderResp.Message)
	}

	log.Printf("✓ 平多仓成功: %s 数量: %s", symbol, qtyStr)

	return map[string]interface{}{
		"order_id": orderResp.Data.OrderID,
		"symbol":   symbol,
		"side":     "long",
		"quantity": qtyStr,
	}, nil
}

// CloseShort 平空仓
func (t *ApexTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	// 获取当前持仓
	positions, err := t.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	var shortPosition map[string]interface{}
	for _, pos := range positions {
		if pos["symbol"] == symbol && pos["side"] == "short" {
			shortPosition = pos
			break
		}
	}

	if shortPosition == nil {
		return nil, fmt.Errorf("未找到 %s 的空仓持仓", symbol)
	}

	// 如果quantity为0，平全部
	if quantity == 0 {
		quantity = shortPosition["quantity"].(float64)
	}

	// 格式化数量
	qtyStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return nil, fmt.Errorf("格式化数量失败: %w", err)
	}

	// 获取当前价格（用于计算手续费）
	currentPrice, err := t.GetMarketPrice(symbol)
	if err != nil {
		log.Printf("⚠️ 获取市场价格失败，使用默认价格: %v", err)
		currentPrice = 0
	}
	priceStr := strconv.FormatFloat(currentPrice, 'f', 2, 64)

	// APEX Omni API: POST /api/v1/orders (平仓)
	orderReq := map[string]interface{}{
		"symbol":     symbol,
		"side":       "BUY", // 平空仓用BUY
		"type":       "MARKET",
		"size":       qtyStr,
		"price":      priceStr,
		"reduceOnly": true, // 平仓标志
		"timeInForce": "GOOD_TIL_CANCEL",
	}

	respBody, err := t.makeRequest("POST", "/api/v1/orders", nil, orderReq)
	if err != nil {
		return nil, fmt.Errorf("平空仓失败: %w", err)
	}

	var orderResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &orderResp); err != nil {
		return nil, fmt.Errorf("解析订单响应失败: %w", err)
	}

	if orderResp.Code != "0" && orderResp.Code != "" {
		return nil, fmt.Errorf("API返回错误: %s", orderResp.Message)
	}

	log.Printf("✓ 平空仓成功: %s 数量: %s", symbol, qtyStr)

	return map[string]interface{}{
		"order_id": orderResp.Data.OrderID,
		"symbol":   symbol,
		"side":     "short",
		"quantity": qtyStr,
	}, nil
}

// SetLeverage 设置杠杆
func (t *ApexTrader) SetLeverage(symbol string, leverage int) error {
	// APEX Omni API: POST /api/v1/leverage
	leverageReq := map[string]interface{}{
		"symbol":   symbol,
		"leverage": leverage,
	}

	respBody, err := t.makeRequest("POST", "/api/v1/leverage", nil, leverageReq)
	if err != nil {
		return fmt.Errorf("设置杠杆失败: %w", err)
	}

	var resp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if resp.Code != "0" && resp.Code != "" {
		return fmt.Errorf("API返回错误: %s", resp.Message)
	}

	log.Printf("✓ %s 杠杆已设置为 %dx", symbol, leverage)
	return nil
}

// SetMarginMode 设置仓位模式
func (t *ApexTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	// APEX Omni API: POST /api/v1/margin-mode
	marginMode := "CROSSED"
	if !isCrossMargin {
		marginMode = "ISOLATED"
	}

	marginReq := map[string]interface{}{
		"symbol":     symbol,
		"marginMode": marginMode,
	}

	respBody, err := t.makeRequest("POST", "/api/v1/margin-mode", nil, marginReq)
	if err != nil {
		return fmt.Errorf("设置仓位模式失败: %w", err)
	}

	var resp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if resp.Code != "0" && resp.Code != "" {
		return fmt.Errorf("API返回错误: %s", resp.Message)
	}

	log.Printf("✓ %s 仓位模式已设置为 %s", symbol, marginMode)
	return nil
}

// SetStopLoss 设置止损单
func (t *ApexTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	// 格式化数量和价格
	qtyStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return fmt.Errorf("格式化数量失败: %w", err)
	}

	prec, err := t.getPrecision(symbol)
	if err != nil {
		return fmt.Errorf("获取精度失败: %w", err)
	}

	stopPriceStr := strconv.FormatFloat(stopPrice, 'f', prec.PricePrecision, 64)

	// APEX API: POST /v1/order (止损单)
	orderReq := map[string]interface{}{
		"accountId": t.accountID,
		"symbol":    symbol,
		"side":      "SELL", // 止损用SELL
		"type":      "STOP_MARKET",
		"size":      qtyStr,
		"stopPrice": stopPriceStr,
		"reduceOnly": true,
	}

	if positionSide == "short" {
		orderReq["side"] = "BUY" // 空仓止损用BUY
	}

	respBody, err := t.makeRequest("POST", "/v1/order", nil, orderReq)
	if err != nil {
		return fmt.Errorf("设置止损单失败: %w", err)
	}

	var resp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if resp.Code != "0" && resp.Code != "" {
		return fmt.Errorf("API返回错误: %s", resp.Message)
	}

	log.Printf("✓ %s 止损单已设置: 价格=%.4f, 数量=%s", symbol, stopPrice, qtyStr)
	return nil
}

// SetTakeProfit 设置止盈单
func (t *ApexTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	// 格式化数量和价格
	qtyStr, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return fmt.Errorf("格式化数量失败: %w", err)
	}

	prec, err := t.getPrecision(symbol)
	if err != nil {
		return fmt.Errorf("获取精度失败: %w", err)
	}

	takeProfitStr := strconv.FormatFloat(takeProfitPrice, 'f', prec.PricePrecision, 64)

	// APEX API: POST /v1/order (止盈单)
	orderReq := map[string]interface{}{
		"accountId":     t.accountID,
		"symbol":        symbol,
		"side":          "SELL", // 止盈用SELL
		"type":          "TAKE_PROFIT_MARKET",
		"size":          qtyStr,
		"takeProfitPrice": takeProfitStr,
		"reduceOnly":    true,
	}

	if positionSide == "short" {
		orderReq["side"] = "BUY" // 空仓止盈用BUY
	}

	respBody, err := t.makeRequest("POST", "/v1/order", nil, orderReq)
	if err != nil {
		return fmt.Errorf("设置止盈单失败: %w", err)
	}

	var resp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if resp.Code != "0" && resp.Code != "" {
		return fmt.Errorf("API返回错误: %s", resp.Message)
	}

	log.Printf("✓ %s 止盈单已设置: 价格=%.4f, 数量=%s", symbol, takeProfitPrice, qtyStr)
	return nil
}

// CancelAllOrders 取消该币种的所有挂单
func (t *ApexTrader) CancelAllOrders(symbol string) error {
	// APEX Omni API: DELETE /api/v1/orders?symbol={symbol}
	respBody, err := t.makeRequest("DELETE", "/api/v1/orders", map[string]string{
		"symbol": symbol,
	}, nil)
	if err != nil {
		return fmt.Errorf("取消所有订单失败: %w", err)
	}

	var resp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if resp.Code != "0" && resp.Code != "" {
		return fmt.Errorf("API返回错误: %s", resp.Message)
	}

	log.Printf("✓ 已取消 %s 的所有挂单", symbol)
	return nil
}

// CancelStopOrders 取消该币种的止盈/止损单
func (t *ApexTrader) CancelStopOrders(symbol string) error {
	// 先获取所有挂单
	// APEX Omni API: GET /api/v1/orders?symbol={symbol}&status=OPEN
	respBody, err := t.makeRequest("GET", "/api/v1/orders", map[string]string{
		"symbol": symbol,
		"status": "OPEN",
	}, nil)
	if err != nil {
		return fmt.Errorf("获取订单列表失败: %w", err)
	}

	var ordersResp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    []struct {
			OrderID string `json:"orderId"`
			Type    string `json:"type"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &ordersResp); err != nil {
		return fmt.Errorf("解析订单列表失败: %w", err)
	}

	// 取消所有止损/止盈单
	canceledCount := 0
	for _, order := range ordersResp.Data {
		if order.Type == "STOP_MARKET" || order.Type == "TAKE_PROFIT_MARKET" {
			// APEX Omni API: DELETE /api/v1/orders/{orderId}
			_, err := t.makeRequest("DELETE", "/api/v1/orders/"+order.OrderID, nil, nil)
			if err != nil {
				log.Printf("⚠️ 取消订单 %s 失败: %v", order.OrderID, err)
				continue
			}
			canceledCount++
		}
	}

	if canceledCount > 0 {
		log.Printf("✓ 已取消 %s 的 %d 个止盈/止损单", symbol, canceledCount)
	} else {
		log.Printf("ℹ %s 没有止盈/止损单需要取消", symbol)
	}

	return nil
}

