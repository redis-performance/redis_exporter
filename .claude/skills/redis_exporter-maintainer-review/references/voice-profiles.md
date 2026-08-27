# Voice profiles — what the mined history actually shows

Mined 2026-08-27 via `gh pr list --repo redis-performance/redis_exporter --state all --limit 100` (21 PRs
total) plus `gh api repos/redis-performance/redis_exporter/pulls/<n>/reviews` and
`.../issues/<n>/comments` on every non-Dependabot PR. Issues are disabled on this repository, so there is no
issue-side history to mine at all.

## fcostaoliveira (MEMBER) — the one substantive technical reviewer

The only account in this fork's history to leave a long, specific, technically-grounded review.

**PR #21 — "Expose additional Shaka and Flex INFO metrics"** (2026-07-14): a structured review that:
- Explicitly states the overall verdict up front ("sound and safe to land... strictly additive... no backward-
  compat breakage") before listing findings — leads with what's fine, not just what's wrong.
- Verifies claims against a **real `INFO` capture** rather than trusting field names by inspection: found that
  the PR mapped a field (`mem_replica_full_sync_buffer`) that doesn't exist on the real build, while the field
  that actually appears (`mem_replication_backlog`) went unmapped.
- Flagged a real, demonstrated `inf`/`NaN` propagation bug (`strconv.ParseFloat` accepts `"inf"`/`"nan"` with
  `err == nil`), backed by a literal value from the real capture (`instantaneous_repl_touch_pct: "-inf"`) —
  not a hypothetical.
- Organized findings into explicit tiers (fix-before-merge vs. naming/convention vs. test/docs), rather than
  a flat list — makes the triage priority explicit instead of implying everything is equally blocking.
- Called out a **tautological test** by name (`TestAdditionalShakaMetricMappings` re-asserts the same map
  against itself, never exercising `extractInfoMetrics`) and explained precisely why it can't catch the bug
  class the PR actually introduced.
- Did not stop at commenting: pushed a follow-up commit directly onto the PR branch fixing the Tier-1 items,
  and left two explicit `NOTE:` comments in code for judgment calls that needed the original author's input
  rather than resolving them unilaterally.
- Explicitly named a process gap unprompted: "there's no `go test`/build workflow on this fork — the green
  checks don't actually exercise the suite."

This is one review, not a personality — but it is real, and it is the strongest single piece of evidence
this skill has for "what does correct, careful review look like on this fork."

**PR #17 ("fix: upgrade Redis service container image")** and **#18 ("docs: add CI badge to README")** — both
closed same-day (2026-05-27) with a one-line, specific rejection reason each:
- #17: "this is a fork of oliver006/redis_exporter; redis version matrix in .drone.yml is intentional
  compatibility testing." (Not upstream drift to "fix.")
- #18: "no redis-performance-owned Docker image on this repo; badge not applicable." (The badge assumed a
  publishing setup this fork doesn't have.)

Both closures share a pattern: the contributor proposed a change that would be reasonable on a generic
exporter repo, but doesn't hold once you check what's actually true of *this* fork. That's the generalizable
lesson, not "Redis version matrices are always intentional" or "badges are always wrong."

## paulorsousa (MEMBER) — silent/empty approval on non-code PRs

Two recorded reviews, both `APPROVED` with an **empty body and zero inline comments**:
- PR #15 (added this repo's own `CONTRIBUTING.md`/`AGENTS.md`)
- PR #16 (bumped GitHub Actions to node24-compatible versions)

Both are docs/CI-only PRs, not exporter code. This is real signal that routine, self-evidently-safe PRs get a
quick, silent approval rather than a narrated one on this fork — but it is exactly zero evidence about what
this account would flag on a metrics-mapping change, because it has never reviewed one.

## What isn't here

- No maintainer has ever left an inline (line-level) review comment on this fork, only PR-level review bodies
  and issue-style comments.
- No PR has ever been formally `REQUEST_CHANGES`'d — PR #21's detailed critique was posted as a `COMMENTED`
  review, with the fix pushed directly rather than gated behind a re-review.
- No second technical reviewer exists to cross-check `fcostaoliveira`'s standards against — there's no
  disagreement, negotiation, or overruled nitpick anywhere in this history to draw on.
- Dependabot PRs (14 of 21 total) carry no human engagement at all — treat these the same way the fork's own
  history does: as pure automation noise, not something requiring commentary.
