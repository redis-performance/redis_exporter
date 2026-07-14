package exporter

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var metricNameRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func sanitizeMetricName(n string) string {
	return metricNameRE.ReplaceAllString(n, "_")
}

func newMetricDescr(namespace string, metricName string, docString string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "", metricName), docString, labels, nil)
}

func (e *Exporter) includeMetric(s string) bool {
	if strings.HasPrefix(s, "db") || strings.HasPrefix(s, "cmdstat_") || strings.HasPrefix(s, "cluster_") {
		return true
	}
	if _, ok := e.metricMapGauges[s]; ok {
		return true
	}

	_, ok := e.metricMapCounters[s]
	return ok
}

func (e *Exporter) parseAndRegisterConstMetric(ch chan<- prometheus.Metric, fieldKey, fieldValue string) {
	orgMetricName := sanitizeMetricName(fieldKey)
	metricName := orgMetricName
	if newName, ok := e.metricMapGauges[metricName]; ok {
		metricName = newName
	} else {
		if newName, ok := e.metricMapCounters[metricName]; ok {
			metricName = newName
		}
	}

	var err error
	var val float64

	switch fieldValue {

	case "ok", "true":
		val = 1

	case "err", "fail", "false":
		val = 0

	default:
		val, err = strconv.ParseFloat(fieldValue, 64)

	}
	if err != nil {
		// Skip emission on a parse error rather than publishing a phantom 0
		// that is indistinguishable from a legitimate zero datapoint.
		log.Debugf("couldn't parse %s, err: %s", fieldValue, err)
		return
	}

	t := prometheus.GaugeValue
	if e.metricMapCounters[orgMetricName] != "" {
		t = prometheus.CounterValue
	}

	// Normalize non-base time units to seconds, following the latest_fork_usec
	// precedent, so time series are comparable across the exporter.
	switch metricName {
	case "latest_fork_usec":
		metricName = "latest_fork_seconds"
		val = val / 1e6
	case "sync_repl_hold_latency_seconds", "rocks_comp_elapsed_seconds_total":
		val = val / 1e6
	}

	// Redis can emit inf/nan for ratio fields (e.g. a divide-by-zero
	// fragmentation ratio prints "inf"); strconv.ParseFloat accepts those with
	// no error. Publishing +Inf/NaN blanks Grafana panels via sum()/avg()/rate()
	// and destroys y-axis autoscaling, so skip the sample instead.
	if math.IsInf(val, 0) || math.IsNaN(val) {
		log.Debugf("skipping non-finite value for %s: %q", metricName, fieldValue)
		return
	}

	e.registerConstMetric(ch, metricName, val, t)
}

func (e *Exporter) registerConstMetricGauge(ch chan<- prometheus.Metric, metric string, val float64, labels ...string) {
	e.registerConstMetric(ch, metric, val, prometheus.GaugeValue, labels...)
}

func (e *Exporter) registerConstMetric(ch chan<- prometheus.Metric, metric string, val float64, valType prometheus.ValueType, labelValues ...string) {
	description := e.findOrCreateMetricDescription(metric, labelValues)
	m, err := prometheus.NewConstMetric(description, valType, val, labelValues...)
	if err != nil {
		log.Debugf("registerConstMetric( %s , %.2f) err: %s", metric, val, err)
		return
	}

	ch <- m
}

func (e *Exporter) registerConstSummary(ch chan<- prometheus.Metric, metric string, labelValues []string, count uint64, sum float64, latencyMap map[float64]float64, cmd string) {
	description := e.findOrCreateMetricDescription(metric, labelValues)

	// Create a constant summary from values we got from a 3rd party telemetry system.
	summary := prometheus.MustNewConstSummary(
		description,
		count, sum,
		latencyMap,
		cmd,
	)
	ch <- summary
}

func (e *Exporter) registerConstHistogram(ch chan<- prometheus.Metric, metric string, labelValues []string, count uint64, sum float64, buckets map[float64]uint64, cmd string) {
	description := e.findOrCreateMetricDescription(metric, labelValues)

	histogram := prometheus.MustNewConstHistogram(
		description,
		count, sum,
		buckets,
		cmd,
	)
	ch <- histogram
}

func (e *Exporter) findOrCreateMetricDescription(metricName string, labels []string) *prometheus.Desc {
	description, found := e.metricDescriptions[metricName]

	if !found {
		description = newMetricDescr(e.options.Namespace, metricName, metricName+" metric", labels)
		e.metricDescriptions[metricName] = description
	}

	return description
}
