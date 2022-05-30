package internal

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/helper"
)

type firmwareCollector struct {
	props       []string
	description *prometheus.Desc
}

func newFirmwareCollector() collector.Collector {
	c := &firmwareCollector{}
	c.init()
	return c
}

func (c *firmwareCollector) init() {
	labelNames := []string{"devicename", "name", "disabled", "version", "build_time"}
	c.description = helper.Description("system", "package", "system packages version", labelNames)
}

func (c *firmwareCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.description
}

func (c *firmwareCollector) Collect(ctx *collector.Context) error {
	reply, err := ctx.Client.Run("/system/package/getall")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		})
		return err
	}

	pkgs := reply.Re

	for _, pkg := range pkgs {
		v := 1.0
		if strings.EqualFold(pkg.Map["disabled"], "true") {
			v = 0.0
		}
		ctx.Ch <- prometheus.MustNewConstMetric(c.description, prometheus.GaugeValue, v, ctx.Device.Name, pkg.Map["name"], pkg.Map["disabled"], pkg.Map["version"], pkg.Map["build-time"])
	}

	return nil
}
