package helper

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "mikrotik"

func metricStringCleanup(in string) string {
	return strings.Replace(in, "-", "_", -1)
}

func DescriptionForPropertyName(prefix, property string, labelNames []string) *prometheus.Desc {
	return DescriptionForPropertyNameHelpText(prefix, property, labelNames, property)
}

func DescriptionForPropertyNameHelpText(prefix, property string, labelNames []string, helpText string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, prefix, metricStringCleanup(property)),
		helpText,
		labelNames,
		nil,
	)
}

func Description(prefix, name, helpText string, labelNames []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, prefix, name),
		helpText,
		labelNames,
		nil,
	)
}
