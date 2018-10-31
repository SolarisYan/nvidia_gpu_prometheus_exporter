package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	// "sync"

	"github.com/mindprince/gonvml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "nvidia_gpu"
)

var (
	addr = flag.String("web.listen-address", ":9445", "Address to listen on for web interface and telemetry.")
	// metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	// labels = []string{"minor_number", "uuid", "name"}
)

	// var (
	// 	listenAddress = flag.String("web.listen-address", ":9401", "Address to listen on for web interface and telemetry.")
	// 	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	// )
	// flag.Parse()

// type Collector struct {
// 	sync.Mutex
// 	numDevices  prometheus.Gauge
// 	usedMemory  *prometheus.GaugeVec
// 	totalMemory *prometheus.GaugeVec
// 	dutyCycle   *prometheus.GaugeVec
// 	powerUsage  *prometheus.GaugeVec
// 	temperature *prometheus.GaugeVec
// 	fanSpeed    *prometheus.GaugeVec
// }

// func NewCollector() *Collector {
// 	return &Collector{
// 		numDevices: prometheus.NewGauge(
// 			prometheus.GaugeOpts{
// 				Namespace: namespace,
// 				Name:      "num_devices",
// 				Help:      "Number of GPU devices",
// 			},
// 		),
// 		usedMemory: prometheus.NewGaugeVec(
// 			prometheus.GaugeOpts{
// 				Namespace: namespace,
// 				Name:      "memory_used_bytes",
// 				Help:      "Memory used by the GPU device in bytes",
// 			},
// 			labels,
// 		),
// 		totalMemory: prometheus.NewGaugeVec(
// 			prometheus.GaugeOpts{
// 				Namespace: namespace,
// 				Name:      "memory_total_bytes",
// 				Help:      "Total memory of the GPU device in bytes",
// 			},
// 			labels,
// 		),
// 		dutyCycle: prometheus.NewGaugeVec(
// 			prometheus.GaugeOpts{
// 				Namespace: namespace,
// 				Name:      "duty_cycle",
// 				Help:      "Percent of time over the past sample period during which one or more kernels were executing on the GPU device",
// 			},
// 			labels,
// 		),
// 		powerUsage: prometheus.NewGaugeVec(
// 			prometheus.GaugeOpts{
// 				Namespace: namespace,
// 				Name:      "power_usage_milliwatts",
// 				Help:      "Power usage of the GPU device in milliwatts",
// 			},
// 			labels,
// 		),
// 		temperature: prometheus.NewGaugeVec(
// 			prometheus.GaugeOpts{
// 				Namespace: namespace,
// 				Name:      "temperature_celsius",
// 				Help:      "Temperature of the GPU device in celsius",
// 			},
// 			labels,
// 		),
// 		fanSpeed: prometheus.NewGaugeVec(
// 			prometheus.GaugeOpts{
// 				Namespace: namespace,
// 				Name:      "fanspeed_percent",
// 				Help:      "Fanspeed of the GPU device as a percent of its maximum",
// 			},
// 			labels,
// 		),
// 	}
// }

