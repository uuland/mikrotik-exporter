package metrics

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/helper"
)

func init() {
	Registry.Add("netwatch", newNetwatchCollector)
}

type netwatchCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newNetwatchCollector() collector.Collector {
	c := &netwatchCollector{}
	c.init()
	return c
}

func (c *netwatchCollector) init() {
	c.props = []string{"host", "comment", "status"}
	labelNames := []string{"name", "address", "host", "comment"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = helper.DescriptionForPropertyName("netwatch", p, labelNames)
	}
}

func (c *netwatchCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *netwatchCollector) Collect(ctx *collector.Context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *netwatchCollector) fetch(ctx *collector.Context) ([]*proto.Sentence, error) {
	reply, err := ctx.Client.Run("/tool/netwatch/print", "?disabled=false", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching netwatch metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *netwatchCollector) collectForStat(re *proto.Sentence, ctx *collector.Context) {
	host := re.Map["host"]
	comment := re.Map["comment"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(p, host, comment, re, ctx)
	}
}

func (c *netwatchCollector) collectMetricForProperty(property, host, comment string, re *proto.Sentence, ctx *collector.Context) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var numericValue float64
		switch value {
		case "up":
			numericValue = 1
		case "unknown":
			numericValue = 0
		case "down":
			numericValue = -1
		default:
			log.WithFields(log.Fields{
				"device":   ctx.Device.Name,
				"host":     host,
				"property": property,
				"value":    value,
				"error":    fmt.Errorf("unexpected netwatch status value"),
			}).Error("error parsing netwatch metric value")
		}
		ctx.Ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, numericValue, ctx.Device.Name, ctx.Device.Address, host, comment)
	}
}
