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

type lteCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newLteCollector() collector.Collector {
	c := &lteCollector{}
	c.init()
	return c
}

func (c *lteCollector) init() {
	c.props = []string{"current-cellid", "primary-band", "ca-band", "rssi", "rsrp", "rsrq", "sinr"}
	labelNames := []string{"name", "address", "interface", "cellid", "primaryband", "caband"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = helper.DescriptionForPropertyName("lte_interface", p, labelNames)
	}
}

func (c *lteCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *lteCollector) Collect(ctx *collector.Context) error {
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

func (c *lteCollector) fetchInterfaceNames(ctx *collector.Context) ([]string, error) {
	reply, err := ctx.Client.Run("/interface/lte/print", "?disabled=false", "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching lte interface names")
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *lteCollector) collectForInterface(iface string, ctx *collector.Context) error {
	reply, err := ctx.Client.Run("/interface/lte/info", fmt.Sprintf("=number=%s", iface), "=once=", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"interface": iface,
			"device":    ctx.Device.Name,
			"error":     err,
		}).Error("error fetching interface statistics")
		return err
	}

	for _, p := range c.props[3:] {
		// there's always going to be only one sentence in reply, as we
		// have to explicitly specify the interface
		c.collectMetricForProperty(p, iface, reply.Re[0], ctx)
	}

	return nil
}

func (c *lteCollector) collectMetricForProperty(property, iface string, re *proto.Sentence, ctx *collector.Context) {
	desc := c.descriptions[property]
	current_cellid := re.Map["current-cellid"]
	// get only band and its width, drop earfcn and phy-cellid info
	primaryband := re.Map["primary-band"]
	if primaryband != "" {
		primaryband = strings.Fields(primaryband)[0]
	}
	caband := re.Map["ca-band"]
	if caband != "" {
		caband = strings.Fields(caband)[0]
	}

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

	ctx.Ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address, iface, current_cellid, primaryband, caband)
}
