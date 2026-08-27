# Nitpick taxonomy

Every item below traces to a specific, real finding in this fork's own history — cited inline. Nothing here is
generic Prometheus-exporter or Go advice added because it "sounds right." If you want to raise a concern this
list doesn't cover, that's fine — just say plainly it's your own judgment, not established precedent on this
fork.

## Items with direct precedent (from PR #21's review)

**1. Exact, case-sensitive `INFO` field-key matching.**
`includeMetric`/`parseAndRegisterConstMetric` (`exporter/metrics.go`) and the `metricMapGauges`/
`metricMapCounters` maps (`exporter/exporter.go`) key off exact left-hand `INFO` field names. A typo, a
renamed field on a newer build, or a field that doesn't actually exist on the target Redis/Enterprise/Flex
build produces **no metric and no error** — completely silent. PR #21 mapped a field
(`mem_replica_full_sync_buffer`) that turned out not to exist on the real build under review, while the field
that does exist (`mem_replication_backlog`) went unmapped, and this was only caught by checking a real `INFO`
capture. When a PR adds mappings for an Enterprise/Speedb/Flex-only `INFO` section, ask whether the field
names were checked against a real capture from that build — naming-convention plausibility isn't enough.

**2. `inf`/`NaN` guard on parsed values.**
`strconv.ParseFloat("inf", 64)` and `("nan", 64)` both return `err == nil`. Any new ratio/percentage metric
(anything shaped like a divide-by-zero) can silently publish `+Inf` or `NaN` into Prometheus with zero
diagnostic, which blanks downstream `sum()`/`avg()`/`rate()` panels. This is demonstrated, not hypothetical:
PR #21's review found `instantaneous_repl_touch_pct` literally rendering as `"-inf"` in a real capture. If new
parsing code doesn't check `math.IsInf`/`math.IsNaN` before emitting, flag it — especially for any new
ratio-shaped gauge.

**3. Counter vs. gauge classification.**
A field that is cumulative/monotonic (counts, cumulative bytes, cumulative elapsed time) belongs in
`metricMapCounters` with a `_total`-suffixed output name; a point-in-time value belongs in `metricMapGauges`
without one. PR #21 had ~13 cumulative RocksDB compaction/flush fields misclassified as gauges — losing
`rate()`/reset semantics with no warning. Check the direction implied by the field's own semantics (can it go
down? if not, it's probably a counter), not just where the author happened to put it.

**4. Naming-convention conformance (promlint-level, still worth flagging).**
- No `_total` suffix on a point-in-time/snapshot gauge — reserved for counters, and `rate()`/`increase()` on a
  `_total`-suffixed gauge produces garbage.
- No `_sum`/`_count` suffix on a metric that isn't part of a summary/histogram — these are promlint-reserved.
- `_bytes` suffix on byte-valued metrics, consistently with existing entries in the same map.
- Time units normalized to seconds — this file already has a real precedent for this exact conversion
  (`latest_fork_usec` → `latest_fork_seconds`, dividing by 1e6); a new `_usec`/`_micros`-suffixed field without
  the same conversion is inconsistent with the file's own established pattern, not a new rule being invented.
- Metric *output* names lowercase even when the source `INFO` field has significant capitalization (e.g.
  RocksDB's own `L0` naming) — the source key can keep its casing, only the exported metric name needs to
  normalize.

**5. Scale consistency between sibling metric families.**
If one family of related metrics is 0–1 (`*_ratio`) and a new field is semantically the same kind of value but
lands on a 0–100 scale (`*_perc`), that's a real trap for a dashboard built assuming one consistent scale
across the family — worth a comment even if the inconsistency is deliberate (self-documented by the differing
suffix), so a reviewer/dashboard author doesn't have to rediscover it by staring at unexpectedly-100x values.

**6. Test quality — avoid tautological metric-mapping tests.**
A test that re-declares the same map literal the production code uses and asserts they're equal (without ever
calling `extractInfoMetrics`) verifies nothing about emitted values, Prometheus types, or unit conversions —
and structurally cannot catch item #3 above (a field misclassified in both the production map and the test's
copy of it passes). This exact anti-pattern appeared in PR #21's initial test
(`TestAdditionalShakaMetricMappings`). Prefer a table-driven test that feeds a synthetic `INFO` payload through
`extractInfoMetrics`, drains the metrics channel, and asserts name + value + Prometheus type
(`GetCounter()`/`GetGauge()` non-nil as appropriate) — the pattern `TestNonExistingHost` and similar existing
tests in `exporter/exporter_test.go` already use.

**7. Docs for new, opaque metric families.**
New metrics get an auto-generated `"<name> metric"` HELP string by default, which is close to useless for
enterprise-only jargon (`sst_`, `rocks_`, `big_`, `rof_` prefixes) with no unit or semantics visible from the
name alone. PR #21's review asked for a short table in the PR/README noting family, field, exported name,
unit, and a one-line meaning, plus a note on which Redis build/edition actually emits it.

## Items with indirect precedent (from PR #17 / #18's closures)

**8. Verify this fork's actual setup before endorsing a change that assumes upstream conventions.**
Two real PRs were closed specifically because their premise didn't hold on this fork: a Redis-version-matrix
"fix" that assumed drift when the matrix is intentional compatibility testing (#17), and a CI badge referencing
a Docker image this fork doesn't publish (#18). If a PR touches `.drone.yml`, CI badges, published Docker
images, or anything else that differs between this fork and the upstream `oliver006/redis_exporter` project,
check what's actually configured here — `.github/workflows/`, `.drone.yml`, `README.md` — before treating the
change as obviously correct or obviously wrong.

## Written policy without observed enforcement — cite carefully

**9. `CONTRIBUTING.md`'s "CI must be green" / "existing tests must pass" gate.**
This is real, current, written policy in this fork's own `CONTRIBUTING.md` — but as of this mining,
`.github/workflows/` contains only `codeql-analysis.yml` and `depsreview.yaml`; no GitHub Actions workflow on
this fork runs `go build`, `go test`, `go vet`, or `gofmt`. A green check here does not mean the Go test suite
passed. If you cite the "tests must pass" policy, be explicit that GitHub Actions doesn't currently enforce it
automatically — the reviewer (human or this skill) is the actual gate today, not CI.

**10. `AGENTS.md`/`CONTRIBUTING.md`'s "at least one maintainer approval required" gate.**
Observed practice is consistent with this policy on the PRs that were reviewed at all (#15, #16, #21 all have
a `MEMBER` approval or substantive comment before merge) — but 14 of 21 PRs in this fork's history are
Dependabot bumps with no human review recorded either way, so don't over-claim universal enforcement.
