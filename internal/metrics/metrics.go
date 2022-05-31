package metrics

import (
	"errors"
	"fmt"

	"mikrotik-exporter/internal/collector"
)

var Registry = &registry{
	features: make(map[string]initialize),
}

type initialize func() collector.Collector

type registry struct {
	features map[string]initialize
}

func (r *registry) Add(name string, init initialize) {
	if _, exists := r.features[name]; exists {
		panic(fmt.Sprintf("already registered of %s", name))
	}

	r.features[name] = init
}

func (r *registry) Load(feats ...string) ([]collector.Collector, error) {
	var cs []collector.Collector

	for _, feat := range feats {
		if init, exists := r.features[feat]; exists {
			cs = append(cs, init())
		} else {
			return nil, errors.New(fmt.Sprintf("no collector for %s", feat))
		}
	}

	return cs, nil
}
