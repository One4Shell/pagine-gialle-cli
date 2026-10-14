# paginegialle

A command-line client for **paginegialle.it** (the Italian Yellow Pages) focused
on **B2B contact research**. It searches listings and enriches company profiles
into stable, machine-readable JSON that is safe to pipe into scripts and AI
agents.

Output is engineered for machines: **one compact JSON document on stdout**,
structured errors on stderr, and semantic exit codes. Pass `--human` for tables.

## Install

Build from source (Go 1.21+):

```sh
make build
./bin/paginegialle --help
```

Cross-compile a release matrix (linux/darwin/windows × amd64/arm64):

```sh
make cross      # binaries in dist/
```

## Commands

### `search` — find businesses by category/keyword and location

```sh
paginegialle search --what idraulici --where roma
paginegialle search --what "centro estetico" --where milano --max-pages 3
paginegialle search --what ristoranti --where torino --fields name,phones,url
```

JSON (`stdout`, one document):

```json
{"schema_version":1,
 "query":{"what":"idraulici","where":"roma","page":1},
 "items":[{"id":"waveimpiantiesistemi-roma","name":"Wave Impianti",
           "categories":["Impianti idraulici e termoidraulici"],
           "address":{"street":"Via Tiburtina, 1072","postal_code":"00156","city":"Roma","province":"RM"},
           "phones":["06 33221417","376 2961468"],"snippet":"…",
           "rating":{"value":4.8,"reviews":37},
           "url":"https://www.paginegialle.it/waveimpiantiesistemi-roma"}],
 "pagination":{"page":1,"per_page":25,"has_more":true,"next_page":2}}
```

### `get` — one enriched business profile (bare object)

```sh
paginegialle get ristorante-al-mago-albairate-mi
paginegialle get https://www.paginegialle.it/osteria-macondo-roma
```

Adds legal name, VAT/tax id, geo, email, website, social links, rating and
opening hours (from the page's schema.org JSON-LD).

### `contacts` — the headline: a normalized B2B contact list

Searches and (by default) fetches each profile to add email/website/VAT.

```sh
paginegialle contacts --what idraulici --where roma --max 50
paginegialle contacts --what "studi legali" --where milano --max 20 --enrich=false
```

JSON envelope adds `stats`:

```json
{"schema_version":1,
 "query":{"what":"idraulici","where":"roma","page":1,"max":50},
 "items":[{"name":"…","categories":["…"],"address":{…},"phones":["…"],
           "email":"info@…","website":"https://…","vat":"…","url":"…"}],
 "pagination":{"page":1,"per_page":25,"has_more":true,"next_page":3},
 "stats":{"requested":50,"enriched":50,"with_email":31,"with_website":44}}
```

Also available: `paginegialle version` and `paginegialle completion <shell>`.

## Configuration

| Env var | Flag | Default |
|---------|------|---------|
| `PAGINEGIALLE_BASE_URL` | `--base-url` | `https://www.paginegialle.it` |
| `PAGINEGIALLE_CONTACT` | `--contact` | (none) — appended to the User-Agent |
| `PAGINEGIALLE_RETRIES` | `--retries` | `3` |
| `PAGINEGIALLE_TIMEOUT` | `--timeout` | `30s` |
| — | `--delay` | `1s` minimum between requests |
| — | `--cache-ttl` | `24h` on-disk cache TTL |
| — | `--no-cache` | disable the cache |
| — | `--json` / `--human` | JSON is the default output |

Resolution order: **flag > environment variable > default**. `--concurrency`
(default `1`) controls parallel profile fetches in `contacts`.

## Output contract

- Every data command emits exactly **one JSON document on stdout** (or a table
  with `--human`). Nothing else — no banners, no progress chatter.
- Success ⇒ **empty stderr**. Failure in JSON mode ⇒ a single-line JSON error on
  stderr: `{"ok":false,"error":{"code":"not_found","message":"…","http_status":404}}`.
- Exit codes: `0` success (including empty results), `1` runtime/API failure,
  `2` usage error.
- `NO_COLOR` is honored and piped/non-TTY output carries no ANSI escapes.
- Envelope field order and types are stable; `schema_version` bumps on breaking
  changes.

Error `code` values: `invalid_argument`, `missing_credential`, `auth_failed`,
`not_found`, `rate_limited`, `api_error`, `network`, `unsupported`, `internal`.

## Responsible use

This tool reads paginegialle.it over HTTP. Note that the site's `robots.txt`
disallows the `/ricerca/` search path for automated agents; by using this CLI you
accept responsibility for compliance with the site's terms and applicable law.

The CLI is deliberately polite: a minimum delay between requests (`--delay`), a
transparent User-Agent, retry with backoff honoring `Retry-After`, and an on-disk
response cache. Set `--contact` to identify yourself. Prefer `--enrich=false`
when you do not need per-profile detail, and keep `--max` small.

## Development

```sh
make test    # go test ./...
make vet     # go vet ./...
make build   # bin/paginegialle with version stamps
```

Parsing lives in `internal/paginegialle` and is pinned by golden tests against
captured fixtures in `internal/paginegialle/testdata`.

## License

MIT
