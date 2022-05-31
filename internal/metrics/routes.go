package metrics

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/helper"
)

func init() {
	Registry.Add("routes", newRoutesCollector)
}

type routesCollector struct {
	protocols         []string
	countDesc         *prometheus.Desc
	countProtocolDesc *prometheus.Desc
}

func newRoutesCollector() collector.Collector {
	c := &routesCollector{}
	c.init()
	return c
}

func (c *routesCollector) init() {
	const prefix = "routes"
	labelNames := []string{"name", "address", "ip_version"}
	c.countDesc = helper.Description(prefix, "total_count", "number of routes in RIB", labelNames)
	c.countProtocolDesc = helper.Description(prefix, "protocol_count", "number of routes per protocol in RIB", append(labelNames, "protocol"))

	c.protocols = []string{"bgp", "static", "ospf", "dynamic", "connect"}
}

func (c *routesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.countDesc
	ch <- c.countProtocolDesc
}

func (c *routesCollector) Collect(ctx *collector.Context) error {
	err := c.colllectForIPVersion("4", "ip", ctx)
	if err != nil {
		return err
	}

	return c.colllectForIPVersion("6", "ip", ctx)
}

func (c *routesCollector) colllectForIPVersion(ipVersion, topic string, ctx *collector.Context) error {
	err := c.colllectCount(ipVersion, topic, ctx)
	if err != nil {
		return err
	}

	for _, p := range c.protocols {
		err := c.colllectCountProtcol(ipVersion, topic, p, ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *routesCollector) colllectCount(ipVersion, topic string, ctx *collector.Context) error {
	reply, err := ctx.Client.Run(fmt.Sprintf("/%s/route/print", topic), "?disabled=false", "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"device":     ctx.Device.Name,
			"topic":      topic,
			"error":      err,
		}).Error("error fetching routes metrics")
		return err
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"device":     ctx.Device.Name,
			"error":      err,
		}).Error("error parsing routes metrics")
		return err
	}

	ctx.Ch <- prometheus.MustNewConstMetric(c.countDesc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address, ipVersion)
	return nil
}

func (c *routesCollector) colllectCountProtcol(ipVersion, topic, protocol string, ctx *collector.Context) error {
	reply, err := ctx.Client.Run(fmt.Sprintf("/%s/route/print", topic), "?disabled=false", fmt.Sprintf("?%s", protocol), "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"protocol":   protocol,
			"device":     ctx.Device.Name,
			"error":      err,
		}).Error("error fetching routes metrics")
		return err
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"ip_version": ipVersion,
			"protocol":   protocol,
			"device":     ctx.Device.Name,
			"error":      err,
		}).Error("error parsing routes metrics")
		return err
	}

	ctx.Ch <- prometheus.MustNewConstMetric(c.countProtocolDesc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address, ipVersion, protocol)
	return nil
}
