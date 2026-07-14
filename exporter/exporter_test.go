package exporter

/*
  to run the tests with redis running on anything but localhost:6379 use
  $ go test   --redis.addr=<host>:<port>

  for html coverage report run
  $ go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

import (
	"fmt"
	"math/rand"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
)

const (
	TestValue   = 1234.56
	TimeToSleep = 200
)

var (
	keys            []string
	keysExpiring    []string
	listKeys        []string
	singleStringKey string
	ts              = int32(time.Now().Unix())

	dbNumStr        = "11"
	altDBNumStr     = "12"
	invalidDBNumStr = "16"
	dbNumStrFull    = fmt.Sprintf("db%s", dbNumStr)
)

const (
	TestSetName    = "test-set"
	TestStreamName = "test-stream"
)

func getTestExporter() *Exporter {
	return getTestExporterWithOptions(Options{Namespace: "test", Registry: prometheus.NewRegistry()})
}

func getTestExporterWithOptions(opt Options) *Exporter {
	addr := os.Getenv("TEST_REDIS_URI")
	if addr == "" {
		panic("missing env var TEST_REDIS_URI")
	}
	e, _ := NewRedisExporter(addr, opt)
	return e
}

func getTestExporterWithAddr(addr string) *Exporter {
	e, _ := NewRedisExporter(addr, Options{Namespace: "test", Registry: prometheus.NewRegistry()})
	return e
}

// fqNameFromDesc pulls the fully-qualified metric name out of a prometheus.Desc
// string (which looks like `Desc{fqName: "test_foo_bytes", help: ...}`) without
// needing an extra regexp import.
func fqNameFromDesc(desc string) string {
	const marker = `fqName: "`
	i := strings.Index(desc, marker)
	if i < 0 {
		return ""
	}
	rest := desc[i+len(marker):]
	if j := strings.Index(rest, `"`); j >= 0 {
		return rest[:j]
	}
	return ""
}

// collectInfoMetrics feeds a synthetic INFO payload through extractInfoMetrics
// (no live Redis needed) and returns, per emitted metric name, its value and
// whether it was emitted as a counter.
func collectInfoMetrics(e *Exporter, info string) (values map[string]float64, isCounter map[string]bool) {
	chM := make(chan prometheus.Metric, 1000)
	go func() {
		e.extractInfoMetrics(chM, info, 0)
		close(chM)
	}()

	values = map[string]float64{}
	isCounter = map[string]bool{}
	for m := range chM {
		name := fqNameFromDesc(m.Desc().String())
		if name == "" {
			continue
		}
		dm := &dto.Metric{}
		if err := m.Write(dm); err != nil {
			continue
		}
		switch {
		case dm.GetGauge() != nil:
			values[name] = dm.GetGauge().GetValue()
			isCounter[name] = false
		case dm.GetCounter() != nil:
			values[name] = dm.GetCounter().GetValue()
			isCounter[name] = true
		}
	}
	return values, isCounter
}

// TestAdditionalShakaMetrics exercises the actual emission path (value + type +
// unit conversion + inf/nan handling) for a representative sample of the
// Shaka/Flex/RocksDB metrics, rather than asserting the maps against a copy of
// themselves.
func TestAdditionalShakaMetrics(t *testing.T) {
	e := getTestExporterWithAddr("redis://localhost:6379")

	// No "# Section" header => every line falls through to the generic
	// includeMetric/parseAndRegisterConstMetric path.
	info := strings.Join([]string{
		"mem_replication_backlog:1048576",
		"rocks_size_on_disk:5368709120",
		"rocks_ram_used_total:2097152",
		"rocks_comp_input_bytes:4200000000",
		"rocks_comp_elapsed_micros:12000000",
		"prefetch_nonblocking:42",
		"sync_repl_hold_latency_usec:500",
		"avg_pipeline_length_sum:900",
		"avg_pipeline_length_cnt:300",
		"repl_touch_keys:13375",
		"instantaneous_repl_touch_pct:-inf", // must be skipped
		"disk_fragmentation_ratio:inf",      // must be skipped
	}, "\n")

	values, isCounter := collectInfoMetrics(e, info)

	const eps = 1e-9
	type want struct {
		val      float64
		counter  bool
		wantEmit bool
	}
	cases := map[string]want{
		// bytes gauge, verbatim value
		"test_mem_replication_backlog_bytes": {val: 1048576, counter: false, wantEmit: true},
		"test_rocks_size_on_disk_bytes":      {val: 5368709120, counter: false, wantEmit: true},
		// _total gauge suffix dropped, byte unit added, stays a gauge
		"test_rocks_ram_used_bytes": {val: 2097152, counter: false, wantEmit: true},
		// cumulative RocksDB field now a counter with _total
		"test_rocks_comp_input_bytes_total": {val: 4200000000, counter: true, wantEmit: true},
		// counter + microseconds normalized to seconds
		"test_rocks_comp_elapsed_seconds_total": {val: 12.0, counter: true, wantEmit: true},
		// moved from gauge to counter
		"test_rof_prefetch_nonblocking_total": {val: 42, counter: true, wantEmit: true},
		// gauge, microseconds normalized to seconds
		"test_sync_repl_hold_latency_seconds": {val: 0.0005, counter: false, wantEmit: true},
		// reserved _sum/_count suffixes renamed, stay counters
		"test_rof_pipeline_length_total":  {val: 900, counter: true, wantEmit: true},
		"test_rof_pipeline_batches_total": {val: 300, counter: true, wantEmit: true},
		"test_repl_touch_keys_total":      {val: 13375, counter: true, wantEmit: true},
		// non-finite samples must not be emitted at all
		"test_instantaneous_repl_touch_pct": {wantEmit: false},
		"test_rof_disk_fragmentation_ratio": {wantEmit: false},
	}

	for name, w := range cases {
		got, emitted := values[name]
		if w.wantEmit != emitted {
			t.Errorf("%s: emitted=%v, want emitted=%v", name, emitted, w.wantEmit)
			continue
		}
		if !w.wantEmit {
			continue
		}
		if diff := got - w.val; diff > eps || diff < -eps {
			t.Errorf("%s: value=%v, want %v", name, got, w.val)
		}
		if isCounter[name] != w.counter {
			t.Errorf("%s: isCounter=%v, want %v", name, isCounter[name], w.counter)
		}
	}
}

// TestMetricMapsAreDisjoint is a genuine invariant (not a copy of the map data):
// a field key in BOTH maps gets the gauge's name but the counter's type, a
// name/type split that is hard to debug on a dashboard.
func TestMetricMapsAreDisjoint(t *testing.T) {
	e := getTestExporterWithAddr("redis://localhost:6379")
	for k := range e.metricMapGauges {
		if _, dup := e.metricMapCounters[k]; dup {
			t.Errorf("INFO field %q is present in both metricMapGauges and metricMapCounters", k)
		}
	}
}

func getTestExporterWithAddrAndOptions(addr string, opt Options) *Exporter {
	e, _ := NewRedisExporter(addr, opt)
	return e
}

func setupKeys(t *testing.T, c redis.Conn, dbNumStr string) error {
	if _, err := c.Do("SELECT", dbNumStr); err != nil {
		log.Printf("setupDBKeys() - couldn't setup redis, err: %s ", err)
		// not failing on this one - cluster doesn't allow for SELECT so we log and ignore the error
	}

	for _, key := range keys {
		if _, err := c.Do("SET", key, TestValue); err != nil {
			t.Errorf("couldn't setup redis, err: %s ", err)
			return err
		}
	}

	// setting to expire in 300 seconds, should be plenty for a test run
	for _, key := range keysExpiring {
		if _, err := c.Do("SETEX", key, "300", TestValue); err != nil {
			t.Errorf("couldn't setup redis, err: %s ", err)
			return err
		}
	}

	for _, key := range listKeys {
		for _, val := range keys {
			if _, err := c.Do("LPUSH", key, val); err != nil {
				t.Errorf("couldn't setup redis, err: %s ", err)
				return err
			}
		}
	}

	c.Do("SADD", TestSetName, "test-val-1")
	c.Do("SADD", TestSetName, "test-val-2")

	c.Do("SET", singleStringKey, "this-is-a-string")

	// Create test streams
	c.Do("XGROUP", "CREATE", TestStreamName, "test_group_1", "$", "MKSTREAM")
	c.Do("XGROUP", "CREATE", TestStreamName, "test_group_2", "$", "MKSTREAM")
	c.Do("XADD", TestStreamName, TestStreamTimestamps[0], "field_1", "str_1")
	c.Do("XADD", TestStreamName, TestStreamTimestamps[1], "field_2", "str_2")
	// Process messages to assign Consumers to their groups
	c.Do("XREADGROUP", "GROUP", "test_group_1", "test_consumer_1", "COUNT", "1", "STREAMS", TestStreamName, ">")
	c.Do("XREADGROUP", "GROUP", "test_group_1", "test_consumer_2", "COUNT", "1", "STREAMS", TestStreamName, ">")
	c.Do("XREADGROUP", "GROUP", "test_group_2", "test_consumer_1", "COUNT", "1", "STREAMS", TestStreamName, "0")

	return nil
}

func deleteKeys(c redis.Conn, dbNumStr string) {
	if _, err := c.Do("SELECT", dbNumStr); err != nil {
		log.Printf("deleteKeysFromDB() - couldn't setup redis, err: %s ", err)
		// not failing on this one - cluster doesn't allow for SELECT so we log and ignore the error
	}

	for _, key := range keys {
		c.Do("DEL", key)
	}

	for _, key := range keysExpiring {
		c.Do("DEL", key)
	}

	for _, key := range listKeys {
		c.Do("DEL", key)
	}

	c.Do("DEL", TestSetName)
	c.Do("DEL", TestStreamName)
	c.Do("DEL", singleStringKey)
}

func setupDBKeys(t *testing.T, uri string) error {
	c, err := redis.DialURL(uri)
	if err != nil {
		t.Errorf("couldn't setup redis for uri %s, err: %s ", uri, err)
		return err
	}
	defer c.Close()

	err = setupKeys(t, c, dbNumStr)
	if err != nil {
		t.Errorf("couldn't setup redis, err: %s ", err)
		return err
	}

	time.Sleep(time.Millisecond * 50)

	return nil
}

func setupDBKeysCluster(t *testing.T, uri string) error {
	e := Exporter{redisAddr: uri}
	c, err := e.connectToRedisCluster()
	if err != nil {
		return err
	}

	defer c.Close()

	err = setupKeys(t, c, "0")
	if err != nil {
		t.Errorf("couldn't setup redis, err: %s ", err)
		return err
	}

	time.Sleep(time.Millisecond * 50)

	return nil
}

func deleteKeysFromDB(t *testing.T, addr string) error {
	c, err := redis.DialURL(addr)
	if err != nil {
		t.Errorf("couldn't setup redis, err: %s ", err)
		return err
	}
	defer c.Close()

	deleteKeys(c, dbNumStr)

	return nil
}

func deleteKeysFromDBCluster(addr string) error {
	e := Exporter{redisAddr: addr}
	c, err := e.connectToRedisCluster()
	if err != nil {
		return err
	}

	defer c.Close()

	deleteKeys(c, "0")

	return nil
}

func TestIncludeSystemMemoryMetric(t *testing.T) {
	for _, inc := range []bool{false, true} {
		r := prometheus.NewRegistry()
		ts := httptest.NewServer(promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
		e, _ := NewRedisExporter(os.Getenv("TEST_REDIS_URI"), Options{Namespace: "test", InclSystemMetrics: inc})
		r.Register(e)

		body := downloadURL(t, ts.URL+"/metrics")
		if inc && !strings.Contains(body, "total_system_memory_bytes") {
			t.Errorf("want metrics to include total_system_memory_bytes, have:\n%s", body)
		} else if !inc && strings.Contains(body, "total_system_memory_bytes") {
			t.Errorf("did NOT want metrics to include total_system_memory_bytes, have:\n%s", body)
		}

		ts.Close()
	}
}

func TestIncludeConfigMetrics(t *testing.T) {
	for _, inc := range []bool{false, true} {
		r := prometheus.NewRegistry()
		ts := httptest.NewServer(promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
		e, _ := NewRedisExporter(os.Getenv("TEST_REDIS_URI"), Options{Namespace: "test", InclConfigMetrics: inc})
		r.Register(e)

		what := `test_config_key_value{key="appendonly",value="no"}`

		body := downloadURL(t, ts.URL+"/metrics")
		if inc && !strings.Contains(body, what) {
			t.Errorf("want metrics to include test_config_key_value, have:\n%s", body)
		} else if !inc && strings.Contains(body, what) {
			t.Errorf("did NOT want metrics to include test_config_key_value, have:\n%s", body)
		}

		ts.Close()
	}
}

func TestNonExistingHost(t *testing.T) {
	e, _ := NewRedisExporter("unix:///tmp/doesnt.exist", Options{Namespace: "test"})

	chM := make(chan prometheus.Metric)
	go func() {
		e.Collect(chM)
		close(chM)
	}()

	want := map[string]float64{"test_exporter_last_scrape_error": 1.0, "test_exporter_scrapes_total": 1.0}

	for m := range chM {
		descString := m.Desc().String()
		for k := range want {
			if strings.Contains(descString, k) {
				g := &dto.Metric{}
				m.Write(g)
				val := 0.0

				if g.GetGauge() != nil {
					val = *g.GetGauge().Value
				} else if g.GetCounter() != nil {
					val = *g.GetCounter().Value
				} else {
					continue
				}

				if val == want[k] {
					want[k] = -1.0
				}
			}
		}
	}
	for k, v := range want {
		if v > 0 {
			t.Errorf("didn't find %s", k)
		}
	}
}

func TestKeysReset(t *testing.T) {
	e, _ := NewRedisExporter(os.Getenv("TEST_REDIS_URI"), Options{Namespace: "test", CheckSingleKeys: dbNumStrFull + "=" + keys[0], Registry: prometheus.NewRegistry()})
	ts := httptest.NewServer(e)
	defer ts.Close()

	setupDBKeys(t, os.Getenv("TEST_REDIS_URI"))
	defer deleteKeysFromDB(t, os.Getenv("TEST_REDIS_URI"))

	chM := make(chan prometheus.Metric, 10000)
	go func() {
		e.Collect(chM)
		close(chM)
	}()

	body := downloadURL(t, ts.URL+"/metrics")
	if !strings.Contains(body, keys[0]) {
		t.Errorf("Did not found key %q\n%s", keys[0], body)
	}

	deleteKeysFromDB(t, os.Getenv("TEST_REDIS_URI"))

	body = downloadURL(t, ts.URL+"/metrics")
	if !strings.Contains(body, keys[0]) {
		t.Errorf("Key %q (from check-single-keys) should be available in metrics with default value 0\n%s", keys[0], body)
	}
}

func TestRedisMetricsOnly(t *testing.T) {
	for _, inc := range []bool{false, true} {
		r := prometheus.NewRegistry()
		ts := httptest.NewServer(promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
		_, err := NewRedisExporter(os.Getenv("TEST_REDIS_URI"), Options{Namespace: "test", Registry: r, RedisMetricsOnly: inc})
		if err != nil {
			t.Fatalf(`error when creating exporter with registry: %s`, err)
		}

		body := downloadURL(t, ts.URL+"/metrics")
		if inc && strings.Contains(body, "exporter_build_info") {
			t.Errorf("want metrics to include exporter_build_info, have:\n%s", body)
		} else if !inc && !strings.Contains(body, "exporter_build_info") {
			t.Errorf("did NOT want metrics to include exporter_build_info, have:\n%s", body)
		}

		ts.Close()
	}
}

func TestConnectionDurations(t *testing.T) {
	metric1 := "exporter_last_scrape_ping_time_seconds"
	metric2 := "exporter_last_scrape_connect_time_seconds"

	for _, inclPing := range []bool{false, true} {
		r := prometheus.NewRegistry()
		ts := httptest.NewServer(promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
		e, _ := NewRedisExporter(os.Getenv("TEST_REDIS_URI"), Options{Namespace: "test", PingOnConnect: inclPing})
		r.Register(e)

		body := downloadURL(t, ts.URL+"/metrics")
		if inclPing && !strings.Contains(body, metric1) {
			t.Fatalf("want metrics to include %s, have:\n%s", metric1, body)
		} else if !inclPing && strings.Contains(body, metric1) {
			t.Fatalf("did NOT want metrics to include %s, have:\n%s", metric1, body)
		}

		// always expect this one
		if !strings.Contains(body, metric2) {
			t.Fatalf("want metrics to include %s, have:\n%s", metric2, body)
		}
		ts.Close()
	}
}

func TestKeyDbMetrics(t *testing.T) {
	setupDBKeys(t, os.Getenv("TEST_KEYDB01_URI"))
	defer deleteKeysFromDB(t, os.Getenv("TEST_KEYDB01_URI"))

	for _, want := range []string{
		`test_db_keys_cached`,
		`test_storage_provider_read_hits`,
	} {
		r := prometheus.NewRegistry()
		ts := httptest.NewServer(promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
		e, _ := NewRedisExporter(os.Getenv("TEST_KEYDB01_URI"), Options{Namespace: "test"})
		r.Register(e)

		body := downloadURL(t, ts.URL+"/metrics")
		if !strings.Contains(body, want) {
			t.Errorf("want metrics to include %s, have:\n%s", want, body)
		}

		ts.Close()
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())

	ll := strings.ToLower(os.Getenv("LOG_LEVEL"))
	if pl, err := log.ParseLevel(ll); err == nil {
		log.Printf("Setting log level to: %s", ll)
		log.SetLevel(pl)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	for _, n := range []string{"john", "paul", "ringo", "george"} {
		keys = append(keys, fmt.Sprintf("key_%s_%d", n, ts))
	}

	singleStringKey = fmt.Sprintf("key_string_%d", ts)

	listKeys = append(listKeys, "beatles_list")

	for _, n := range []string{"A.J.", "Howie", "Nick", "Kevin", "Brian"} {
		keysExpiring = append(keysExpiring, fmt.Sprintf("key_exp_%s_%d", n, ts))
	}
}
