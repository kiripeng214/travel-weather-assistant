package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// ==================== 高德天气 API 响应结构 ====================

// AmapWeatherResponse 高德天气 API 实时天气响应结构
type AmapWeatherResponse struct {
	Status   string `json:"status"`
	Count    string `json:"count"`
	Info     string `json:"info"`
	Infocode string `json:"infocode"`
	Lives    []struct {
		Province      string `json:"province"`
		City          string `json:"city"`
		Adcode        string `json:"adcode"`
		Weather       string `json:"weather"`
		Temperature   string `json:"temperature"`
		WindDirection string `json:"winddirection"`
		WindPower     string `json:"windpower"`
		Humidity      string `json:"humidity"`
		ReportTime    string `json:"reporttime"`
	} `json:"lives"`
}

// ==================== 高德周边 POI 搜索 API 响应结构 ====================

// AmapPOIResponse 高德 V5 周边 POI 搜索响应结构
type AmapPOIResponse struct {
	Status   string `json:"status"`
	Info     string `json:"info"`
	Infocode string `json:"infocode"`
	Count    int    `json:"count"`
	POIs     struct {
		Poi []POIItem `json:"poi"`
	} `json:"pois"`
}

type POIItem struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Location string `json:"location"`
	Type     string `json:"type"`
	Typecode string `json:"typecode"`
	Pname    string `json:"pname"`
	Cityname string `json:"cityname"`
	Adname   string `json:"adname"`
	Address  string `json:"address"`
	Pcode    string `json:"pcode"`
	Adcode   string `json:"adcode"`
	Citycode string `json:"citycode"`
}

// ==================== 高德地理编码通用结构 ====================

// GeocodeResponse 高德地理编码 API 响应
type GeocodeResponse struct {
	Status   string        `json:"status"`
	Geocodes []GeocodeItem `json:"geocodes"`
}

type GeocodeItem struct {
	Location string `json:"location"` // "lng,lat"
	Adcode   string `json:"adcode"`
}

// amapApiKey 启动时统一校验后赋值
var amapApiKey string

// ==================== 城市编码映射 ====================

var cityCodeMap = map[string]string{
	"北京": "110000", "北京市": "110000", "beijing": "110000",
	"上海": "310000", "上海市": "310000", "shanghai": "310000",
	"天津": "120000", "天津市": "120000", "tianjin": "120000",
	"重庆": "500000", "重庆市": "500000", "chongqing": "500000",
	"广州": "440100", "广州市": "440100", "guangzhou": "440100",
	"深圳": "440300", "深圳市": "440300", "shenzhen": "440300",
	"杭州": "330100", "杭州市": "330100", "hangzhou": "330100",
	"成都": "510100", "成都市": "510100", "chengdu": "510100",
	"南京": "320100", "南京市": "320100", "nanjing": "320100",
	"武汉": "420100", "武汉市": "420100", "wuhan": "420100",
	"西安": "610100", "西安市": "610100", "xian": "610100",
	"长沙": "430100", "长沙市": "430100", "changsha": "430100",
	"厦门": "350200", "厦门市": "350200", "xiamen": "350200",
	"苏州": "320500", "苏州市": "320500", "suzhou": "320500",
	"青岛": "370200", "青岛市": "370200", "qingdao": "370200",
	"大连": "210200", "大连市": "210200", "dalian": "210200",
	"昆明": "530100", "昆明市": "530100", "kunming": "530100",
	"福州": "350100", "福州市": "350100", "fuzhou": "350100",
	"拉萨": "540100", "拉萨市": "540100", "lasa": "540100",
	"哈尔滨": "230100", "哈尔滨市": "230100", "haerbin": "230100",
	"乌鲁木齐": "650100", "乌鲁木齐市": "650100", "wulumuqi": "650100",
	"香港": "810000", "hongkong": "810000",
	"澳门": "820000", "macau": "820000",
	"台湾": "710000", "taiwan": "710000",
}

func resolveCityCode(input string) (string, error) {
	if matched, _ := regexp.MatchString(`^\d{6}$`, input); matched {
		return input, nil
	}
	key := strings.ToLower(strings.TrimSpace(input))
	if code, ok := cityCodeMap[key]; ok {
		return code, nil
	}
	return geocodeCity(input)
}

// ==================== 地理编码 ====================

func geocodeCity(city string) (string, error) {
	item, err := geocode(city)
	if err != nil {
		return "", err
	}
	return item.Adcode, nil
}

func geocodeLocation(city string) (string, error) {
	item, err := geocode(city)
	if err != nil {
		return "", err
	}
	return item.Location, nil
}

