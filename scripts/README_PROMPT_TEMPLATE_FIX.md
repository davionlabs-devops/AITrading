# 提示词模板配置修复说明

## 问题描述

之前系统提示词模板（`system_prompt_template`）在更新交易员配置时无法正确保存和使用，导致即使选择了 `confidence_strategy` 模板，系统仍然使用默认模板。

## 已修复的问题

### 1. 后端修复 (`api/server.go`)

- ✅ **添加了 `SystemPromptTemplate` 字段到 `UpdateTraderRequest`**
  - 现在更新请求可以接收模板名称

- ✅ **修复了更新逻辑**
  - 之前强制使用原值，现在会使用请求中的新值
  - 如果请求中提供了新模板名称，会更新数据库

- ✅ **修复了 `handleGetTraderConfig` 返回结果**
  - 添加了 `system_prompt_template` 字段到返回结果
  - 前端现在可以正确获取当前配置的模板

### 2. 前端修复 (`web/src/components/AITradersPage.tsx`)

- ✅ **在更新请求中添加了 `system_prompt_template` 字段**
  - 现在更新交易员时会正确发送模板名称

## 使用方法

### 检查现有配置

运行诊断工具检查所有交易员的模板配置：

```bash
cd /Users/ez/Downloads/ez1/nofx-dev
go run scripts/check_prompt_templates.go
```

或者指定数据库路径：

```bash
go run scripts/check_prompt_templates.go /path/to/config.db
```

### 正确配置模板

1. **在交易员配置界面**：
   - 选择 "系统提示词模板" 为 "APEX 前端自研策略"（对应 `confidence_strategy`）
   - **不要勾选** "覆盖默认提示词"（除非要完全自定义）
   - "附加提示词" 可以留空，或填写补充内容（会追加到模板后面）

2. **保存配置**：
   - 点击 "保存修改"
   - 系统会更新数据库中的 `system_prompt_template` 字段

3. **验证配置**：
   - 查看日志，应该看到：`🤖 正在请求AI分析并决策... [模板: confidence_strategy]`
   - 检查决策日志文件（`decision_logs/` 目录），`system_prompt` 字段应该包含模板内容

## 重要提醒

### ⚠️ 覆盖默认提示词的影响

如果同时满足以下条件：
- ✅ 勾选了 "覆盖默认提示词"
- ✅ 填写了 "附加提示词"

**系统会完全忽略模板**，只使用自定义提示词！

### 推荐配置方式

#### 方式1：使用模板 + 补充提示词（推荐）
```
✅ 选择模板：confidence_strategy
❌ 不勾选 "覆盖默认提示词"
✅ 可以填写 "附加提示词"（会追加到模板后面）
```

#### 方式2：完全自定义
```
❌ 不选择模板（或选择任意模板，会被忽略）
✅ 勾选 "覆盖默认提示词"
✅ 填写完整的 "附加提示词"
```

#### 方式3：仅使用模板
```
✅ 选择模板：confidence_strategy
❌ 不勾选 "覆盖默认提示词"
❌ 不填写 "附加提示词"
```

## 模板文件位置

所有模板文件位于 `prompts/` 目录：

- `default.txt` - 默认稳健策略
- `adaptive.txt` - 自适应策略
- `adaptive_relaxed.txt` - 自适应宽松策略
- `confidence_strategy.txt` - APEX 前端自研策略（置信度算法）
- `Hansen.txt` - Hansen 策略
- `nof1.txt` - NOF1 策略
- `taro_long_prompts.txt` - Taro 长提示词策略

## 验证修复

### 方法1：查看日志

重启服务后，查看日志输出：

```bash
# 应该看到模板加载信息
✓ 已加载 X 个系统提示词模板
  📄 加载提示词模板: confidence_strategy (confidence_strategy.txt)

# 决策时应该看到
🤖 正在请求AI分析并决策... [模板: confidence_strategy]
```

### 方法2：检查决策日志

查看 `decision_logs/` 目录下的 JSON 文件：

```json
{
  "system_prompt": "你是 APEX 前端自研策略的专业加密货币交易AI...",
  ...
}
```

如果 `system_prompt` 开头是 "你是 APEX 前端自研策略..."，说明模板正确加载。

### 方法3：使用诊断工具

```bash
go run scripts/check_prompt_templates.go
```

会显示所有交易员的模板配置，并指出潜在问题。

## 常见问题

### Q: 为什么选择了模板但没有生效？

A: 检查以下几点：
1. 是否勾选了 "覆盖默认提示词" 并填写了 "附加提示词"？
2. 数据库中的 `system_prompt_template` 字段是否正确？
3. 模板文件是否存在（`prompts/confidence_strategy.txt`）？
4. 服务是否已重启？

### Q: 如何手动更新数据库？

A: 使用 SQLite 命令行工具：

```bash
sqlite3 config.db

# 查看所有交易员的模板配置
SELECT id, name, system_prompt_template, override_base_prompt 
FROM traders;

# 更新特定交易员的模板
UPDATE traders 
SET system_prompt_template = 'confidence_strategy' 
WHERE id = 'your_trader_id';
```

### Q: 模板文件修改后需要重启吗？

A: 是的，模板文件在服务启动时加载。修改后需要重启服务才能生效。

## 技术细节

### 模板加载流程

1. 服务启动时，`prompt_manager.go` 会扫描 `prompts/` 目录
2. 加载所有 `.txt` 文件，文件名（不含扩展名）作为模板名称
3. 存储在内存中的 `globalPromptManager` 中

### 模板使用流程

1. 交易员配置中的 `system_prompt_template` 字段指定模板名称
2. `buildSystemPrompt` 函数根据模板名称加载模板内容
3. 如果模板不存在，回退到 `default` 模板
4. 如果勾选了 "覆盖默认提示词" 且有自定义提示词，完全忽略模板

### 数据库字段

- `system_prompt_template`: 模板名称（如 `confidence_strategy`）
- `override_base_prompt`: 是否覆盖基础提示词（0/1）
- `custom_prompt`: 自定义提示词内容

