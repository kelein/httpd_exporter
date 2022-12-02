package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/log"
)

const (
	namespace = "apache"
)

var (
	listenAddress   = flag.String("telemetry.address", ":9113", "Address on which to expose metrics.")
	metricsPath     = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metrics.")
	apacheScrapeURI = flag.String("scrape_uri", "http://localhost/server-status", "URI to apache server status page.")
	insecure        = flag.Bool("insecure", true, "Ignore server certificate if using https")
)

// Exporter for apache server status
type Exporter struct {
	URI               string
	mutex             sync.RWMutex
	client            *http.Client
	version           *prometheus.GaugeVec
	cpuLoad           prometheus.Gauge
	serverUptime      prometheus.Gauge
	totalKBytes       prometheus.Gauge
	totalWorkers      *prometheus.GaugeVec
	totalAccesses     prometheus.Gauge
	totalProcesses    prometheus.Gauge
	bytesPerSecond    prometheus.Gauge
	bytesPerRequest   prometheus.Gauge
	requestsPerSecond prometheus.Gauge
	scrapeFailures    prometheus.Counter
}

// NewApacheExporter create a exporter instance
func NewApacheExporter(uri string) *Exporter {
	return &Exporter{
		URI: uri,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: *insecure},
			},
		},
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_scrape_failures_total",
			Help:      "Number of errors while scraping apache2.",
		}),
		version: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "version",
			Help:      "Apache version",
		},
			[]string{"version"},
		),
		serverUptime: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "server_uptime",
			Help:      "Apache server uptime",
		}),
		totalKBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "total_kbytes",
			Help:      "Total Bytest in KB",
		}),
		totalAccesses: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "total_accesses",
			Help:      "Total Accesses",
		}),
		cpuLoad: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cpu_load",
			Help:      "CPU Load in %",
		}),
		requestsPerSecond: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "requests_per_second",
			Help:      "Requests per second",
		}),
		bytesPerSecond: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "bytes_per_second",
			Help:      "Bytes per second",
		}),
		bytesPerRequest: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "bytes_per_request",
			Help:      "Bytes per request",
		}),
		totalProcesses: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "total_processes",
			Help:      "Apache httpd process number, sum of busyworkers & idleworkers.",
		}),
		totalWorkers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "total_workers",
			Help:      "Apache worker number with busy or idle label.",
		},
			[]string{"worker"},
		),
	}
}

// Describe implements prometheus Collector's Describe method
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.version.Describe(ch)
	e.totalKBytes.Describe(ch)
	e.totalAccesses.Describe(ch)
	e.cpuLoad.Describe(ch)
	e.requestsPerSecond.Describe(ch)
	e.bytesPerSecond.Describe(ch)
	e.bytesPerRequest.Describe(ch)
	e.requestsPerSecond.Describe(ch)
	e.scrapeFailures.Describe(ch)
	e.totalKBytes.Describe(ch)
	e.serverUptime.Describe(ch)
	e.totalProcesses.Describe(ch)
	e.totalWorkers.Describe(ch)
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) error {
	html, err := goquery.NewDocument(e.URI)
	if err != nil {
		return fmt.Errorf("Scrape Apache Server Status Error: %v", err)
	}

	var minVersion string

	dt := html.Find("dt")
	dt.Each(func(_ int, sel *goquery.Selection) {
		content := strings.TrimSpace(sel.Text())
		if strings.Contains(content, "Server Version") {
			minVersion = strings.Split(content, "/")[1]
			minVersion = strings.Split(minVersion, " ")[0]
		}
	})
	maxVersion, _ := strconv.ParseFloat(minVersion[0:3], 64)
	e.version.WithLabelValues(minVersion).Set(float64(maxVersion))
	return nil
}

