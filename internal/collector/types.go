package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/routeros.v2"

	"mikrotik-exporter/internal/config"
)

type Context struct {
	Ch     chan<- prometheus.Metric
	Device *config.Device
	Client *routeros.Client
}

type Collector interface {
	Describe(ch chan<- *prometheus.Desc)
	Collect(ctx *Context) error
}