// func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
// 	ch <- c.numDevices.Desc()
// 	c.usedMemory.Describe(ch)
// 	c.totalMemory.Describe(ch)
// 	c.dutyCycle.Describe(ch)
// 	c.powerUsage.Describe(ch)
// 	c.temperature.Describe(ch)
// 	c.fanSpeed.Describe(ch)
// }

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	// Only one Collect call in progress at a time.
	c.Lock()
	defer c.Unlock()

	c.usedMemory.Reset()
	c.totalMemory.Reset()
	c.dutyCycle.Reset()
	c.powerUsage.Reset()
	c.temperature.Reset()
	c.fanSpeed.Reset()

	numDevices, err := gonvml.DeviceCount()
	if err != nil {
		log.Printf("DeviceCount() error: %v", err)
		return
	} else {
		c.numDevices.Set(float64(numDevices))
		ch <- c.numDevices
	}

	for i := 0; i < int(numDevices); i++ {
		dev, err := gonvml.DeviceHandleByIndex(uint(i))
		if err != nil {
			log.Printf("DeviceHandleByIndex(%d) error: %v", i, err)
			continue
		}

		minorNumber, err := dev.MinorNumber()
		if err != nil {
			log.Printf("MinorNumber() error: %v", err)
			continue
		}
		minor := strconv.Itoa(int(minorNumber))

		uuid, err := dev.UUID()
		if err != nil {
			log.Printf("UUID() error: %v", err)
			continue
		}

		name, err := dev.Name()
		if err != nil {
			log.Printf("Name() error: %v", err)
			continue
		}

		totalMemory, usedMemory, err := dev.MemoryInfo()
		if err != nil {
			log.Printf("MemoryInfo() error: %v", err)
		} else {
			c.usedMemory.WithLabelValues(minor, uuid, name).Set(float64(usedMemory))
			c.totalMemory.WithLabelValues(minor, uuid, name).Set(float64(totalMemory))
		}

		dutyCycle, _, err := dev.UtilizationRates()
		if err != nil {
			log.Printf("UtilizationRates() error: %v", err)
		} else {
			c.dutyCycle.WithLabelValues(minor, uuid, name).Set(float64(dutyCycle))
		}

		powerUsage, err := dev.PowerUsage()
		if err != nil {
			log.Printf("PowerUsage() error: %v", err)
		} else {
			c.powerUsage.WithLabelValues(minor, uuid, name).Set(float64(powerUsage))
		}

		temperature, err := dev.Temperature()
		if err != nil {
			log.Printf("Temperature() error: %v", err)
		} else {
			c.temperature.WithLabelValues(minor, uuid, name).Set(float64(temperature))
		}

		fanSpeed, err := dev.FanSpeed()
		if err != nil {
			log.Printf("FanSpeed() error: %v", err)
		} else {
			c.fanSpeed.WithLabelValues(minor, uuid, name).Set(float64(fanSpeed))
		}
	}
	c.usedMemory.Collect(ch)
	c.totalMemory.Collect(ch)
	c.dutyCycle.Collect(ch)
	c.powerUsage.Collect(ch)
	c.temperature.Collect(ch)
	c.fanSpeed.Collect(ch)
}

type Exporter struct {
	up                    prometheus.Gauge
	info                  *prometheus.GaugeVec
	deviceCount           prometheus.Gauge
	temperatures          *prometheus.GaugeVec
	deviceInfo            *prometheus.GaugeVec
	powerUsage            *prometheus.GaugeVec
	powerUsageAverage     *prometheus.GaugeVec
	fanSpeed              *prometheus.GaugeVec
	memoryTotal           *prometheus.GaugeVec
	memoryUsed            *prometheus.GaugeVec
	utilizationMemory     *prometheus.GaugeVec
	utilizationGPU        *prometheus.GaugeVec
	utilizationGPUAverage *prometheus.GaugeVec
}

func NewExporter() *Exporter {
	return &Exporter{
		up: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "up",
				Help:      "NVML Metric Collection Operational",
			},
		),
		info: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "driver_info",
				Help:      "NVML Info",
			},
			[]string{"version"},
		),
		deviceCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "device_count",
				Help:      "Count of found nvidia devices",
			},
		),
		deviceInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "info",
				Help:      "Info as reported by the device",
			},
			[]string{"index", "minor", "uuid", "name"},
		),
		temperatures: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "temperatures",
				Help:      "Temperature as reported by the device",
			},
			[]string{"minor"},
		),
		powerUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "power_usage",
				Help:      "Power usage as reported by the device",
			},
			[]string{"minor"},
		),
		powerUsageAverage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "power_usage_average",
				Help:      "Power usage as reported by the device averaged over 10s",
			},
			[]string{"minor"},
		),
		fanSpeed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "fanspeed",
				Help:      "Fan speed as reported by the device",
			},
			[]string{"minor"},
		),
		memoryTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_total",
				Help:      "Total memory as reported by the device",
			},
			[]string{"minor"},
		),
		memoryUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_used",
				Help:      "Used memory as reported by the device",
			},
			[]string{"minor"},
		),
		utilizationMemory: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "utilization_memory",
				Help:      "Memory Utilization as reported by the device",
			},
			[]string{"minor"},
		),
		utilizationGPU: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "utilization_gpu",
				Help:      "GPU utilization as reported by the device",
			},
			[]string{"minor"},
		),
		utilizationGPUAverage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "utilization_gpu_average",
				Help:      "Used memory as reported by the device averraged over 10s",
			},
			[]string{"minor"},
		),
	}
}

