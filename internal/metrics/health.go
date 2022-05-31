package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/helper"
)

func init() {
	Registry.Add("health", newhealthCollector)
}

type healthCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newhealthCollector() collector.Collector {
	c := &healthCollector{}
	c.init()
	return c
}

func (c *healthCollector) init() {
	c.props = []string{"voltage", "temperature", "cpu-temperature"}

	labelNames := []string{"name", "address"}
	helpText := []string{"Input voltage to the RouterOS board, in volts", "Temperature of RouterOS board, in degrees Celsius", "Temperature of RouterOS CPU, in degrees Celsius"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for i, p := range c.props {
		c.descriptions[p] = helper.DescriptionForPropertyNameHelpText("health", p, labelNames, helpText[i])
	}
}

func (c *healthCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *healthCollector) Collect(ctx *collector.Context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *healthCollector) fetch(ctx *collector.Context) ([]*proto.Sentence, error) {
	reply, err := ctx.Client.Run("/system/health/print", "=.proplist=name,value,type")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching system health metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *healthCollector) collectForStat(re *proto.Sentence, ctx *collector.Context) {
	c.collectMetricForProperty(re.Map["name"], re.Map["value"], ctx)
}

func (c *healthCollector) collectMetricForProperty(property, value string, ctx *collector.Context) {
	if value == "" {
		return
	}

	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.Device.Name,
			"property": property,
			"value":    value,
			"error":    err,
		}).Error("error parsing system health metric value")
		return
	}

	desc := c.descriptions[property]
	ctx.Ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address)
}
