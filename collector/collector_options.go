package collector

import (
	"time"
)

// WithBGP enables BGP routing metrics
func WithBGP() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newBGPCollector())
	}
}

// WithRoutes enables routing table metrics
func WithRoutes() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newRoutesCollector())
	}
}

// WithDHCP enables DHCP serrver metrics
func WithDHCP() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newDHCPCollector())
	}
}

// WithDHCPL enables DHCP server leases
func WithDHCPL() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newDHCPLCollector())
	}
}

// WithDHCPv6 enables DHCPv6 serrver metrics
func WithDHCPv6() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newDHCPv6Collector())
	}
}

// WithFirmware grab installed firmware and version
func WithFirmware() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newFirmwareCollector())
	}
}

// WithHealth enables board Health metrics
func WithHealth() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newhealthCollector())
	}
}

// WithPOE enables PoE metrics
func WithPOE() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newPOECollector())
	}
}

// WithPools enables IP(v6) pool metrics
func WithPools() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newPoolCollector())
	}
}

// WithOptics enables optical diagnstocs
func WithOptics() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newOpticsCollector())
	}
}

// WithW60G enables w60g metrics
func WithW60G() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, neww60gInterfaceCollector())
	}
}

// WithWlanSTA enables wlan STA metrics
func WithWlanSTA() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newWlanSTACollector())
	}
}

// WithWlanIF enables wireless interface metrics
func WithWlanIF() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newWlanIFCollector())
	}
}

// WithMonitor enables ethernet monitor collector metrics
func Monitor() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newMonitorCollector())
	}
}

// WithTimeout sets timeout for connecting to router
func WithTimeout(d time.Duration) Option {
	return func(c *collector) {
		c.timeout = d
	}
}

// WithTLS enables TLS
func WithTLS(insecure bool) Option {
	return func(c *collector) {
		c.enableTLS = true
		c.insecureTLS = insecure
	}
}

// WithIpsec enables ipsec metrics
func WithIpsec() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newIpsecCollector())
	}
}

// WithConntrack enables firewall/NAT connection tracking metrics
func WithConntrack() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newConntrackCollector())
	}
}

// WithLte enables lte metrics
func WithLte() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newLteCollector())
	}
}

// WithNetwatch enables netwatch metrics
func WithNetwatch() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newNetwatchCollector())
	}
}
