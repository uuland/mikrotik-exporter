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

type wlanSTACollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newWlanSTACollector() collector.Collector {
	c := &wlanSTACollector{}
	c.init()
	return c
}

func (c *wlanSTACollector) init() {
	c.props = []string{"interface", "mac-address", "signal-to-noise", "signal-strength", "packets", "bytes", "frames"}
	labelNames := []string{"name", "address", "interface", "mac_address"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props[:len(c.props)-3] {
		c.descriptions[p] = helper.DescriptionForPropertyName("wlan_station", p, labelNames)
	}
	for _, p := range c.props[len(c.props)-3:] {
		c.descriptions["tx_"+p] = helper.DescriptionForPropertyName("wlan_station", "tx_"+p, labelNames)
		c.descriptions["rx_"+p] = helper.DescriptionForPropertyName("wlan_station", "rx_"+p, labelNames)
	}
}

func (c *wlanSTACollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *wlanSTACollector) Collect(ctx *collector.Context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *wlanSTACollector) fetch(ctx *collector.Context) ([]*proto.Sentence, error) {
	reply, err := ctx.Client.Run("/interface/wireless/registration-table/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching wlan station metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *wlanSTACollector) collectForStat(re *proto.Sentence, ctx *collector.Context) {
	iface := re.Map["interface"]
	mac := re.Map["mac-address"]

	for _, p := range c.props[2 : len(c.props)-3] {
		c.collectMetricForProperty(p, iface, mac, re, ctx)
	}
	for _, p := range c.props[len(c.props)-3:] {
		c.collectMetricForTXRXCounters(p, iface, mac, re, ctx)
	}
}

func (c *wlanSTACollector) collectMetricForProperty(property, iface, mac string, re *proto.Sentence, ctx *collector.Context) {
	if re.Map[property] == "" {
		return
	}
	p := re.Map[property]
	i := strings.Index(p, "@")
	if i > -1 {
		p = p[:i]
	}
	v, err := strconv.ParseFloat(p, 64)
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.Device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing wlan station metric value")
		return
	}

	desc := c.descriptions[property]
	ctx.Ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address, iface, mac)
}

func (c *wlanSTACollector) collectMetricForTXRXCounters(property, iface, mac string, re *proto.Sentence, ctx *collector.Context) {
	tx, rx, err := helper.SplitStringToFloats(re.Map[property])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.Device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing wlan station metric value")
		return
	}
	desc_tx := c.descriptions["tx_"+property]
	desc_rx := c.descriptions["rx_"+property]
	ctx.Ch <- prometheus.MustNewConstMetric(desc_tx, prometheus.CounterValue, tx, ctx.Device.Name, ctx.Device.Address, iface, mac)
	ctx.Ch <- prometheus.MustNewConstMetric(desc_rx, prometheus.CounterValue, rx, ctx.Device.Name, ctx.Device.Address, iface, mac)
}
