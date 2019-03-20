// Copyright 2019 Yoshi Yamaguchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/devices/apa102"
	"periph.io/x/periph/devices/bmxx80"
	"periph.io/x/periph/experimental/devices/rainbowhat"
	"periph.io/x/periph/host"
)

const (
	TickInterval   = 10 * time.Second
	ReportInterval = 60 * time.Second

	MeasureTemperature = "temperature"
	MeasurePressure    = "pressure"

	TemperatureUnit = "C"
	PressureUnit    = "hPa"

	WeatherLon = 139.7041
	WeatherLat = 35.6618

	ResourceNamespace = "ymotongpoo"
)

var (
	MTemperature = stats.Float64(MeasureTemperature, "air temperature", TemperatureUnit)
	MPressure    = stats.Float64(MeasurePressure, "barometic pressure", PressureUnit)

	KeyRainfall, _ = tag.NewKey("rainfall")

	TemperatureView = &view.View{
		Name:        MeasureTemperature,
		Measure:     MTemperature,
		Description: "air temperature",
		TagKeys:     []tag.Key{KeyRainfall},
		Aggregation: view.LastValue(),
	}
	PressureView = &view.View{
		Name:        MeasurePressure,
		Measure:     MPressure,
		Description: "barometric pressure",
		TagKeys:     []tag.Key{KeyRainfall},
		Aggregation: view.LastValue(),
	}

	Hostname, _ = os.Hostname()
)

func main() {
	hat, err := InitHAT()
	if err != nil {
		log.Fatalf("failed to initialize Rainbow HAT: %v", err)
	}
	defer hat.Halt()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sig)

	// OpenCensus + Stackdriver Monitoring settings
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v\n", err)
	}
	labels := &stackdriver.Labels{}
	labels.Set("location", "asia-northeast1-a", "")
	labels.Set("namespace", "ymotongpoo", "")
	labels.Set("node_id", hostname, "")
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               os.Getenv("GOOGLE_CLOUD_PROJECT"),
		Location:                "asia-northeast1-a",
		MonitoredResource:       &GenericNodeMonitoredResource{},
		DefaultMonitoringLabels: labels,
		GetMetricType:           GetMetricType,
	})
	if err != nil {
		log.Fatalf("Failed to create Stackdriver exporter: %v\n", err)
	}
	defer exporter.Flush()
	view.SetReportingPeriod(ReportInterval)
	view.RegisterExporter(exporter)
	if err := view.Register(TemperatureView, PressureView); err != nil {
		log.Fatalf("Failed to enable views: %v\n", err)
	}

	LoopSensing(hat, sig)
}

//---- Rainbow HAT ----

type BMP280Data struct {
	Temperature float64
	Pressure    float64
}

func InitHAT() (*rainbowhat.Dev, error) {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}
	return rainbowhat.NewRainbowHat(&apa102.DefaultOpts)
}

func LoopSensing(hat *rainbowhat.Dev, sig chan os.Signal) {
	sensor := hat.GetBmp280()
	ticker := time.NewTicker(TickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			d, err := GetSensorData(sensor)
			if err != nil {
				log.Printf("error on sensing BMP280: %v\n", err)
				break
			}
			err = RecordMeasurement(d)
		case s := <-sig:
			fmt.Println(s)
			os.Exit(0)
		}
	}
}

func GetSensorData(sensor *bmxx80.Dev) (BMP280Data, error) {
	var env physic.Env
	if err := sensor.Sense(&env); err != nil {
		return BMP280Data{}, err
	}
	d := BMP280Data{
		Temperature: float64(env.Temperature-physic.ZeroCelsius) / float64(physic.Kelvin),
		Pressure:    float64(env.Pressure) / float64(100*physic.Pascal),
	}
	return d, nil
}

//---- OpenCensus + Stackdriver ----

type GenericNodeMonitoredResource struct{}

func (mr *GenericNodeMonitoredResource) MonitoredResource() (string, map[string]string) {
	labels := map[string]string{
		"location":  "asia-northeast1-a",
		"namespace": "ymotongpoo",
		"node_id":   Hostname,
	}
	return "generic_node", labels
}

func RecordMeasurement(data BMP280Data) error {
	rf, err := fetchRainfall(WeatherLon, WeatherLat)
	if err != nil {
		return err
	}
	ctx, err := tag.New(context.Background(), tag.Insert(KeyRainfall, strconv.Itoa(rf)))
	if err != nil {
		return err
	}
	stats.Record(ctx, MTemperature.M(data.Temperature), MPressure.M(data.Pressure))
	return nil
}

func GetMetricType(v *view.View) string {
	return fmt.Sprintf("custom.googleapis.com/%s", v.Name)
}

//---- rainfall data ----

type RainfallData struct {
	Feature []struct {
		Property struct {
			WeatherList struct {
				Weather []struct {
					Type     string  `json:"Type"`
					Rainfall float64 `json:"Rainfall"`
				} `json:"Weather"`
			} `json:"WeatherList"`
		} `json:"Property"`
	} `json:"Feature"`
}

func fetchRainfall(lon, lat float64) (int, error) {
	appId := os.Getenv("YAHOO_APP_ID")
	if appId == "" {
		return -1, fmt.Errorf("Set Yahoo! Japan App ID from developers dashboard: https://e.developer.yahoo.co.jp/dashboard/")
	}
	url := fmt.Sprintf("https://map.yahooapis.jp/weather/V1/place?coordinates=%f,%f&appid=%s&output=json",
		lon, lat, appId)
	resp, err := http.Get(url)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	r := RainfallData{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&r); err != nil {
		return -1, err
	}
	weathers := r.Feature[0].Property.WeatherList.Weather
	for _, w := range weathers {
		if w.Type == "observation" {
			return int(w.Rainfall), nil
		}
	}
	return -1, nil
}