func (e *Exporter) Collect(metrics chan<- prometheus.Metric) {
	data, err := collectMetrics()
	if err != nil {
		log.Printf("Failed to collect metrics: %s\n", err)
		e.up.Set(0)
		e.up.Collect(metrics)
		return
	}

	e.up.Set(1)
	e.info.WithLabelValues(data.Version).Set(1)
	e.deviceCount.Set(float64(len(data.Devices)))

	for i := 0; i < len(data.Devices); i++ {
		d := data.Devices[i]
		e.deviceInfo.WithLabelValues(d.Index, d.MinorNumber, d.Name, d.UUID).Set(1)
		e.fanSpeed.WithLabelValues(d.MinorNumber).Set(d.FanSpeed)
		e.memoryTotal.WithLabelValues(d.MinorNumber).Set(d.MemoryTotal)
		e.memoryUsed.WithLabelValues(d.MinorNumber).Set(d.MemoryUsed)
		e.powerUsage.WithLabelValues(d.MinorNumber).Set(d.PowerUsage)
		e.powerUsageAverage.WithLabelValues(d.MinorNumber).Set(d.PowerUsageAverage)
		e.temperatures.WithLabelValues(d.MinorNumber).Set(d.Temperature)
		e.utilizationGPU.WithLabelValues(d.MinorNumber).Set(d.UtilizationGPU)
		e.utilizationGPUAverage.WithLabelValues(d.MinorNumber).Set(d.UtilizationGPUAverage)
		e.utilizationMemory.WithLabelValues(d.MinorNumber).Set(d.UtilizationMemory)
	}

	e.deviceCount.Collect(metrics)
	e.deviceInfo.Collect(metrics)
	e.fanSpeed.Collect(metrics)
	e.info.Collect(metrics)
	e.memoryTotal.Collect(metrics)
	e.memoryUsed.Collect(metrics)
	e.powerUsage.Collect(metrics)
	e.powerUsageAverage.Collect(metrics)
	e.temperatures.Collect(metrics)
	e.up.Collect(metrics)
	e.utilizationGPU.Collect(metrics)
	e.utilizationGPUAverage.Collect(metrics)
	e.utilizationMemory.Collect(metrics)
}

func (e *Exporter) Describe(descs chan<- *prometheus.Desc) {
	e.deviceCount.Describe(descs)
	e.deviceInfo.Describe(descs)
	e.fanSpeed.Describe(descs)
	e.info.Describe(descs)
	e.memoryTotal.Describe(descs)
	e.memoryUsed.Describe(descs)
	e.powerUsage.Describe(descs)
	e.powerUsageAverage.Describe(descs)
	e.temperatures.Describe(descs)
	e.up.Describe(descs)
	e.utilizationGPU.Describe(descs)
	e.utilizationGPUAverage.Describe(descs)
	e.utilizationMemory.Describe(descs)
}

func main() {
	flag.Parse()

	// if err := gonvml.Initialize(); err != nil {
	// 	log.Fatalf("Couldn't initialize gonvml: %v. Make sure NVML is in the shared library search path.", err)
	// }
	// defer gonvml.Shutdown()

	if driverVersion, err := gonvml.SystemDriverVersion(); err != nil {
		log.Printf("SystemDriverVersion() error: %v", err)
	} else {
		log.Printf("SystemDriverVersion(): %v", driverVersion)
	}

	// prometheus.MustRegister(NewCollector())
	prometheus.MustRegister(NewExporter())

	// Serve on all paths under addr
	log.Fatalf("ListenAndServe error: %v", http.ListenAndServe(*addr, promhttp.Handler()))
}
