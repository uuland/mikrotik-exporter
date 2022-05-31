package collector

import (
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2"

	"mikrotik-exporter/internal/config"
	"mikrotik-exporter/internal/helper"
)

var (
	scrapeDurationDesc = helper.DescriptionForPropertyNameHelpText(
		"scrape", "collector_duration_seconds",
		[]string{"device"}, "mikrotik_exporter: duration of a collector scrape",
	)
	scrapeSuccessDesc = helper.DescriptionForPropertyNameHelpText(
		"scrape", "collector_success",
		[]string{"device"}, "mikrotik_exporter: whether a collector succeeded",
	)
)

type collector struct {
	devices     []*config.Device
	collectors  []Collector
	timeout     time.Duration
	enableTLS   bool
	insecureTLS bool
}

// Option applies options to collector
type Option func(*collector)

// NewCollector creates a collector instance
func NewCollector(cfg *config.Config, opts ...Option) (prometheus.Collector, error) {
	log.WithFields(log.Fields{
		"numDevices": len(cfg.Devices),
	}).Info("setting up collector for devices")

	c := &collector{
		devices:    cfg.Devices,
		timeout:    5 * time.Second,
		collectors: make([]Collector, 0),
	}

	for _, o := range opts {
		o(c)
	}

	if err := c.prepare(); err != nil {
		return nil, err
	}

	return c, nil
}

// Describe implements the prometheus.Collector interface.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc

	for _, co := range c.collectors {
		co.Describe(ch)
	}
}

// Collect implements the prometheus.Collector interface.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.realCollect(ch)
}

func (c *collector) prepare() error {
	for i, dev := range c.devices {
		if (config.SrvRecord{}) == dev.Srv {
			if _, err := c.connect(dev); err != nil {
				return err
			}
			continue
		}

		log.WithFields(log.Fields{
			"SRV": dev.Srv.Record,
		}).Info("SRV configuration detected")

		conf, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
		dnsServer := net.JoinHostPort(conf.Servers[0], "53")
		if (config.DnsServer{}) != dev.Srv.Dns {
			dnsServer = net.JoinHostPort(dev.Srv.Dns.Address, strconv.Itoa(dev.Srv.Dns.Port))
			log.WithFields(log.Fields{
				"DnsServer": dnsServer,
			}).Info("Custom DNS config detected")
		}
		dnsMsg := new(dns.Msg)
		dnsCli := new(dns.Client)

		dnsMsg.RecursionDesired = true
		dnsMsg.SetQuestion(dns.Fqdn(dev.Srv.Record), dns.TypeSRV)
		r, _, err := dnsCli.Exchange(dnsMsg, dnsServer)

		if err != nil {
			return err
		}

		for _, k := range r.Answer {
			if s, ok := k.(*dns.SRV); ok {
				d := &config.Device{}
				d.Name = strings.TrimRight(s.Target, ".")
				d.Address = strings.TrimRight(s.Target, ".")
				d.User = dev.User
				d.Password = dev.Password
				if err := c.getIdentity(d); err != nil {
					return err
				}
				c.devices[i] = d
			}
		}
	}

	return nil
}

func (c *collector) realCollect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}

	wg.Add(len(c.devices))

	for _, dev := range c.devices {
		go func(d *config.Device) {
			c.collectForDevice(d, ch)
			wg.Done()
		}(dev)
	}

	wg.Wait()
}

func (c *collector) collectForDevice(d *config.Device, ch chan<- prometheus.Metric) {
	begin := time.Now()

	err := c.connectAndCollect(d, ch)

	duration := time.Since(begin)
	var success float64
	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", d.Name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", d.Name, duration.Seconds())
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), d.Name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, d.Name)
}

func (c *collector) connectAndCollect(d *config.Device, ch chan<- prometheus.Metric) error {
	cl, err := c.connect(d)
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error dialing device")
		return err
	}

	defer func() {
		if err != nil && errors.Is(err, syscall.EPIPE) {
			d.Lock()
			defer d.Unlock()
			d.Cli = nil
		}
	}()

	for _, co := range c.collectors {
		ctx := &Context{ch, d, cl}
		err = co.Collect(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *collector) getIdentity(d *config.Device) error {
	cl, err := c.connect(d)
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error dialing device")
		return err
	}
	reply, err := cl.Run("/system/identity/print")
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error fetching ethernet interfaces")
		return err
	}
	for _, id := range reply.Re {
		d.Name = id.Map["name"]
	}
	return nil
}

func (c *collector) connect(d *config.Device) (*routeros.Client, error) {
	d.Lock()
	defer d.Unlock()

	if d.Cli != nil {
		return d.Cli, nil
	}

	var conn net.Conn
	var err error

	log.WithField("device", d.Name).Debug("trying to Dial")
	if !c.enableTLS {
		if (d.Port) == "" {
			d.Port = "8728"
		}
		conn, err = net.DialTimeout("tcp", d.Address+":"+d.Port, c.timeout)
		if err != nil {
			return nil, err
		}
	} else {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: c.insecureTLS,
		}
		if (d.Port) == "" {
			d.Port = "8729"
		}
		conn, err = tls.DialWithDialer(&net.Dialer{
			Timeout: c.timeout,
		},
			"tcp", d.Address+":"+d.Port, tlsCfg)
		if err != nil {
			return nil, err
		}
	}
	log.WithField("device", d.Name).Debug("done dialing")

	client, err := routeros.NewClient(conn)
	if err != nil {
		return nil, err
	}
	log.WithField("device", d.Name).Debug("got client")

	log.WithField("device", d.Name).Debug("trying to login")
	if err := client.Login(d.User, d.Password); err != nil {
		return nil, err
	}
	log.WithField("device", d.Name).Debug("done with login")

	d.Cli = client

	return client, nil
}
