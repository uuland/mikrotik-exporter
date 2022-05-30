package internal

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/helper"
)

type dhcpv6Collector struct {
	bindingCountDesc *prometheus.Desc
}

func newDHCPv6Collector() collector.Collector {
	c := &dhcpv6Collector{}
	c.init()
	return c
}

func (c *dhcpv6Collector) init() {
	const prefix = "dhcpv6"

	labelNames := []string{"name", "address", "server"}
	c.bindingCountDesc = helper.Description(prefix, "binding_count", "number of active bindings per DHCPv6 server", labelNames)
}

func (c *dhcpv6Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.bindingCountDesc
}

func (c *dhcpv6Collector) Collect(ctx *collector.Context) error {
	names, err := c.fetchDHCPServerNames(ctx)
	if err != nil {
		return err
	}

	for _, n := range names {
		err := c.colllectForDHCPServer(ctx, n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *dhcpv6Collector) fetchDHCPServerNames(ctx *collector.Context) ([]string, error) {
	reply, err := ctx.Client.Run("/ipv6/dhcp-server/print", "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching DHCPv6 server names")
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *dhcpv6Collector) colllectForDHCPServer(ctx *collector.Context, dhcpServer string) error {
	reply, err := ctx.Client.Run("/ipv6/dhcp-server/binding/print", fmt.Sprintf("?server=%s", dhcpServer), "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"dhcpv6_server": dhcpServer,
			"device":        ctx.Device.Name,
			"error":         err,
		}).Error("error fetching DHCPv6 binding counts")
		return err
	}

	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"dhcpv6_server": dhcpServer,
			"device":        ctx.Device.Name,
			"error":         err,
		}).Error("error parsing DHCPv6 binding counts")
		return err
	}

	ctx.Ch <- prometheus.MustNewConstMetric(c.bindingCountDesc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address, dhcpServer)
	return nil
}
