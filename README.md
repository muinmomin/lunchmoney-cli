# lunchmoney-cli

A simple, focused CLI for Lunch Money v2.

This tool is optimized for one workflow: list transactions, review uncategorized/unreviewed items, update category/notes, and mark reviewed.

## Highlights

- Lunch Money **v2 API only**
- Minimal command surface
- Pagination handled internally (fetches all pages)
- Opinionated defaults for fast review workflows
- JSON output support for agent/script usage

## Requirements

- Go 1.26+
- A Lunch Money API key (`LUNCHMONEY_API_KEY`)

## Install (From Source)

```bash
git clone https://github.com/muinmomin/lunchmoney-cli.git
cd lunchmoney-cli
go build -o ./bin/lm ./cmd/lm
```

## Configuration

Set your API key in your shell:

```bash
export LUNCHMONEY_API_KEY=your_api_key_here
```

## Commands

### `lm tx list`

List transactions in a date range.

```bash
lm tx list --start YYYY-MM-DD [--end YYYY-MM-DD] [--unreviewed] [--include-pending] [--json]
```

Behavior:

- `--start` is required
- `--end` defaults to local today when omitted
- default status is `reviewed`
- `--unreviewed` switches status to `unreviewed`
- pending transactions are excluded by default
- `--include-pending` includes pending transactions
- all pages are fetched automatically
- in reviewed mode, categories marked `exclude_from_totals` are filtered out
- in unreviewed mode, everything is included

### `lm category list`

List categories (archived categories are excluded).

```bash
lm category list [--json]
```

### `lm tx update`

Update a single transaction's category and/or note.

```bash
lm tx update <tx-id> [--category-id <id>] [--note <text>]
```

Rules:

- single transaction per command
- at least one of `--category-id` or `--note` is required
- empty notes are rejected

### `lm tx mark-reviewed`

Mark one or more transactions as reviewed.

```bash
lm tx mark-reviewed <tx-id> [<tx-id>...]
```

## Examples

```bash
lm tx list --start 2026-02-01
lm tx list --start 2026-02-01 --unreviewed
lm tx list --start 2026-02-01 --unreviewed --include-pending --json

lm category list
lm category list --json

lm tx update 2355632583 --category-id 1170290
lm tx update 2355632583 --note "testing"

lm tx mark-reviewed 2355632583 2355632591
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

## Homebrew

This repo is its own tap.

```bash
brew tap muinmomin/lunchmoney-cli https://github.com/muinmomin/lunchmoney-cli
brew install muinmomin/lunchmoney-cli/lunchmoney-cli
```

### Updating

```bash
brew update
brew upgrade muinmomin/lunchmoney-cli/lunchmoney-cli
```

## Release Flow (Homebrew + GitHub Releases)

Releases are tag-driven via GitHub Actions, with a helper script so you do not have to manually calculate versions or remember steps.

Use one of:

```bash
./scripts/release.sh patch
./scripts/release.sh minor
./scripts/release.sh major
./scripts/release.sh 0.2.0
```

What `scripts/release.sh` does:

- validates a clean working tree on `main`
- fast-forwards local `main` to `origin/main`
- runs `go build ./...`, `go vet ./...`, and `go test ./...`
- pushes `main`
- creates and pushes the new `vX.Y.Z` tag

The workflow at `.github/workflows/release-homebrew.yml` then automatically:

   - builds `lm` for macOS `arm64` and `amd64`
   - uploads `lm-darwin-arm64.tar.gz` and `lm-darwin-amd64.tar.gz` to the GitHub Release
   - computes SHA256 checksums
   - updates `Formula/lunchmoney-cli.rb` with the exact version and checksums
   - commits the formula update to `main`

After release, users can run the update commands above.
