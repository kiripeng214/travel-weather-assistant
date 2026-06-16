# 🧭 旅游 & 天气助手 (Travel & Weather Assistant)

一个基于 **DeepSeek API** + **高德地图 API** 的 Go 命令行智能助手，支持自然语言交互查询天气和周边旅游景点。

## ✨ 功能

- **🌤️ 实时天气查询** — 输入城市名，获取当前天气、温度、湿度、风力等信息
- **🗺️ 周边 POI 搜索** — 搜索指定城市的景点、餐馆、博物馆等兴趣点
- **🧠 自然语言交互** — 基于 DeepSeek Chat 模型 + Function Calling，支持连续对话
- **🔍 智能城市识别** — 支持中文/英文城市名、高德 adcode 编码，自动地理编码回退
- **⚙️ 多工具协同** — 一问同时查天气和景点，模型自动判断需要调用哪个工具

## 🚀 快速开始

### 前置条件

- Go 1.26+（或任意兼容版本）
- [高德开放平台](https://lbs.amap.com/) Web 服务 API Key
- [DeepSeek](https://platform.deepseek.com/) API Key

### 1. 克隆项目

```bash
git clone <your-repo-url>
cd travel-weather-assistant
```

### 2. 设置环境变量

```bash
export AMAP_API_KEY="你的高德Web服务APIKey"
export DEEPSEEK_API_KEY="你的DeepSeekAPIKey"
```

可将上述配置写入 `~/.bashrc` 或 `~/.zshrc` 避免重复输入。

### 3. 运行

```bash
go run main.go
```

### 4. 使用示例

```
=== 旅游 & 天气助手 ===
输入城市名查询天气或旅游信息，输入 'exit' 退出
例如：北京今天天气怎么样？ 北京有什么好玩的？ 上海的天气和景点
--------------------------------------------------

> 北京今天天气怎么样？

🤖 助手: 🌤️ 北京当前天气：晴，气温 22°C，湿度 45%，东北风 3级。

> 北京有什么好玩的景点？

🤖 助手: 🏛️ 北京热门景点推荐：故宫博物院、天安门广场、颐和园、八达岭长城、天坛公园等。

> 上海的天气和景点

🤖 助手: 🌤️ 上海当前天气：多云，气温 26°C...
🏛️ 上海周边景点推荐：外滩、东方明珠、豫园...
```

## 🧩 技术架构

```
用户输入
   │
   ▼
┌──────────────────────────────────────┐
│     DeepSeek Chat Model (LLM)        │
│  自然语言理解 + Function Calling     │
└──────┬─────────────────────┬─────────┘
       │                     │
       ▼                     ▼
┌──────────────┐   ┌──────────────────┐
│ 天气查询      │   │ 周边 POI 搜索    │
│ (高德天气API) │   │ (高德V5周边搜索) │
└──────────────┘   └──────────────────┘
       │                     │
       ▼                     ▼
┌──────────────────────────────────────┐
│   结构化 JSON 结果 → LLM 组织回复    │
└──────────────────────────────────────┘
```

### 核心依赖

| 依赖 | 用途 |
|------|------|
| `github.com/sashabaranov/go-openai` | OpenAI 兼容 API 客户端（连接 DeepSeek） |
| 高德 Web 服务 API v3/v5 | 天气查询 & 地理编码 & 周边 POI 搜索 |

## 🛠️ 工具说明

| 工具 | 描述 | 参数 |
|------|------|------|
| `get_current_weather` | 获取指定城市的实时天气 | `location` (城市名), `city_code` (adcode) |
| `search_nearby_poi` | 搜索周边兴趣点 | `location` (城市名), `keywords` (关键词), `radius` (半径米) |

系统内置 20+ 中国主要城市的 adcode 映射（北京、上海、广州、深圳等），未覆盖的城市会自动通过高德地理编码 API 实时查询。

## 🔧 环境变量

| 变量 | 必填 | 说明 |
|------|------|------|
| `AMAP_API_KEY` | ✅ | 高德开放平台 Web 服务 API Key |
| `DEEPSEEK_API_KEY` | ✅ | DeepSeek 平台 API Key |

## 📁 项目结构

```
travel-weather-assistant/
├── main.go          # 主程序：交互循环、工具注册、API 调用
├── go.mod           # Go 模块定义
├── go.sum           # 依赖版本锁
└── README.md        # 本文件
```

## 📝 注意事项

- 高德天气 API 免费版 QPS 有限制，大量请求时请注意频率
- 本工具默认使用 `deepseek-chat` 模型（非 reasoning 模型），避免 `reasoning_content` 兼容性问题
- 城市编码映射覆盖主要城市，其他城市会自动通过地理编码接口实时查询

## 📄 License

MIT
