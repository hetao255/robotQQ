package main

import (
    "context"
    "io/ioutil"
    "log"
    "os"
    "strings"
    "time"
    "net/http" 
    "encoding/json"

    "github.com/tencent-connect/botgo"
    "github.com/tencent-connect/botgo/dto"
    "github.com/tencent-connect/botgo/openapi"
    "github.com/tencent-connect/botgo/token"
    "github.com/tencent-connect/botgo/websocket"
    "github.com/tencent-connect/botgo/event"
    yaml "gopkg.in/yaml.v2"
)

//Config 定义了配置文件的结构
type Config struct {
    AppID uint64 `yaml:"appid"` //机器人的appid
    Token string `yaml:"token"` //机器人的token
}

//WeatherResp 定义了返回天气数据的结构
type WeatherResp struct {
    Success    string `json:"success"` //标识请求是否成功，0表示成功，1表示失败
    ResultData Result `json:"result"`  //请求成功时，获取的数据
    Msg        string `json:"msg"`     //请求失败时，失败的原因
}

//Result 定义了具体天气数据结构
type Result struct {
    Days            string `json:"days"`             //日期，例如2022-03-01
    Week            string `json:"week"`             //星期几
    CityNm          string `json:"citynm"`           //城市名
    Temperature     string `json:"temperature"`      //当日温度区间
    TemperatureCurr string `json:"temperature_curr"` //当前温度
    Humidity        string `json:"humidity"`         //湿度
    Weather         string `json:"weather"`          //天气情况
    Wind            string `json:"wind"`             //风向
    Winp            string `json:"winp"`             //风力
    TempHigh        string `json:"temp_high"`        //最高温度
    TempLow         string `json:"temp_low"`         //最低温度
    WeatherIcon     string `json:"weather_icon"`     //气象图标
}

var config Config
var api openapi.OpenAPI
var ctx context.Context

//第一步： 获取机器人的配置信息，即机器人的appid和token
func init() {
    content, err := ioutil.ReadFile("config.yaml")
    if err != nil {
        log.Println("读取配置文件出错， err = ", err)
        os.Exit(1)
    }

    err = yaml.Unmarshal(content, &config)
    if err != nil {
        log.Println("解析配置文件出错， err = ", err)
        os.Exit(1)
    }
    log.Println(config)
}

//atMessageEventHandler 处理 @机器人 的消息
func atMessageEventHandler(event *dto.WSPayload, data *dto.WSATMessageData) error {
    if strings.HasSuffix(data.Content, "> hello") {
        //回复：你好，请问想查询哪个城市的天气呢
		api.PostMessage(ctx, data.ChannelID, &dto.MessageToCreate{MsgID: data.ID, Content: "你好，请问想查询哪个城市的天气呢"})
    }else {
        //获取天气数据
        weatherData := getWeatherByCity(strings.Split(data.Content," ")[1])
        api.PostMessage(ctx, data.ChannelID, &dto.MessageToCreate{MsgID: data.ID,
            Content: weatherData.ResultData.CityNm + " " + weatherData.ResultData.Weather + " " + weatherData.ResultData.Days + " " + weatherData.ResultData.Week + "\n" + weatherData.ResultData.TempLow + "~" + weatherData.ResultData.TempHigh + " 当前温度：" + weatherData.ResultData.TemperatureCurr,
            Image: weatherData.ResultData.WeatherIcon,//天气图片
        })
    }
    return nil
}

func main() {
    //第二步：生成token，用于校验机器人的身份信息
    token := token.BotToken(config.AppID, config.Token) 
    //第三步：获取操作机器人的API对象
    api = botgo.NewOpenAPI(token).WithTimeout(3 * time.Second)
    //获取context
    ctx = context.Background()
    //第四步：获取websocket
    ws, err := api.WS(ctx, nil, "") 
    if err != nil {
        log.Fatalln("websocket错误， err = ", err)
        os.Exit(1)
    }

    var atMessage event.ATMessageEventHandler = atMessageEventHandler

    intent := websocket.RegisterHandlers(atMessage)     // 注册socket消息处理
    botgo.NewSessionManager().Start(ws, token, &intent) // 启动socket监听
}

//获取对应城市的天气数据
func getWeatherByCity(cityName string) *WeatherResp {
    url := "http://api.k780.com/?app=weather.today&cityNm=" + cityName + "&appkey=10003&sign=b59bc3ef6191eb9f747dd4e83c99f2a4&format=json"
    resp, err := http.Get(url)
    if err != nil {
        log.Fatalln("天气预报接口请求异常, err = ", err)
        return nil
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatalln("天气预报接口数据异常, err = ", err)
        return nil
    }
    var weatherData WeatherResp
    err = json.Unmarshal(body, &weatherData)
    if err != nil {
        log.Fatalln("解析数据异常 err = ", err, body)
        return nil
    }
    if weatherData.Success != "1" {
        log.Fatalln("返回数据问题 err = ", weatherData.Msg)
        return nil
    }
    return &weatherData
}