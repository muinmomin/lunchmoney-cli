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

## Install & Update

### Homebrew (Recommended)

Install:

```bash
brew tap muinmomin/lunchmoney-cli https://github.com/muinmomin/lunchmoney-cli
brew install muinmomin/lunchmoney-cli/lm
```

Update:

```bash
brew update
brew upgrade
```

### From Source

Install:

```bash
git clone https://github.com/muinmomin/lunchmoney-cli.git
cd lunchmoney-cli
go build -o ./bin/lm ./cmd/lm
```

Update:

```bash
cd lunchmoney-cli
git pull --ff-only
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
- `--include-pending` lists pending transactions only and requires `--unreviewed`
- all pages are fetched automatically
- in reviewed mode, categories marked `exclude_from_totals` are filtered out
- in unreviewed mode, `exclude_from_totals` filtering is not applied

### `lm category list`

List categories (archived categories are excluded).

```bash
lm category list [--json]
```

### `lm balances`

Show active account balances, grouped into cash/deposit accounts and credit cards.

```bash
lm balances [--json] [--include-inactive]
```

Behavior:

- active accounts are shown by default
- cash and deposit balances are totaled as cash
- credit account balances are totaled as card balances
- net cash after cards is calculated as cash minus card balances
- `--include-inactive` adds closed/inactive accounts to the account list

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

### `lm tx split`

Split one transaction into child transactions.

```bash
lm tx split <tx-id> --parts <n> [--dry-run] [--json]
lm tx split <tx-id> --amount <amount> --amount <amount>... [--dry-run] [--json]
```

Behavior:

- fetches the parent transaction first and validates before making the split API call
- refuses transactions that are already split or grouped
- `--parts` splits the parent amount into equal cent-based parts
- extra cents from `--parts` are distributed to earlier children so the sum stays exact
- repeated `--amount` values must contain at least two child amounts
- repeated `--amount` values must sum exactly to the parent amount
- child transactions inherit payee, date, category, and notes from the parent
- `--dry-run` validates and prints the child amounts without calling the split API

## Examples

```bash
lm tx list --start 2026-02-01
lm tx list --start 2026-02-01 --unreviewed
lm tx list --start 2026-02-01 --unreviewed --include-pending --json

lm category list
lm category list --json

lm balances
lm balances --json

lm tx update 2355632583 --category-id 1170290
lm tx update 2355632583 --note "testing"

lm tx mark-reviewed 2355632583 2355632591

lm tx split 2426461955 --parts 2 --dry-run
lm tx split 2426461955 --parts 3
lm tx split 2426461955 --amount 56.98 --amount 56.97
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
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
   - updates `Formula/lm.rb` with the exact version and checksums
   - commits the formula update to `main`

After release, users can run the update commands above.
