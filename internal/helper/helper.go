package helper

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const namespace = "mikrotik"

var durationRegex = regexp.MustCompile(`(?:(\d*)w)?(?:(\d*)d)?(?:(\d*)h)?(?:(\d*)m)?(?:(\d*)s)?`)
var durationParts = [5]time.Duration{time.Hour * 168, time.Hour * 24, time.Hour, time.Minute, time.Second}

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

func SplitStringToFloats(metric string) (float64, float64, error) {
	strs := strings.Split(metric, ",")
	if len(strs) == 0 {
		return 0, 0, nil
	}
	m1, err := strconv.ParseFloat(strs[0], 64)
	if err != nil {
		return math.NaN(), math.NaN(), err
	}
	m2, err := strconv.ParseFloat(strs[1], 64)
	if err != nil {
		return math.NaN(), math.NaN(), err
	}
	return m1, m2, nil
}

func ParseDuration(duration string) (float64, error) {
	var u time.Duration

	reMatch := durationRegex.FindAllStringSubmatch(duration, -1)

	// should get one and only one match back on the regex
	if len(reMatch) != 1 {
		return 0, fmt.Errorf("invalid duration value sent to regex")
	} else {
		for i, match := range reMatch[0] {
			if match != "" && i != 0 {
				v, err := strconv.Atoi(match)
				if err != nil {
					log.WithFields(log.Fields{
						"duration": duration,
						"value":    match,
						"error":    err,
					}).Error("error parsing duration field value")
					return float64(0), err
				}
				u += time.Duration(v) * durationParts[i-1]
			}
		}
	}
	return u.Seconds(), nil
}
