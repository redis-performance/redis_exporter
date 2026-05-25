# Contributing

We treat this repo as "Open Source" within Redis: anyone who clears the bar below is welcome to contribute.

## Local setup

```bash
git clone git@github.com:redis-performance/redis_exporter.git
cd redis_exporter
go build .
./redis_exporter --version
```

Go 1.20 or later is required. Dependencies are managed with Go modules; running `go build .` will fetch them automatically.

## Branch naming

```
<type>/<short-description>
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

Example: `feat/add-pipeline-mode`

## Coding standards

- Keep changes focused; one logical change per PR.
- Follow the conventions already present in the codebase (formatting, naming, error handling).
- No dead code, no commented-out blocks.

## Submitting changes

1. Fork or create a branch from `master`.
2. Make your changes with clear, atomic commits.
3. Open a pull request against `master` with a descriptive title and summary.
4. Address review comments promptly; force-push to the same branch to update.

## Testing

- All new behaviour must be covered by tests.
- Existing tests must pass: run the test suite locally before opening a PR.
- Coverage should not decrease.

The test suite requires running Redis instances (multiple versions). The easiest way to spin them up is via Docker Compose:

```bash
# Start all Redis containers used by the tests
docker-compose -f contrib/docker-compose-for-tests.yml up -d

# Run the full test suite (inside the container that has all URIs pre-set)
make docker-test

# Or, if the Redis instances are already reachable, run tests directly:
go test -v -covermode=atomic -cover -race -coverprofile=coverage.txt -p 1 ./...
```

Static checks and formatting:

```bash
make checks   # runs go vet and verifies gofmt compliance
make lint     # runs golangci-lint (requires golangci-lint installed)
```

## Review process

- At least one maintainer approval is required before merge.
- CI must be green.
- Maintainers may request changes or close PRs that don't meet the bar — this is normal and not personal.
