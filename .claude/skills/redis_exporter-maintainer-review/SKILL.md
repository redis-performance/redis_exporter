---
name: redis_exporter-maintainer-review
description: Review a redis-performance/redis_exporter pull request, branch, or diff against this fork's own documented standards (AGENTS.md, CONTRIBUTING.md) and the concrete bug classes its real history shows — grounded in one genuinely detailed maintainer review this fork actually produced (PR #21), not a generic Go/Prometheus-exporter checklist. Use this whenever asked to review a redis_exporter PR "like a maintainer would," whether a redis_exporter PR would pass real review, or wants a redis_exporter-specific pre-merge check. Prefer this over a generic code-review skill for redis-performance/redis_exporter — it's grounded in this fork's actual precedent, honestly scoped to how much of it there is.
---

# redis_exporter maintainer-style review

## Honesty note — read this first

`redis-performance/redis_exporter` is a **fork** of the external, upstream `oliver006/redis_exporter` project. This
skill is mined ONLY from this fork's own PR history under the `redis-performance` org — never from upstream's
history — because upstream's contributors and reviewers are not this repo's own maintainers. As mined on
2026-08-27 (`gh pr list --repo redis-performance/redis_exporter --state all --limit 100`, 21 PRs total,
issues disabled on this repo entirely):

- **14 of 21 PRs are Dependabot version bumps** with zero human review activity (no review object, no comment) —
  pure noise for this skill's purposes.
