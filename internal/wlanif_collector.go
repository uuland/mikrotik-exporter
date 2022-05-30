package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/helper"
)

type wlanIFCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newWlanIFCollector() collector.Collector {
	c := &wlanIFCollector{}
	c.init()
	return c
}

func (c *wlanIFCollector) init() {
	c.props = []string{"channel", "registered-clients", "noise-floor", "overall-tx-ccq"}
	labelNames := []string{"name", "address", "interface", "channel"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = helper.DescriptionForPropertyName("wlan_interface", p, labelNames)
	}
}

func (c *wlanIFCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *wlanIFCollector) Collect(ctx *collector.Context) error {
	names, err := c.fetchInterfaceNames(ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.collectForInterface(n, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *wlanIFCollector) fetchInterfaceNames(ctx *collector.Context) ([]string, error) {
	reply, err := ctx.Client.Run("/interface/wireless/print", "?disabled=false", "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching wireless interface names")
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *wlanIFCollector) collectForInterface(iface string, ctx *collector.Context) error {
	reply, err := ctx.Client.Run("/interface/wireless/monitor", fmt.Sprintf("=numbers=%s", iface), "=once=", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"interface": iface,
			"device":    ctx.Device.Name,
			"error":     err,
		}).Error("error fetching interface statistics")
		return err
	}

	for _, p := range c.props[1:] {
		// there's always going to be only one sentence in reply, as we
		// have to explicitly specify the interface
		c.collectMetricForProperty(p, iface, reply.Re[0], ctx)
	}

	return nil
}

func (c *wlanIFCollector) collectMetricForProperty(property, iface string, re *proto.Sentence, ctx *collector.Context) {
	desc := c.descriptions[property]
	channel := re.Map["channel"]
	if re.Map[property] == "" {
		return
	}
	v, err := strconv.ParseFloat(re.Map[property], 64)
	if err != nil {
		log.WithFields(log.Fields{
			"property":  property,
			"interface": iface,
			"device":    ctx.Device.Name,
			"error":     err,
		}).Error("error parsing interface metric value")
		return
	}

	ctx.Ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address, iface, channel)
}
