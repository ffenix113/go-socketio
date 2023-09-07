package engineio

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	CurrentClients prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		CurrentClients: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "pleasetalk",
			Subsystem: "engineio",
			Name:      "current_clients",
			Help:      "Number of current clients",
		}),
	}

	if reg != nil {
		reg.MustRegister(m.CurrentClients)
	}

	return m
}
