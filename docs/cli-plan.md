# Lunch Money CLI Plan (v2 only)

## Scope
A minimal CLI focused on reviewing and maintaining transactions with a small, stable command surface.

## Commands

### `lm tx list`
List transactions for a date range.

Usage:
- `lm tx list --start YYYY-MM-DD [--end YYYY-MM-DD] [--unreviewed] [--include-pending] [--json]`

Behavior:
- `--start` is required.
- `--end` defaults to today's local date.
- Default status filter is `reviewed`.
- `--unreviewed` switches status filter to `unreviewed`.
- Pending transactions are excluded by default.
- `--include-pending` lists pending transactions only and requires `--unreviewed`.
- Pagination is internal and automatic until all pages are fetched (`has_more=false`).
- For default listing (`reviewed`): filter out transactions where category has `exclude_from_totals=true`.
- For review listing (`--unreviewed`): do not filter by `exclude_from_totals`.

Transaction output fields (MCP-like, plus review metadata):
- `id`
- `date`
- `description`
- `category`
- `amount` (normalized sign: outflow negative, inflow positive)
- `account`
- `institution`
- `group`
- `type` (`income`, `transfer`, `expense`)
- `notes`
- `tags`
- `status`
- `is_pending`

### `lm category list`
List categories.

Usage:
- `lm category list [--json]`

### `lm tx update`
Update a single transaction category and/or note.

Usage:
- `lm tx update <tx-id> [--category-id <id>] [--note <text>]`

Behavior:
- Exactly one transaction id is accepted.
- At least one of `--category-id` or `--note` is required.
- Empty note values are rejected (no note-clearing behavior).
- Only category and note are updated.

### `lm tx mark-reviewed`
Mark one or more transactions as reviewed.

Usage:
- `lm tx mark-reviewed <tx-id> [<tx-id>...]`

Behavior:
- Sends all ids in a single bulk update request.
- No special retry/fallback behavior; API response is surfaced.

## API Notes
- API version: Lunch Money v2 only (`https://api.lunchmoney.dev/v2`).
- Auth: `LUNCHMONEY_API_KEY` environment variable.

## Non-goals (for now)
- v1 support and compatibility modes.
- Broad account/tag/rule operations.
- Interactive prompts.
- Destructive commands (delete/clear note, etc.).