func geocode(city string) (*GeocodeItem, error) {
	apiURL := fmt.Sprintf(
		"https://restapi.amap.com/v3/geocode/geo?key=%s&address=%s&city=%s&output=JSON",
		amapApiKey, url.QueryEscape(city), url.QueryEscape(city),
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("地理编码请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取地理编码响应失败: %v", err)
	}

	var geoResp GeocodeResponse
	if err := json.Unmarshal(body, &geoResp); err != nil {
		return nil, fmt.Errorf("解析地理编码结果失败: %v", err)
	}

	if geoResp.Status != "1" || len(geoResp.Geocodes) == 0 {
		return nil, fmt.Errorf("未找到城市 '%s' 的编码信息", city)
	}

	item := &geoResp.Geocodes[0]
	log.Printf("地理编码查询: %s → adcode=%s, location=%s", city, item.Adcode, item.Location)
	return item, nil
}

// ==================== 天气查询 ====================

func getWeather(location, cityCode string) (string, error) {
	code := cityCode
	if code == "" {
		var err error
		code, err = resolveCityCode(location)
		if err != nil {
			return "", fmt.Errorf("无法解析城市编码: %v", err)
		}
	}

	apiURL := fmt.Sprintf(
		"https://restapi.amap.com/v3/weather/weatherInfo?key=%s&city=%s&extensions=base&output=JSON",
		amapApiKey, url.QueryEscape(code),
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("请求高德天气 API 失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var amapResp AmapWeatherResponse
	if err := json.Unmarshal(body, &amapResp); err != nil {
		return "", fmt.Errorf("解析高德天气数据失败: %v", err)
	}

	if amapResp.Status != "1" {
		return "", fmt.Errorf("高德 API 返回错误: %s (code: %s)", amapResp.Info, amapResp.Infocode)
	}
	if len(amapResp.Lives) == 0 {
		return "", fmt.Errorf("未找到城市 '%s' 的天气数据", location)
	}

	live := amapResp.Lives[0]
	result := fmt.Sprintf(
		`{"city":"%s","city_code":"%s","weather":"%s","temperature":"%s°C","humidity":"%s%%","wind_direction":"%s","wind_power":"%s级","report_time":"%s"}`,
		live.City, live.Adcode, live.Weather, live.Temperature, live.Humidity,
		live.WindDirection, live.WindPower, live.ReportTime,
	)
	return result, nil
}

// ==================== 周边 POI 搜索 ====================

func searchNearbyPOI(city, keywords string, radius int) (string, error) {
	location, err := geocodeLocation(city)
	if err != nil {
		return "", fmt.Errorf("获取城市坐标失败: %v", err)
	}

	if radius <= 0 {
		radius = 5000
	}

	apiURL := fmt.Sprintf(
		"https://restapi.amap.com/v5/place/around?key=%s&keywords=%s&location=%s&radius=%d&output=JSON",
		amapApiKey, url.QueryEscape(keywords), url.QueryEscape(location), radius,
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("请求周边搜索 API 失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取周边搜索响应失败: %v", err)
	}

	var poiResp AmapPOIResponse
	if err := json.Unmarshal(body, &poiResp); err != nil {
		return "", fmt.Errorf("解析周边搜索结果失败: %v", err)
	}

	if poiResp.Status != "1" {
		return "", fmt.Errorf("高德周边搜索 API 返回错误: %s (code: %s)", poiResp.Info, poiResp.Infocode)
	}

	type SimplePOI struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Type    string `json:"type"`
	}

	pois := make([]SimplePOI, 0, len(poiResp.POIs.Poi))
	for _, p := range poiResp.POIs.Poi {
		pois = append(pois, SimplePOI{Name: p.Name, Address: p.Address, Type: p.Type})
	}

	result := map[string]interface{}{
		"city":        city,
		"keywords":    keywords,
		"total_count": poiResp.Count,
		"pois":        pois,
	}

	b, _ := json.Marshal(result)
	return string(b), nil
}

// ==================== 工具调用注册表 ====================

type WeatherArgs struct {
	Location string `json:"location"`
	CityCode string `json:"city_code"`
}

type POIArgs struct {
	Location string `json:"location"`
	Keywords string `json:"keywords"`
	Radius   int    `json:"radius"`
}

type ToolRegistry struct {
	handlers map[string]func(args string) (string, error)
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{handlers: make(map[string]func(args string) (string, error))}
}

func (r *ToolRegistry) Register(name string, fn func(args string) (string, error)) {
	r.handlers[name] = fn
}

func (r *ToolRegistry) Execute(name, args string) (string, error) {
	fn, ok := r.handlers[name]
	if !ok {
		return "", fmt.Errorf("未知工具: %s", name)
	}
	return fn(args)
}

func weatherToolHandler(argsJSON string) (string, error) {
	var args WeatherArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %v", err)
	}
	if args.Location == "" && args.CityCode == "" {
		return "", fmt.Errorf("缺少 location 或 city_code 参数")
	}
	return getWeather(args.Location, args.CityCode)
}

