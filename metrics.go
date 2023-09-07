package socketio

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	TotalSockets prometheus.Gauge

	EmitForUserCalls prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		TotalSockets: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "pleasetalk",
			Subsystem: "socketio",
			Name:      "total_sockets",
			Help:      "Number of total sockets. Each user can have multiple sockets.",
		}),
		EmitForUserCalls: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "pleasetalk",
			Subsystem: "socketio",
			Name:      "emits_total",
			Help:      "Number of calls to EmitForUser.",
		}),
	}

	if reg != nil {
		reg.MustRegister(
			m.TotalSockets,
			m.EmitForUserCalls,
		)
	}

	return m
}
