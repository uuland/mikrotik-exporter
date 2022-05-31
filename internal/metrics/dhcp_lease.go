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
	Registry.Add("dhcp_lease", newDHCPLCollector)
}

type dhcpLeaseCollector struct {
	props        []string
	descriptions *prometheus.Desc
}

func (c *dhcpLeaseCollector) init() {
	c.props = []string{"active-mac-address", "server", "status", "expires-after", "active-address", "host-name"}

	labelNames := []string{"name", "address", "activemacaddress", "server", "status", "expiresafter", "activeaddress", "hostname"}
	c.descriptions = helper.Description("dhcp", "leases_metrics", "number of metrics", labelNames)

}

func newDHCPLCollector() collector.Collector {
	c := &dhcpLeaseCollector{}
	c.init()
	return c
}

func (c *dhcpLeaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.descriptions
}

func (c *dhcpLeaseCollector) Collect(ctx *collector.Context) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectMetric(ctx, re)
	}

	return nil
}

func (c *dhcpLeaseCollector) fetch(ctx *collector.Context) ([]*proto.Sentence, error) {
	reply, err := ctx.Client.Run("/ip/dhcp-server/lease/print", "?status=bound", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.Device.Name,
			"error":  err,
		}).Error("error fetching DHCP leases metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *dhcpLeaseCollector) collectMetric(ctx *collector.Context, re *proto.Sentence) {
	v := 1.0

	f, err := helper.ParseDuration(re.Map["expires-after"])
	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.Device.Name,
			"property": "expires-after",
			"value":    re.Map["expires-after"],
			"error":    err,
		}).Error("error parsing duration metric value")
		return
	}

	activemacaddress := re.Map["active-mac-address"]
	server := re.Map["server"]
	status := re.Map["status"]
	activeaddress := re.Map["active-address"]
	hostname := re.Map["host-name"]

	ctx.Ch <- prometheus.MustNewConstMetric(c.descriptions, prometheus.GaugeValue, v, ctx.Device.Name, ctx.Device.Address, activemacaddress, server, status, strconv.FormatFloat(f, 'f', 0, 64), activeaddress, hostname)
}
