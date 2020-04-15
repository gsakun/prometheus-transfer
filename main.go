package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var metriclist []string

type Exporter struct {
	gauge map[string]prometheus.Gauge
}

func generatelabels() map[string]string {
	var labels map[string]string = make(map[string]string)
	labels["pod_name"] = os.Getenv("PODNAME")
	labels["pod_ip"] = os.Getenv("POD_IP")
	labels["pod_namespace"] = os.Getenv("POD_NAMESPACE")
	return labels
}

func NewExporter(metricsname []string) *Exporter {
	labels := generatelabels()
	var list map[string]prometheus.Gauge = make(map[string]prometheus.Gauge)
	for _, i := range metricsname {
		list[i] = prometheus.NewGauge(prometheus.GaugeOpts{
			ConstLabels: labels,
			Name:        i,
			Help:        fmt.Sprintf("This is a gauge metric for %s", i)})
	}
	return &Exporter{
		gauge: list,
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	data := queryMetric()
	log.Infoln(data)
	for key, value := range data {
		e.gauge[key].Set(value)
		e.gauge[key].Collect(ch)
	}
}

func queryMetric() (data map[string]float64) {
	url := os.Getenv("URI")
	req, err := http.NewRequest("POST", url, nil)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorln(err)
	}
	defer resp.Body.Close()
	if err != nil {
		log.Errorf("querydata error %v")
	}
	defer resp.Body.Close()
	s, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("querydata error %v")
	}
	_ = json.Unmarshal(s, &data)
	return
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, i := range e.gauge {
		i.Describe(ch)
	}
}

func init() {
	data := queryMetric()
	if len(data) != 0 {
		for k, _ := range data {
			metriclist = append(metriclist, k)
		}
	}
}

func main() {
	metricsPath := "/metrics"
	listenAddress := "0.0.0.0:16666"
	if len(metriclist) == 0 {
		log.Fatalln("App's metric list is nil,please check app metric interface")
	}
	exporter := NewExporter(metriclist)
	prometheus.MustRegister(exporter)
	http.Handle(metricsPath, promhttp.Handler())
	fmt.Println(http.ListenAndServe(listenAddress, nil))
}
