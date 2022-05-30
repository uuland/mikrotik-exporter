package internal

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/helper"
)

type conntrackCollector struct {
	props            []string
	totalEntriesDesc *prometheus.Desc
	maxEntriesDesc   *prometheus.Desc
}

func newConntrackCollector() collector.Collector {
	const prefix = "conntrack"

	labelNames := []string{"name", "address"}
	return &conntrackCollector{
		props:            []string{"total-entries", "max-entries"},
		totalEntriesDesc: helper.Description(prefix, "entries", "Number of tracked connections", labelNames),
		maxEntriesDesc:   helper.Description(prefix, "max_entries", "Conntrack table capacity", labelNames),
	}
}

func (c *conntrackCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalEntriesDesc
	ch <- c.maxEntriesDesc
}

func (c *conntrackCollector) Collect(ctx *collector.Context) error {
	reply, err := ctx.Client.Run("/ip/firewall/connection/tracking/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching conntrack table metrics")
		return err
	}

	for _, re := range reply.Re {
		c.collectMetricForProperty("total-entries", c.totalEntriesDesc, re, ctx)
		c.collectMetricForProperty("max-entries", c.maxEntriesDesc, re, ctx)
	}

	return nil
}

func (c *conntrackCollector) collectMetricForProperty(property string, desc *prometheus.Desc, re *proto.Sentence, ctx *collector.Context) {
	if re.Map[property] == "" {
		return
	}
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.Device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing conntrack metric value")
		return
	}

	ctx.Ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address)
}
