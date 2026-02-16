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

Homebrew distribution is planned next.
