package config

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestShouldParse(t *testing.T) {
	b := loadTestFile(t)
	c, err := Load(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("could not parse: %v", err)
	}

	if len(c.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %v", len(c.Devices))
	}

	assertDevice("test1", "192.168.1.1", "foo", "bar", c.Devices[0], t)
	assertDevice("test2", "192.168.2.1", "test", "123", c.Devices[1], t)
	assertFeature("BGP", getFeature(c, "bgp"), t)
	assertFeature("Conntrack", getFeature(c, "conntrack"), t)
	assertFeature("DHCP", getFeature(c, "dhcp"), t)
	assertFeature("DHCPv6", getFeature(c, "dhcpv6"), t)
	assertFeature("Pools", getFeature(c, "pools"), t)
	assertFeature("Routes", getFeature(c, "routes"), t)
	assertFeature("Optics", getFeature(c, "optics"), t)
	assertFeature("WlanSTA", getFeature(c, "wlansta"), t)
	assertFeature("WlanIF", getFeature(c, "wlanif"), t)
	assertFeature("Ipsec", getFeature(c, "ipsec"), t)
	assertFeature("Lte", getFeature(c, "lte"), t)
	assertFeature("Netwatch", getFeature(c, "netwatch"), t)
}

func loadTestFile(t *testing.T) []byte {
	b, err := ioutil.ReadFile("config.test.yml")
	if err != nil {
		t.Fatalf("could not load config: %v", err)
	}

	return b
}

func getFeature(c *Config, name string) bool {
	v, e := c.Features[name]
	return e && v
}

func assertDevice(name, address, user, password string, c *Device, t *testing.T) {
	if c.Name != name {
		t.Fatalf("expected name %s, got %s", name, c.Name)
	}

	if c.Address != address {
		t.Fatalf("expected address %s, got %s", address, c.Address)
	}

	if c.User != user {
		t.Fatalf("expected user %s, got %s", user, c.User)
	}

	if c.Password != password {
		t.Fatalf("expected password %s, got %s", password, c.Password)
	}
}

func assertFeature(name string, v bool, t *testing.T) {
	if !v {
		t.Fatalf("exprected feature %s to be enabled", name)
	}
}
