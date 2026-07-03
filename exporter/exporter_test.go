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

func TestAdditionalShakaMetricMappings(t *testing.T) {
	e := getTestExporterWithAddr("redis://localhost:6379")

	gaugeMappings := map[string]string{
		"max_process_mem":                       "max_process_mem_bytes",
		"current_active_defrag_time":            "current_active_defrag_time",
		"rdb_saves_consecutive_failures":        "rdb_saves_consecutive_failures",
		"aof_rewrites_consecutive_failures":     "aof_rewrites_consecutive_failures",
		"current_cow_peak":                      "current_cow_peak",
		"current_cow_size":                      "current_cow_size",
		"current_fork_perc":                     "current_fork_perc",
		"loading_loaded_perc":                   "loading_loaded_perc",
		"instantaneous_input_kbps":              "instantaneous_input_kbps",
		"instantaneous_output_kbps":             "instantaneous_output_kbps",
		"instantaneous_ops_per_sec":             "instantaneous_ops_per_sec",
		"expired_subkeys_active":                "expired_subkeys_active",
		"expired_keys_active":                   "expired_keys_active",
		"keys_trim_perc":                        "keys_trim_perc",
		"slave_read_only":                       "slave_read_only",
		"slave_read_repl_offset":                "slave_read_repl_offset",
		"slave_oom_pause":                       "slave_oom_pause",
		"master_sync_total_bytes":               "master_sync_total_bytes",
		"master_sync_read_bytes":                "master_sync_read_bytes",
		"master_sync_attempts":                  "master_sync_attempts",
		"master_link_down_since_seconds":        "master_link_down_since_seconds",
		"master_sync_perc":                      "master_sync_perc",
		"mem_replica_full_sync_buffer":          "mem_replica_full_sync_buffer_bytes",
		"used_ram_for_swapout":                  "rof_used_ram_for_swapout_bytes",
		"max_ram_by_data_ratio":                 "rof_max_ram_by_data_ratio",
		"disk_actual":                           "rof_disk_actual_bytes",
		"disk_inuse":                            "rof_disk_inuse_bytes",
		"disk_fragmentation_ratio":              "rof_disk_fragmentation_ratio",
		"disk_allocation":                       "rof_disk_allocation_bytes",
		"disk_ios":                              "rof_disk_ios",
		"disk_actual_key_names":                 "rof_disk_actual_key_names_bytes",
		"used_disk_key_names":                   "rof_used_disk_key_names_bytes",
		"ram_overhead":                          "rof_ram_overhead_bytes",
		"keys_ram_overhead":                     "rof_keys_ram_overhead_bytes",
		"ram_keys_size":                         "rof_ram_keys_size_bytes",
		"big_inst_avg_read_io_queue":            "rof_io_read_queue_length_avg",
		"big_inst_avg_write_io_queue":           "rof_io_write_queue_length_avg",
		"big_inst_avg_del_io_queue":             "rof_io_del_queue_length_avg",
		"big_inst_avg_io_blocked_clients":       "rof_io_blocked_clients_avg",
		"big_inst_avg_io_postponed_clients":     "rof_io_postponed_clients_avg",
		"io_keys_waiting":                       "rof_io_keys_waiting",
		"wait_busy_key":                         "rof_wait_busy_key",
		"prefetch_nonblocking":                  "rof_prefetch_nonblocking",
		"big_user_io_ratio_redis":               "rof_user_io_ratio_redis",
		"big_user_io_ratio_flash":               "rof_user_io_ratio_flash",
		"big_io_ratio_redis":                    "rof_io_ratio_redis",
		"big_io_ratio_flash":                    "rof_io_ratio_flash",
		"active_clients":                        "rof_active_clients",
		"ram_keys_needed":                       "rof_ram_keys_needed",
		"non_ram_keys_needed":                   "rof_non_ram_keys_needed",
		"big_ttl_scan_running":                  "rof_ttl_scan_running",
		"mem_ttl_histograms":                    "rof_mem_ttl_histograms",
		"unblessed_keys_awaiting_swapout":       "rof_unblessed_keys_awaiting_swapout",
		"sst_base_rdb_size":                     "sst_base_rdb_size_bytes",
		"sst_backup_size":                       "sst_backup_size_bytes",
		"sst_aof_incr_size":                     "sst_aof_incr_size_bytes",
		"sst_speedb_user_bytes_since_base":      "sst_speedb_user_bytes_since_base",
		"sync_repl_pending_clients":             "sync_repl_pending_clients",
		"sync_repl_pending_commands":            "sync_repl_pending_commands",
		"sync_repl_hold_count":                  "sync_repl_hold_count",
		"sync_repl_hold_depth_sum":              "sync_repl_hold_depth_sum",
		"sync_repl_hold_latency_usec":           "sync_repl_hold_latency_usec",
		"sync_dirty_keys_count":                 "sync_dirty_keys_count",
		"rocks_flush_started":                   "rocks_flush_started",
		"rocks_flush_completed":                 "rocks_flush_completed",
		"rocks_meta_flush_completed":            "rocks_meta_flush_completed",
		"rocks_meta_comp_completed":             "rocks_meta_comp_completed",
		"rocks_flush_writes_slowdown":           "rocks_flush_writes_slowdown",
		"rocks_flush_writes_stop":               "rocks_flush_writes_stop",
		"rocks_comp_started":                    "rocks_comp_started",
		"rocks_comp_completed":                  "rocks_comp_completed",
		"rocks_comp_input_bytes":                "rocks_comp_input_bytes",
		"rocks_comp_output_bytes":               "rocks_comp_output_bytes",
		"rocks_comp_elapsed_micros":             "rocks_comp_elapsed_micros",
		"rocks_comp_input_records":              "rocks_comp_input_records",
		"rocks_comp_output_records":             "rocks_comp_output_records",
		"rocks_comp_records_replaced":           "rocks_comp_records_replaced",
		"rocks_comp_records_deleted":            "rocks_comp_records_deleted",
		"rocks_L0_files":                        "rocks_L0_files",
		"rocks_meta_L0_files":                   "rocks_meta_L0_files",
		"rocks_keys_in_memtables":               "rocks_keys_in_memtables",
		"rocks_dels_in_memtables":               "rocks_dels_in_memtables",
		"rocks_num_immutable_mem_table":         "rocks_num_immutable_mem_table",
		"rocks_num_immutable_mem_table_flushed": "rocks_num_immutable_mem_table_flushed",
		"rocks_num_mem_table_flush_pending":     "rocks_num_mem_table_flush_pending",
		"rocks_num_compactions_pending":         "rocks_num_compactions_pending",
		"rocks_flush_num_entries":               "rocks_flush_num_entries",
		"rocks_flush_data_size":                 "rocks_flush_data_size",
		"rocks_size_of_actual_data":             "rocks_size_of_actual_data",
		"rocks_memtable_memory_budget":          "rocks_memtable_memory_budget",
		"rocks_ram_used_total":                  "rocks_ram_used_total",
		"rocks_keys_total":                      "rocks_keys_total",
		"rocks_size_on_disk":                    "rocks_size_on_disk",
		"rocks_total_files":                     "rocks_total_files",
		"rocks_ram_used_for_mem_tables":         "rocks_ram_used_for_mem_tables",
		"rocks_additional_ram_used_for_readers": "rocks_additional_ram_used_for_readers",
	}

	counterMappings := map[string]string{
		"keyspace_read_hits":                       "keyspace_read_hits_total",
		"keyspace_read_misses":                     "keyspace_read_misses_total",
		"keyspace_write_hits":                      "keyspace_write_hits_total",
		"keyspace_write_misses":                    "keyspace_write_misses_total",
		"keys_trimmed":                             "keys_trimmed_total",
		"keys_trim_scanned":                        "keys_trim_scanned_total",
		"keys_trim_total":                          "keys_trim_total",
		"total_forks":                              "forks_total",
		"total_active_defrag_time":                 "active_defrag_time_total",
		"rdb_saves":                                "rdb_saves_total",
		"aof_rewrites":                             "aof_rewrites_total",
		"repl_touch_bytes":                         "repl_touch_bytes_total",
		"repl_oom_buffer_rejections":               "repl_oom_buffer_rejections_total",
		"blocking_reads_missed":                    "rof_blocking_reads_missed_total",
		"prefetch_missing":                         "rof_prefetch_missing_total",
		"prefetch_expired":                         "rof_prefetch_expired_total",
		"prefetch_meta":                            "rof_prefetch_meta_total",
		"ramfetch_meta":                            "rof_ramfetch_meta_total",
		"big_io_writes_metaonly":                   "rof_io_writes_metaonly_total",
		"big_metadata_returned_to_ram":             "rof_metadata_returned_to_ram_total",
		"big_metadata_clean_returned_to_ram":       "rof_metadata_clean_returned_to_ram_total",
		"big_evex_scans_triggered":                 "rof_evex_scans_triggered_total",
		"big_evex_scans_completed":                 "rof_evex_scans_completed_total",
		"big_ttl_scans_triggered":                  "rof_ttl_scans_triggered_total",
		"big_ttl_histogram_switches":               "rof_ttl_histogram_switches_total",
		"big_disk_expired_subkeys_loaded":          "rof_disk_expired_subkeys_loaded_total",
		"big_ttl_histogram_expired":                "rof_ttl_histogram_expired_total",
		"big_ttl_histogram_oo_range":               "rof_ttl_histogram_oo_range_total",
		"big_disk_expired_keys":                    "rof_disk_expired_keys_total",
		"big_disk_evicted_keys":                    "rof_disk_evicted_keys_total",
		"io_blessed_keys":                          "rof_io_blessed_keys_total",
		"io_blessed_keys_serialized_size":          "rof_io_blessed_keys_serialized_size_bytes_total",
		"big_blessed_total":                        "rof_blessed_total",
		"big_unblessed_oom_total":                  "rof_unblessed_oom_total",
		"big_unblessed_keysize_total":              "rof_unblessed_keysize_total",
		"eventloop_cycles_with_clients_processing": "rof_eventloop_cycles_with_clients_processing_total",
		"total_client_processing_events":           "rof_client_processing_events_total",
		"big_head_of_line_unblocked":               "rof_head_of_line_unblocked_total",
		"big_next_in_line_unblocked":               "rof_next_in_line_unblocked_total",
		"big_head_of_line_blocked":                 "rof_head_of_line_blocked_total",
		"big_next_in_line_blocked":                 "rof_next_in_line_blocked_total",
		"avg_pipeline_length_sum":                  "rof_avg_pipeline_length_sum",
		"avg_pipeline_length_cnt":                  "rof_avg_pipeline_length_count",
	}

	for infoField, wantMetric := range gaugeMappings {
		if gotMetric := e.metricMapGauges[infoField]; gotMetric != wantMetric {
			t.Errorf("metricMapGauges[%q] = %q, want %q", infoField, gotMetric, wantMetric)
		}
	}

	for infoField, wantMetric := range counterMappings {
		if gotMetric := e.metricMapCounters[infoField]; gotMetric != wantMetric {
			t.Errorf("metricMapCounters[%q] = %q, want %q", infoField, gotMetric, wantMetric)
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