- **One PR has a real, substantive, technical maintainer review**: PR #21 ("Expose additional Shaka and Flex
  INFO metrics"), reviewed by `fcostaoliveira` (a `MEMBER` of this org). That review is long, specific, and
  verified against a live Redis `INFO` capture rather than read off the diff alone — it is the single best
  source of ground truth this skill has, and most of this skill's taxonomy comes directly from it. Unusually,
  the reviewer didn't stop at commenting: after leaving the written review, they **pushed corrective commits
  directly onto the contributor's branch** rather than only requesting changes. This skill's own review step
  cannot replicate that (a bot has no push access and shouldn't get one) — treat it as evidence of what
  "acceptable to merge" looks like on this fork, not as a behavior to imitate.
- **Two PRs (#15, #16)** — both docs/CI chore PRs, one of which added this very `AGENTS.md`/`CONTRIBUTING.md` —
  got a same-pattern **empty-body `APPROVED` review from `paulorsousa`**: no comment, no inline notes, pure
  rubber-stamp. That is real signal about how *routine* PRs get treated here (silently approved, not narrated),
  but it says nothing about technical standards.
- **Two PRs (#17, #18)** were closed by `fcostaoliveira` with a short, specific one-line reason each — both
  because the contributor assumed something about this fork that isn't true (a Docker Hub image this fork
  doesn't publish; a Redis-version test matrix in `.drone.yml` that's intentional, not upstream drift). The
  lesson generalizes: **check what's actually true of this fork before proposing or endorsing a change that
  reads as "obviously correct" on a generic exporter.**
- **One PR (#20)** was closed as a duplicate opened from the wrong GitHub account, with an identical one-line
  comment from both the closer and the original author — pure process housekeeping, not a code-quality signal.
- **CI on this fork is thin**: `.github/workflows/` contains only `codeql-analysis.yml` and `depsreview.yaml`.
  There is **no `go test`, `go vet`, `gofmt`, or build workflow running in GitHub Actions on this fork**, despite
  `CONTRIBUTING.md` stating "CI must be green" and "existing tests must pass" as merge gates, and despite a
  `.drone.yml` and `make docker-test` / `make checks` / `make lint` existing in-repo. `fcostaoliveira` flagged
  this explicitly in the PR #21 review thread as a real gap, not this skill's own inference. **Treat a green
  GitHub Actions check on this fork as proof of nothing about test correctness** — CodeQL and a dependency
  review do not run the Go test suite.
- Issues are disabled on this repository, so `claude-issue-triage.yml` has nothing to trigger on today. It's
  included for parity and in case issues are ever enabled — see the PR description for this skill's rollout.

There is one real, detailed technical voice in this fork's history (`fcostaoliveira` on PR #21) and it is worth
taking seriously — but it is one data point, not a personality to embellish. Do not invent a second maintainer's
opinions, do not attribute nitpicks to `paulorsousa` (whose only recorded contribution is silent approval on
non-code PRs), and do not claim a review pattern is "the team's standard" when it's one person's one review.

## Process

1. **Get the material.** `gh pr view <n> --repo redis-performance/redis_exporter --json body,commits,files,author`
   and `gh pr diff <n> --repo redis-performance/redis_exporter`. Read the description first — this fork's better
   PRs (e.g. #21) include a `## Summary`/`## Notes` section naming what was and wasn't tested; if the author
   already flagged a limitation (e.g. "`go test ./...` panics without `TEST_REDIS_URI`"), acknowledge it rather
   than rediscovering it as new.

2. **Work the checklist** in `references/nitpick-taxonomy.md`. Every item there traces to a specific, real
   finding from PR #21's review (field-name verification, inf/NaN handling, counter/gauge classification,
   naming conventions, test quality) or to a specific closed PR (#17/#18's "check this fork's actual setup
   first" lesson). None of it is generic Prometheus-exporter advice bolted on afterward — if a concern doesn't
   trace to one of these, treat it as your own judgment call and say so plainly rather than implying precedent
   that doesn't exist.

3. **Because CI here provides almost no automated correctness signal, basic correctness is in scope, not
   deferred to "CI will catch it."** No workflow in `.github/workflows/` runs `go build`, `go test`, `go vet`,
   or `gofmt` on this fork. If the PR touches `exporter/*.go`, actually reason about whether the code compiles
   and behaves as claimed — nothing else will catch it before merge.

4. **If the PR adds or changes `INFO`-field-to-metric mappings** (`metricMapGauges` / `metricMapCounters` in
   `exporter/exporter.go`, or the allowlist logic in `exporter/metrics.go`'s `includeMetric` /
   `parseAndRegisterConstMetric`, or `exporter/info.go`'s `extractInfoMetrics`), apply the exact checklist PR #21
   was reviewed against — see `references/nitpick-taxonomy.md` items 1–5. These are this fork's best-evidenced,
   most concrete standards:
   - Left-hand `INFO` keys are matched **exactly and case-sensitively**; a typo silently emits nothing (no
     build error, no log line). Where practical, ask whether the mapped field names were checked against a
     real `INFO` capture from the target build (Speedb/Flex/Enterprise fields especially) rather than assumed
     from naming convention alone.
   - Does `parseAndRegisterConstMetric` (or equivalent new parsing code) guard against `inf`/`-inf`/`nan`
     values? `strconv.ParseFloat` accepts these with `err == nil`, so an unguarded ratio field (any divide-by-
     zero-shaped metric) can publish `+Inf`/`NaN` into Prometheus with no diagnostic — this has happened for
     real on this fork (`instantaneous_repl_touch_pct` was literally `"-inf"` in the capture that caught it).
   - Is each new field in the *correct* map — cumulative/monotonic values in `metricMapCounters` with a
     `_total` suffix, point-in-time values in `metricMapGauges` without one? A gauge-classified counter loses
     `rate()`/reset semantics silently; a `_total`-suffixed gauge produces garbage under `rate()` and trips
     promlint.
   - Naming: no `_sum`/`_count` suffix on a non-summary/histogram metric (promlint-reserved); `_bytes` suffix
     on byte-valued metrics; time units normalized to seconds (not `_usec`/`_micros`) matching the existing
     `latest_fork_usec` → `latest_fork_seconds` precedent in this file; metric names lowercase even when the
     source field has significant capitalization (e.g. RocksDB's `L0` → `l0` on the output side only).

5. **If the PR adds a test for new metric mappings**, check it isn't a tautology. This fork's own history
   produced exactly that anti-pattern once (see `references/nitpick-taxonomy.md` item 6): a test that
   re-declares the same map literal and asserts it equals itself, without ever calling `extractInfoMetrics` —
   it can't catch a field placed in the wrong map, or a value that never gets parsed, because it never exercises
   the parsing path at all. Prefer a table-driven test that feeds a synthetic `INFO` payload through
   `extractInfoMetrics`, drains the metrics channel, and asserts name + value + Prometheus type
   (`GetCounter()`/`GetGauge()`) for a representative sample.

6. **If the PR's premise assumes something about this fork's setup that might not be true** — a Docker image
   this fork publishes, a CI badge, a version matrix, an upstream-only file or workflow — verify it against
   what actually exists here before endorsing or flagging it. Two real PRs on this fork (#17, #18) were closed
   specifically because that check wasn't done first; see `references/voice-profiles.md`.

7. **Write the review terse and mostly as questions on a routine PR**, matching this fork's own default (silent
   or one-line approval on docs/CI/chore PRs). Reserve a longer, structured review for PRs that actually touch
   `exporter/*.go` metric-mapping or parsing logic — that's the one category with a real precedent for a
   detailed review, and forcing that same depth onto a one-line dependency bump or a README fix would be
   inventing engagement this fork's history doesn't show. If the PR is routine and well-described, the honest
   output may be no comment at all (`skip_comment: true`).

8. **Land on a plain-prose verdict.** No literal "Verdict:" label, no bolded summary line, no `@`-mention of any
   GitHub username — these rules apply regardless of what the one detailed mined review does (it does contain a
   real `@`-mention, made by a human maintainer to another human; this skill's automated output must not
   reproduce that pattern). See the workflow's own critical safety rules for why.

## What NOT to do

- Don't claim a rich, multi-person "maintainer voice" — there is one detailed technical review in this fork's
  entire history, from one person, on one PR. Say so if asked, rather than implying a broader review culture.
- Don't attribute technical nitpicks to `paulorsousa` — the only recorded signal from that account is two
  empty-body approvals on non-code (docs/CI) PRs.
- Don't skip basic correctness checks on new Go code on the theory that "CI would catch it" — no CI workflow on
  this fork currently runs `go build`, `go test`, `go vet`, or `gofmt`.
- Don't manufacture a duplicate-approval comment ("LGTM") on a routine PR — silence/terse approval is this
  fork's actual observed default for anything that isn't a metrics-mapping change.
- Don't literally `@`-mention any GitHub username in generated output, ever, for any reason, even though the
  one detailed human review this skill is grounded in does exactly that (human-to-human, not this workflow).