func (e *Exporter) scrapeBasic(ch chan<- prometheus.Metric) error {
	URL := e.URI + "?auto"
	resp, err := e.client.Get(URL)
	if err != nil {
		return fmt.Errorf("Error scraping apache status via auto: %v", err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	log.Debugln(data)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if err != nil {
			data = []byte(err.Error())
		}
		return fmt.Errorf("Status %s (%d): %s", resp.Status, resp.StatusCode, data)
	}

	lines := strings.Split(string(data), "\n")
	var (
		Uptime        string
		CPULoad       string
		ReqPerSec     string
		BytesPerSec   string
		BytesPerReq   string
		BusyWorkers   string
		IdleWorkers   string
		TotalKBytes   string
		TotalAccesses string
	)

	for _, line := range lines {
		metric := strings.Split(line, ":")
		if strings.Contains(metric[0], "Total Accesses") {
			TotalAccesses = strings.TrimSpace(metric[1])
		}
		if strings.Contains(metric[0], "Total kBytes") {
			TotalKBytes = strings.TrimSpace(metric[1])
		}
		if strings.Contains(metric[0], "CPULoad") {
			CPULoad = strings.TrimSpace(metric[1])
		}
		if strings.Contains(metric[0], "Uptime") {
			Uptime = strings.TrimSpace(metric[1])
		}
		if strings.Contains(metric[0], "ReqPerSec") {
			ReqPerSec = strings.TrimSpace(metric[1])
		}
		if strings.Contains(metric[0], "BytesPerSec") {
			BytesPerSec = strings.TrimSpace(metric[1])
		}
		if strings.Contains(metric[0], "BytesPerReq") {
			BytesPerReq = strings.TrimSpace(metric[1])
		}
		if strings.Contains(metric[0], "BusyWorkers") {
			BusyWorkers = strings.TrimSpace(metric[1])
		}
		if strings.Contains(metric[0], "IdleWorkers") {
			IdleWorkers = strings.TrimSpace(metric[1])
		}
	}

	totalAccess, _ := strconv.ParseFloat(TotalAccesses, 64)
	totalKBytes, _ := strconv.ParseFloat(TotalKBytes, 64)
	cpuLoad, _ := strconv.ParseFloat(CPULoad, 64)
	serverUptime, _ := strconv.ParseFloat(Uptime, 64)
	reqPerSec, _ := strconv.ParseFloat(ReqPerSec, 64)
	bytePerSec, _ := strconv.ParseFloat(BytesPerSec, 64)
	bytesPerReq, _ := strconv.ParseFloat(BytesPerReq, 64)
	busyWorkers, _ := strconv.ParseFloat(BusyWorkers, 64)
	idleWorkers, _ := strconv.ParseFloat(IdleWorkers, 64)
	totalProcesses := busyWorkers + idleWorkers

	e.totalAccesses.Set(float64(totalAccess))
	e.totalKBytes.Set(float64(totalKBytes))
	e.cpuLoad.Set(float64(cpuLoad))
	e.serverUptime.Set(float64(serverUptime))
	e.requestsPerSecond.Set(float64(reqPerSec))
	e.bytesPerSecond.Set(float64(bytePerSec))
	e.bytesPerRequest.Set(float64(bytesPerReq))
	e.totalProcesses.Set(float64(totalProcesses))
	e.totalWorkers.WithLabelValues("busy").Set(float64(busyWorkers))
	e.totalWorkers.WithLabelValues("idle").Set(float64(idleWorkers))

	return nil
}

// Collect implements prometheus Collector's Collect method
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if err := e.scrape(ch); err != nil {
		log.Printf("Error scraping apache: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	if err := e.scrapeBasic(ch); err != nil {
		log.Printf("Error scraping data via auto: %s", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}

	e.version.Collect(ch)
	e.cpuLoad.Collect(ch)
	e.serverUptime.Collect(ch)
	e.totalKBytes.Collect(ch)
	e.totalWorkers.Collect(ch)
	e.totalAccesses.Collect(ch)
	e.totalProcesses.Collect(ch)
	e.bytesPerSecond.Collect(ch)
	e.bytesPerRequest.Collect(ch)
	e.requestsPerSecond.Collect(ch)
	return
}

func main() {
	flag.Parse()

	exporter := NewApacheExporter(*apacheScrapeURI)
	prometheus.MustRegister(exporter)
	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
                <head><title>Apache Exporter</title></head>
                <body>
                   <h1>Apache Exporter</h1>
                   <p><a href='` + *metricsPath + `'>Metrics</a></p>
                   </body>
                </html>
              `))
	})
	log.Infof("Starting Server: %s", *listenAddress)
	log.Infof("Scraping URL: %s", *apacheScrapeURI)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
