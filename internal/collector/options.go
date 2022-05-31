package collector

import (
	"time"
)

// Option applies options to collector
type Option func(*collector)

// WithMetrics add more feature to collector
func WithMetrics(cs ...Collector) Option {
	return func(c *collector) {
		for _, m := range cs {
			c.collectors = append(c.collectors, m)
		}
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
