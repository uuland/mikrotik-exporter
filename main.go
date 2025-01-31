package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"

	"mikrotik-exporter/internal/collector"
	"mikrotik-exporter/internal/config"
	"mikrotik-exporter/internal/metrics"
)

// single device can be defined via CLI flags, multiple via config file.
var (
	listen      = flag.String("port", ":9436", "port number to listen on")
	address     = flag.String("address", "", "address of the device to monitor")
	configFile  = flag.String("config-file", "", "config file to load")
	device      = flag.String("device", "", "single device to monitor")
	devPort     = flag.String("deviceport", "8728", "port for single device")
	user        = flag.String("user", "", "user for authentication with single device")
	password    = flag.String("password", "", "password for authentication for single device")
	timeout     = flag.Duration("timeout", 5*time.Second, "timeout when connecting to devices")
	tls         = flag.Bool("tls", false, "use tls to connect to routers")
	insecure    = flag.Bool("insecure", false, "skips verification of server certificate when using TLS (not recommended)")
	features    = flag.String("features", "interface,resource", "enabled features")
	metricsPath = flag.String("path", "/metrics", "path to answer requests on")
	logFormat   = flag.String("log-format", "json", "logformat text or json (default json)")
	logLevel    = flag.String("log-level", "info", "log level")
	showVersion = flag.Bool("version", false, "show the version of binary")

	cfg *config.Config

	appVersion = "DEVELOPMENT"
	shortSha   = "0xDEADBEEF"
)

func init() {
	prometheus.MustRegister(version.NewCollector("mikrotik_exporter"))
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("\nVersion:   %s\nShort SHA: %s\n\n", appVersion, shortSha)
		os.Exit(0)
	}

	configureLog()

	c, err := loadConfig()
	if err != nil {
		log.Errorf("Could not load config: %v", err)
		os.Exit(3)
	}
	cfg = c

	startServer()
}

func configureLog() {
	ll, err := log.ParseLevel(*logLevel)
	if err != nil {
		panic(err)
	}

	log.SetLevel(ll)

	if *logFormat == "text" {
		log.SetFormatter(&log.TextFormatter{})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}
}

func loadConfig() (*config.Config, error) {
	if *configFile != "" {
		return loadConfigFromFile()
	}

	return loadConfigFromFlags()
}

func loadConfigFromFile() (*config.Config, error) {
	b, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	return config.Load(bytes.NewReader(b))
}

func loadConfigFromFlags() (*config.Config, error) {
	// Attempt to read credentials from env if not already defined
	if *user == "" {
		*user = os.Getenv("MIKROTIK_USER")
	}
	if *password == "" {
		*password = os.Getenv("MIKROTIK_PASSWORD")
	}
	if *device == "" || *address == "" || *user == "" || *password == "" {
		return nil, fmt.Errorf("missing required param for single device configuration")
	}

	return &config.Config{
		Devices: []*config.Device{
			{
				Name:     *device,
				Address:  *address,
				User:     *user,
				Password: *password,
				Port:     *devPort,
			},
		},
	}, nil
}

func startServer() {
	h, err := createMetricsHandler()
	if err != nil {
		log.Fatal(err)
	}
	http.Handle(*metricsPath, h)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Mikrotik Exporter</title></head>
			<body>
			<h1>Mikrotik Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Info("Listening on ", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

func createMetricsHandler() (http.Handler, error) {
	feats, err := metrics.Registry.Load(strings.Split(*features, ",")...)
	if err != nil {
		return nil, err
	}

	opts := []collector.Option{
		collector.WithTimeout(*timeout),
		collector.WithMetrics(feats...),
	}

	if *tls {
		opts = append(opts, collector.WithTLS(*insecure))
	}

	nc, err := collector.NewCollector(cfg, opts...)
	if err != nil {
		return nil, err
	}

	registry := prometheus.NewRegistry()
	if err := registry.Register(nc); err != nil {
		return nil, err
	}

	return promhttp.HandlerFor(registry,
		promhttp.HandlerOpts{
			ErrorLog:      log.New(),
			ErrorHandling: promhttp.ContinueOnError,
		}), nil
}
