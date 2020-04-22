package metricsadapter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type GardenMemStats struct {
	Alloc float64 `json:"Alloc"`
}

type GardenDebugMetrics struct {
	NumGoroutines int            `json:"numGoroutines"`
	Memstats      GardenMemStats `json:"memstats"`
}

type Metrics []Metric

type Series struct {
	Series Metrics `json:"series"`
}

type Metric struct {
	Metric string              `json:"metric"`
	Points MetricPoints `json:"points"`
	Host   string              `json:"host"`
	Tags   []string            `json:"tags"`
}

type MetricPoints [][2]float64

func fromGardenDebugMetrics(m GardenDebugMetrics, host string) Series {
	now := time.Now().Unix()
	return Series{
		Series: Metrics{
			Metric{
				Metric: "garden.numGoroutines",
				Points: MetricPoints{[2]float64{float64(now), float64(m.NumGoroutines)}},
				Host:   host,
				Tags:   []string{},
			},
			Metric{
				Metric: "garden.memory",
				Points: MetricPoints{[2]float64{float64(now), float64(m.Memstats.Alloc)}},
				Host:   host,
				Tags:   []string{},
			},
		},
	}
}

func CollectMetrics(url, host string) (Series, error) {
	body, err := getResponseBody(url)
	if err != nil {
		return Series{}, err
	}

	var gardenDebugMetrics GardenDebugMetrics
	err = json.Unmarshal(body, &gardenDebugMetrics)
	if err != nil {
		return Series{}, err
	}

	return fromGardenDebugMetrics(gardenDebugMetrics, host), nil
}

func getResponseBody(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}

func EmitMetrics(metrics Series, wavefrontAddress string) error {
	conn, err := net.Dial("tcp", wavefrontAddress)
	if err != nil {
		return err
	}
	defer conn.Close()

	for _, m := range metrics.Series {
		for _, p := range m.Points {
			_, err := fmt.Fprintf(conn, "%s %f %f source=%s\n", m.Metric, p[1], p[0], m.Host)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
