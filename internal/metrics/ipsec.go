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
	Registry.Add("ipsec", newIpsecCollector)
}

type ipsecCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newIpsecCollector() collector.Collector {
	c := &ipsecCollector{}
	c.init()
	return c
}

func (c *ipsecCollector) init() {
	c.props = []string{"src-address", "dst-address", "ph2-state", "invalid", "active", "comment"}

	labelNames := []string{"devicename", "srcdst", "comment"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[1:] {
		c.descriptions[p] = helper.DescriptionForPropertyName("ipsec", p, labelNames)
	}
}

func (c *ipsecCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *ipsecCollector) Collect(ctx *collector.Context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *ipsecCollector) fetch(ctx *collector.Context) ([]*proto.Sentence, error) {
	reply, err := ctx.Client.Run("/ip/ipsec/policy/print", "?disabled=false", "?dynamic=false", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching interface metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *ipsecCollector) collectForStat(re *proto.Sentence, ctx *collector.Context) {
	srcdst := re.Map["src-address"] + "-" + re.Map["dst-address"]
	comment := re.Map["comment"]

	for _, p := range c.props[2:] {
		c.collectMetricForProperty(p, srcdst, comment, re, ctx)
	}
}

func (c *ipsecCollector) collectMetricForProperty(property, srcdst, comment string, re *proto.Sentence, ctx *collector.Context) {
	desc := c.descriptions[property]
	if value := re.Map[property]; value != "" {
		var v float64
		var err error
		v, err = strconv.ParseFloat(value, 64)

		switch property {
		case "ph2-state":
			if value == "established" {
				v, err = 1, nil
			} else {
				v, err = 0, nil
			}
		case "active", "invalid":
			if value == "true" {
				v, err = 1, nil
			} else {
				v, err = 0, nil
			}
		case "comment":
			return
		}

		if err != nil {
			log.WithFields(log.Fields{
				"device":   ctx.Device.Name,
				"srcdst":   srcdst,
				"property": property,
				"value":    value,
				"error":    err,
			}).Error("error parsing ipsec metric value")
			return
		}
		ctx.Ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, ctx.Device.Name, srcdst, comment)
	}
}
