# Agent guidelines

Instructions for AI coding agents (Claude Code, Copilot, Cursor, etc.) working in this repo.

## Project overview

`redis_exporter` is a Prometheus exporter for Redis metrics. It connects to one or more Redis instances and exposes their `INFO` statistics, keyspace data, and optional per-key metrics as a Prometheus `/metrics` endpoint (default port 9121). It supports Redis 2.x, 3.x, 4.x, 5.x, 6.x, and 7.x, as well as Redis Cluster, Redis Sentinel, KeyDB, and Tile38. The exporter is written in Go and uses the `gomodule/redigo` client library and the `prometheus/client_golang` instrumentation library.

## Local setup

```bash
git clone git@github.com:redis-performance/redis_exporter.git
cd redis_exporter
go build .
./redis_exporter --version
```

Go 1.20 or later is required. Dependencies are managed with Go modules and are fetched automatically by `go build`.

## Branch naming

Same as human contributors: `<type>/<short-description>` (e.g. `fix/off-by-one-in-pipeline`).

## Coding standards

- Match the style already in the file you are editing.
- Prefer clear, minimal changes over large refactors unless explicitly asked.
- Do not add comments that describe *what* the code does — only add comments when the *why* is non-obvious.
- Do not introduce new dependencies without checking with the maintainer.

## Running tests

The test suite requires live Redis instances. Spin them up with Docker Compose before running tests:

```bash
# Start all required Redis containers
docker-compose -f contrib/docker-compose-for-tests.yml up -d

# Run the full suite (inside the test container with all URIs configured)
make docker-test

# Or run tests directly when the Redis instances are already reachable:
go test -v -covermode=atomic -cover -race -coverprofile=coverage.txt -p 1 ./...
```

For static checks:

```bash
go vet ./...
make checks   # gofmt compliance check
```

Always run tests before declaring a task complete.

## How to submit changes

1. Create a branch: `git checkout -b <type>/<description>`.
2. Commit with a clear message focused on *why*, not *what*.
3. Open a pull request against `master`.
4. Do **not** push directly to `master`.

## What to avoid

- Do not reformat files unrelated to your change.
- Do not remove error handling or tests.
- Do not commit secrets, credentials, or large binary files.
- Do not amend published commits.
