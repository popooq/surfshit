package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-resty/resty/v2"
)

type SurfInfo struct {
	Hours []struct {
		SwellDirection struct {
			Sg float64 `json:"sg"`
		} `json:"swellDirection"`
		SwellHeight struct {
			Sg float64 `json:"sg"`
		} `json:"swellHeight"`
		SwellPeriod struct {
			Sg float64 `json:"sg"`
		} `json:"swellPeriod"`
		Time             time.Time `json:"time"`
		WaterTemperature struct {
			Sg float64 `json:"sg"`
		} `json:"waterTemperature"`
		WindSpeed struct {
			Sg float64 `json:"sg"`
		} `json:"windSpeed"`
	} `json:"hours"`
	Meta struct {
		Cost         int      `json:"cost"`
		DailyQuota   int      `json:"dailyQuota"`
		End          string   `json:"end"`
		Lat          float64  `json:"lat"`
		Lng          float64  `json:"lng"`
		Params       []string `json:"params"`
		RequestCount int      `json:"requestCount"`
		Source       []string `json:"source"`
		Start        string   `json:"start"`
	} `json:"meta"`
}

func main() {
	var info SurfInfo
	client := resty.New()
	key := "8693c2da-9ca8-11ed-bce5-0242ac130002-8693c3c0-9ca8-11ed-bce5-0242ac130002"
	now := time.Now().UTC()
	start := fmt.Sprint(now.Unix())
	end := now.Add(time.Hour * 24)
	stop := fmt.Sprint(end.Unix())

	resp, err := client.R().SetQueryParams(map[string]string{
		"lat":    "32.801263",
		"lng":    "34.956112",
		"params": "swellHeight,swellPeriod,swellDirection,waterTemperature,windSpeed",
		"start":  start,
		"end":    stop,
		"source": "sg",
	}).SetHeader("Authorization", key).
		Get("https://api.stormglass.io/v2/weather/point")
	if err != nil {
		log.Printf("error at %s", err)
	}

	err = json.Unmarshal(resp.Body(), &info)
	if err != nil {
		log.Printf("error during unmarshalling %s", err)
	}

	fmt.Println("Response Info:")
	fmt.Println("  Error      :", err)
	fmt.Println("  Status Code:", resp.StatusCode())
	fmt.Println("  Status     :", resp.Status())
	fmt.Println("  Proto      :", resp.Proto())
	fmt.Println("  Time       :", resp.Time())
	fmt.Println("  Received At:", resp.ReceivedAt())
	fmt.Println("  Body       :\n", resp)
	fmt.Println()
}
