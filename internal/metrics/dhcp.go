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
	Registry.Add("dhcp", newDHCPCollector)
}

type dhcpCollector struct {
	leasesActiveCountDesc *prometheus.Desc
}

func (c *dhcpCollector) init() {
	const prefix = "dhcp"

	labelNames := []string{"name", "address", "server"}
	c.leasesActiveCountDesc = helper.Description(prefix, "leases_active_count", "number of active leases per DHCP server", labelNames)
}

func newDHCPCollector() collector.Collector {
	c := &dhcpCollector{}
	c.init()
	return c
}

func (c *dhcpCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.leasesActiveCountDesc
}

func (c *dhcpCollector) Collect(ctx *collector.Context) error {
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

func (c *dhcpCollector) fetchDHCPServerNames(ctx *collector.Context) ([]string, error) {
	reply, err := ctx.Client.Run("/ip/dhcp-server/print", "=.proplist=name")
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching DHCP server names")
		return nil, err
	}

	names := []string{}
	for _, re := range reply.Re {
		names = append(names, re.Map["name"])
	}

	return names, nil
}

func (c *dhcpCollector) colllectForDHCPServer(ctx *collector.Context, dhcpServer string) error {
	reply, err := ctx.Client.Run("/ip/dhcp-server/lease/print", fmt.Sprintf("?server=%s", dhcpServer), "=active=", "=count-only=")
	if err != nil {
		log.WithFields(log.Fields{
			"dhcp_server": dhcpServer,
			"device":      ctx.Device.Name,
			"error":       err,
		}).Error("error fetching DHCP lease counts")
		return err
	}
	if reply.Done.Map["ret"] == "" {
		return nil
	}
	v, err := strconv.ParseFloat(reply.Done.Map["ret"], 32)
	if err != nil {
		log.WithFields(log.Fields{
			"dhcp_server": dhcpServer,
			"device":      ctx.Device.Name,
			"error":       err,
		}).Error("error parsing DHCP lease counts")
		return err
	}

	ctx.Ch <- prometheus.MustNewConstMetric(c.leasesActiveCountDesc, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address, dhcpServer)
	return nil
}
