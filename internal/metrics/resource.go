package metrics

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/helper"
)

func init() {
	Registry.Add("resource", newResourceCollector)
}

type resourceCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newResourceCollector() collector.Collector {
	c := &resourceCollector{}
	c.init()
	return c
}

func (c *resourceCollector) init() {
	c.props = []string{"free-memory", "total-memory", "cpu-load", "free-hdd-space", "total-hdd-space", "uptime", "board-name", "version"}

	labelNames := []string{"name", "address", "boardname", "version"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = helper.DescriptionForPropertyName("system", p, labelNames)
	}
}

func (c *resourceCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *resourceCollector) Collect(ctx *collector.Context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *resourceCollector) fetch(ctx *collector.Context) ([]*proto.Sentence, error) {
	reply, err := ctx.Client.Run("/system/resource/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching system resource metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *resourceCollector) collectForStat(re *proto.Sentence, ctx *collector.Context) {
	for _, p := range c.props[:6] {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *resourceCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *collector.Context) {
	var v float64
	var err error
	//	const boardname = "BOARD"
	//	const version = "3.33.3"

	boardname := re.Map["board-name"]
	version := re.Map["version"]

	if property == "uptime" {
		v, err = helper.ParseDuration(re.Map[property])
	} else {
		if re.Map[property] == "" {
			return
		}
		v, err = strconv.ParseFloat(re.Map[property], 64)
	}

	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.Device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing system resource metric value")
		return
	}

	desc := c.descriptions[property]
	ctx.Ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, ctx.Device.Name, ctx.Device.Address, boardname, version)
}