func poiToolHandler(argsJSON string) (string, error) {
	var args POIArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %v", err)
	}
	if args.Location == "" {
		return "", fmt.Errorf("缺少 location（城市名称）参数")
	}
	if args.Keywords == "" {
		args.Keywords = "旅游景点"
	}
	if args.Radius <= 0 {
		args.Radius = 5000
	}
	return searchNearbyPOI(args.Location, args.Keywords, args.Radius)
}

func buildWeatherTool() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "get_current_weather",
			Description: "获取指定城市的当前天气信息",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "城市名称，例如：Beijing, Shanghai, 北京, 上海",
					},
					"city_code": map[string]interface{}{
						"type":        "string",
						"description": "高德城市编码（adcode），例如：110000（北京）、310000（上海）",
					},
				},
				"required": []string{"location", "city_code"},
			},
		},
	}
}

func buildPOITool() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "search_nearby_poi",
			Description: "搜索指定城市周边的景点/旅游点/餐馆/购物等兴趣点（POI）",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "城市名称，例如：北京, 上海, Guangzhou",
					},
					"keywords": map[string]interface{}{
						"type":        "string",
						"description": "搜索关键词，例如：旅游景点、博物馆、公园、餐馆（默认：旅游景点）",
					},
					"radius": map[string]interface{}{
						"type":        "integer",
						"description": "搜索半径（米），默认 5000（5公里）",
					},
				},
				"required": []string{"location"},
			},
		},
	}
}

// ==================== DeepSeek 客户端 ====================

func newDeepSeekClient() *openai.Client {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置环境变量 DEEPSEEK_API_KEY")
	}
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.deepseek.com/v1"
	return openai.NewClientWithConfig(config)
}

// ==================== 主程序 ====================

func main() {
	// 启动时统一校验环境变量
	if os.Getenv("AMAP_API_KEY") == "" {
		log.Fatal("请设置环境变量 AMAP_API_KEY（高德开放平台 Web 服务 API Key）")
	}
	amapApiKey = os.Getenv("AMAP_API_KEY")

	client := newDeepSeekClient()

	// 工具注册
	registry := NewToolRegistry()
	registry.Register("get_current_weather", weatherToolHandler)
	registry.Register("search_nearby_poi", poiToolHandler)

	tools := []openai.Tool{buildWeatherTool(), buildPOITool()}

	// 对话历史
	model := "deepseek-chat" // 使用稳定的非 reasoning 模型避免 reasoning_content 问题
	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: "你是一个旅游和天气助手。你可以回答天气和旅游信息。\n" +
				"- 当用户询问天气时，使用 get_current_weather 工具获取实时天气。\n" +
				"- 当用户询问景点、旅游推荐、周边好玩的地方时，使用 search_nearby_poi 工具搜索。\n" +
				"- 如果用户同时问天气和旅游，请同时调用两个工具。\n" +
				"两个工具都基于高德地图数据。",
		},
	}

	fmt.Println("=== 旅游 & 天气助手 ===")
	fmt.Println("输入城市名查询天气或旅游信息，输入 'exit' 退出")
	fmt.Println("例如：北京今天天气怎么样？ 北京有什么好玩的？ 上海的天气和景点")
	fmt.Println(strings.Repeat("-", 50))

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		userInput := scanner.Text()
		if userInput == "exit" {
			fmt.Println("再见！")
			break
		}
		if userInput == "" {
			continue
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userInput,
		})

		// 循环处理：模型返回 tool_calls 就执行并继续，直到纯文本回复为止
		// 这保证了 tool_calls 永远有对应的 tool 结果跟随，不会出现 orphaned tool_calls
		for {
			resp, err := client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    model,
					Messages: messages,
					Tools:    tools,
				},
			)
			if err != nil {
				log.Printf("请求 DeepSeek 失败: %v", err)
				// 出错时清理本轮添加的用户消息，避免消息序列被污染
				// 简单做法：移除最后一个 user 消息
				messages = messages[:len(messages)-1]
				break
			}

			assistantMsg := resp.Choices[0].Message

			// debug: 检查 reasoning_content
			if assistantMsg.ReasoningContent != "" {
				log.Printf("[debug] 助手回复包含 reasoning_content (%d 字符)", len(assistantMsg.ReasoningContent))
			}

			messages = append(messages, assistantMsg)

			if len(assistantMsg.ToolCalls) == 0 {
				// 模型不再调工具，输出回复并退出内层循环
				fmt.Printf("\n🤖 助手: %s\n", assistantMsg.Content)
				break
			}

			// 执行所有工具调用（registry 自动匹配，无需硬编码工具名）
			for _, toolCall := range assistantMsg.ToolCalls {
				result, err := registry.Execute(toolCall.Function.Name, toolCall.Function.Arguments)
				if err != nil {
					result = fmt.Sprintf("工具调用失败: %v", err)
				}
				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
				})
			}
			// 继续内层循环，让模型综合工具结果
		}
	}
}
