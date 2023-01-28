package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type MeteoAPI interface {
	GetMeteoData(start, stop string, client *resty.Client) ([]byte, error)
	FormatReport(weather []byte) (formattedInfo string, err error)
	ReportCount(weather []byte) (count int, err error)
	SaveToFile(data []byte) error
	LoadFromFile() (data []byte, err error)
}

type MeteoAPIData struct {
	cfg Config
}

type MeteoController struct {
	api    MeteoAPI
	client resty.Client
	cfg    Config
}

type Config struct {
	Address   string
	StoreFile string
	Restore   bool
	Key       string
}

type SurfInfo struct {
	Hours []Hours `json:"hours"`
	Meta  Meta    `json:"meta"`
}
type SwellDirection struct {
	Sg float64 `json:"sg"`
}
type SwellHeight struct {
	Sg float64 `json:"sg"`
}
type SwellPeriod struct {
	Sg float64 `json:"sg"`
}
type WaterTemperature struct {
	Sg float64 `json:"sg"`
}
type WindSpeed struct {
	Sg float64 `json:"sg"`
}
type Hours struct {
	SwellDirection   SwellDirection   `json:"swellDirection"`
	SwellHeight      SwellHeight      `json:"swellHeight"`
	SwellPeriod      SwellPeriod      `json:"swellPeriod"`
	Time             time.Time        `json:"time"`
	WaterTemperature WaterTemperature `json:"waterTemperature"`
	WindSpeed        WindSpeed        `json:"windSpeed"`
}
type Meta struct {
	Cost         int      `json:"cost"`
	DailyQuota   int      `json:"dailyQuota"`
	End          string   `json:"end"`
	Lat          float64  `json:"lat"`
	Lng          float64  `json:"lng"`
	Params       []string `json:"params"`
	RequestCount int      `json:"requestCount"`
	Source       []string `json:"source"`
	Start        string   `json:"start"`
}

func main() {
	cfg := NewServerConfig()
	client := resty.New()
	api := &MeteoAPIData{
		cfg: *cfg,
	}
	controller := &MeteoController{
		api:    api,
		client: *client,
		cfg:    *cfg,
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	router := r.Group("/")
	{
		router.GET("/", controller.WeatherReport)
		router.GET("/update", controller.UpdateWeatherReport)
	}
	r.Run(cfg.Address)

}

func NewServerConfig() *Config {
	var cfg Config
	flag.StringVar(&cfg.Key, "k", "8693c2da-9ca8-11ed-bce5-0242ac130002-8693c3c0-9ca8-11ed-bce5-0242ac130002", "set API key")
	flag.StringVar(&cfg.Address, "a", "10.0.0.12:8080", "set server listening address")
	flag.StringVar(&cfg.StoreFile, "f", "/Users/leonidagupov/Dev/surfshit/surfweather.json", "directory for saving metrics")
	flag.BoolVar(&cfg.Restore, "r", true, "recovering from backup before start")
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Printf("env parse failed :%s", err)
	}
	return &cfg
}

func (r *MeteoAPIData) GetMeteoData(start, stop string, client *resty.Client) (body []byte, err error) {
	resp, err := client.R().SetQueryParams(map[string]string{
		"lat":    "32.801263",
		"lng":    "34.956112",
		"params": "swellHeight,swellPeriod,swellDirection,waterTemperature,windSpeed",
		"start":  start,
		"end":    stop,
		"source": "sg",
	}).SetHeader("Authorization", r.cfg.Key).
		Get("https://api.stormglass.io/v2/weather/point")
	if err != nil {
		err = fmt.Errorf("error at %s", err)
		return
	}

	body = resp.Body()

	return body, err
}

func (r *MeteoAPIData) FormatReport(weather []byte) (formattedInfo string, err error) {

	var info SurfInfo

	err = json.Unmarshal(weather, &info)
	if err != nil {
		err = fmt.Errorf("error during unmarshalling %s", err)
		return
	}

	for k := range info.Hours {
		formattedInfo += fmt.Sprintf(
			"Во время %s \n Высота волн: %.2f, Направление волн %.2f, Частота волн %.2f \n",
			info.Hours[k].Time.String(),
			info.Hours[k].SwellHeight.Sg,
			info.Hours[k].SwellDirection.Sg,
			info.Hours[k].SwellPeriod.Sg,
		)
	}
	return
}

func (r *MeteoAPIData) ReportCount(weather []byte) (count int, err error) {
	var info SurfInfo

	err = json.Unmarshal(weather, &info)
	if err != nil {
		err = fmt.Errorf("error during unmarshalling %s", err)
		return
	}

	count = 10 - info.Meta.RequestCount
	if count < 0 {
		err = fmt.Errorf("апи блокнулось, приходи завтра - будет еще 10 запросов к апи")
		return
	}
	return
}

func (r *MeteoAPIData) SaveToFile(data []byte) (err error) {

	file, err := os.OpenFile(r.cfg.StoreFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)

	file.Truncate(0)

	_, err = writer.Write(data)
	if err != nil {
		err = fmt.Errorf("error during writing %q", err)
		return
	}

	return writer.Flush()

}

func (r *MeteoAPIData) LoadFromFile() (data []byte, err error) {

	file, err := os.OpenFile(r.cfg.StoreFile, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(file)

	data, err = io.ReadAll(reader)
	if err != nil {
		log.Printf("read err : %s", err)
		return
	}
	log.Print(string(data))
	return
}

func (contr *MeteoController) UpdateWeatherReport(ctx *gin.Context) {
	var status string
	start := fmt.Sprint(time.Now().UTC().Unix())
	stop := fmt.Sprint(time.Now().Add(time.Hour * 168).Unix())

	weather, err := contr.api.GetMeteoData(start, stop, &contr.client)
	if err != nil {
		log.Printf("%s", err)
		return
	}

	err = contr.api.SaveToFile(weather)
	if err != nil {
		log.Printf("%s", err)
	}

	count, err := contr.api.ReportCount(weather)
	if err != nil {
		log.Printf("error during counting: %s", err)
	}

	if err != nil {
		status = fmt.Sprintf("чота наебнулось: %s", err)
	} else {
		status = fmt.Sprintf("всё балдеж, иди на главную, кстати сегодня осталось всего %d обновлений", count)
	}

	ctx.HTML(http.StatusOK, "update.tmpl", gin.H{
		"status": status,
	})
}

func (contr *MeteoController) WeatherReport(ctx *gin.Context) {

	weather, err := contr.api.LoadFromFile()
	if err != nil {
		log.Printf("error: %s", err)
	}

	report, err := contr.api.FormatReport(weather)
	if err != nil {
		log.Printf("%s", err)
	}

	ctx.String(http.StatusOK, report)

}
