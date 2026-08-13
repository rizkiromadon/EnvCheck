# EnvCheck

A fast, lightweight, cross-platform CLI for validating, analyzing, and
security-checking `.env` files. Built for local development and CI/CD
pipelines alike.

- **Zero runtime dependencies** — a single static Go binary.
- **Never modifies your files** — read-only by design.
- **Never leaks secrets** — values are always masked in output and logs.
- **Works fully offline** — no network access required.

## Why Go?

EnvCheck is written in Go because the tool's core requirements — a single
static binary, fast startup, low memory footprint, easy cross-compilation,
and painless packaging into `.deb`, `.rpm`, and Arch (`PKGBUILD`) — map
directly onto what the Go toolchain is built for. There is no runtime to
install, no interpreter version to match, and `go build` produces a fully
static binary per target platform out of the box.

## Install

### From source

```bash
git clone https://github.com/rizkiromadon/envcheck.git
cd envcheck
go build -ldflags "-s -w -X main.version=1.0.0" -o envcheck ./cmd/envcheck
sudo mv envcheck /usr/local/bin/
```

### Debian / Ubuntu (.deb)

```bash
sudo dpkg -i envcheck_1.0.0_amd64.deb
```

### Fedora / RHEL (.rpm)

```bash
sudo rpm -i envcheck-1.0.0-1.x86_64.rpm
```

### Arch Linux

```bash
makepkg -si
```

See [`packaging/`](./packaging) for the packaging manifests used to build
each of these.

## Quick start

```bash
# Check ./.env by default
envcheck

# Check a specific file
envcheck .env.production

# Explicit subcommands (aliases of the default behavior)
envcheck check .env
envcheck doctor .env

# Compare against .env.example
envcheck .env --compare .env.example

# Validate against a schema
envcheck .env --schema .env.schema

# CI-friendly JSON output
envcheck .env --json --strict --no-color
```

## Example output

```
EnvCheck v1.0.0

File: .env

✓ Syntax
✓ Variable names
✗ duplicate key 'PORT' declared on lines [2 7]
⚠ 'API_KEY' has an empty value
⚠ possible password detected in 'DATABASE_PASSWORD' [Su****d!]

Summary
12 variables
1 error
1 warning
```

## Features

### `.env` parsing

- Standard `KEY=VALUE` syntax, full-line and inline `#` comments.
- Unquoted, single-quoted, and double-quoted values.
- Double-quoted values may span multiple physical lines (useful for PEM
  keys or JSON blobs); single-quoted values follow strict POSIX semantics
  and do not span lines.
- `export KEY=value` syntax.
- Never writes to or modifies the source file.

### Validation

- Invalid syntax (missing `=`, malformed keys, unclosed quotes).
- Duplicate keys, with all line numbers reported.
- Empty values (configurable via `--allow-empty`).
- Invalid variable names (`^[A-Za-z_][A-Za-z0-9_]*$`).
- `${VAR}` reference detection, including references to variables that
  aren't defined anywhere in the file.
- Type validation when a schema is supplied (string, number, boolean, URL,
  port, enum).

### Security scanning

- Detects API keys, tokens, passwords, secrets, private keys, JWTs, and
  database connection strings with embedded credentials.
- Built on an extensible rule table (`internal/security/rules.go`) — add a
  new `Rule{}` entry to extend detection, no engine changes required.
- Severities: `info`, `warning`, `critical`.
- Secret values are **always masked** in terminal and JSON output; raw
  values are never printed or logged.
- Common placeholder values (`changeme`, `xxx`, `your_key_here`, etc.) are
  excluded to reduce false positives.

### `.env.example` comparison

```bash
envcheck .env --compare .env.example
```

Reports variables missing from `.env`, extras not documented in the
example file, duplicates in either file, and the set present in both.

### Schema validation

Define a simple schema file:

```
PORT=number|required
DATABASE_URL=url|required|secret
NODE_ENV=enum:development,production,test|required
DEBUG=boolean|optional
```

```bash
envcheck .env --schema .env.schema
```

Supported types: `string`, `number`, `boolean`, `url`, `port`,
`enum:val1,val2,...`. Modifiers: `required`, `optional` (default), `secret`
(informational, marks the variable as sensitive for tooling/documentation
purposes).

### Git integration

EnvCheck checks (read-only, via `git ls-files`) whether the target file is
tracked by Git and warns if so — a common source of leaked credentials.
It never modifies `.gitignore` or the repository.

### CI/CD

- `--json` — machine-readable output.
- `--strict` — treat warnings as failing conditions.
- `--no-color` — disable ANSI colors.
- `--quiet` — only print failures and the summary.
- Deterministic exit codes (see below) for pipeline gating.

## CLI reference

```
envcheck [file] [flags]
envcheck check [file] [flags]
envcheck doctor [file] [flags]
```

If `[file]` is omitted, EnvCheck looks for `.env` in the current directory.

| Flag              | Description                                       |
|-------------------|-----------------------------------------------------|
| `--compare <file>`| Compare against an `.env.example`-style file         |
| `--schema <file>` | Validate against a schema file                       |
| `--json`          | Output JSON instead of text                           |
| `--strict`        | Treat warnings as failing (non-zero exit)             |
| `--quiet`         | Only print failures and the summary                   |
| `--no-color`      | Disable ANSI colors                                    |
| `--no-git`        | Skip the Git tracking check                            |
| `--no-security`   | Skip the security scanner                              |
| `--allow-empty`   | Do not flag empty values                               |
| `-h`, `--help`    | Show help                                              |
| `-v`, `--version` | Show version                                           |

### Exit codes

| Code | Meaning                                                          |
|------|-------------------------------------------------------------------|
| `0`  | Valid — no errors, no critical security findings                  |
| `1`  | Validation error (syntax, duplicate keys, schema violations, etc.)|
| `2`  | Security issue (critical secret, or any finding in `--strict`)    |
| `3`  | Usage error (bad flags, missing/unreadable file)                  |
| `4`  | Internal error                                                    |

Validation errors take precedence over security findings: a file that
fails to parse can't be meaningfully scanned for secrets either.

## GitHub Actions example

```yaml
- name: Check .env
  run: envcheck .env --schema .env.schema --strict --no-color --json > envcheck-report.json
```

## Architecture

```
cmd/envcheck        CLI entrypoint, argument parsing
internal/model       Shared data structures (Entry, Finding, Report, ...)
internal/parser      Raw text -> structured entries
internal/validator    Structural checks (syntax, duplicates, naming, refs)
internal/security     Secret detection rules + masking
internal/schema       Schema file parsing + type validation
internal/compare      .env vs .env.example diffing
internal/gitcheck     Read-only Git tracking detection
internal/formatter    Human-readable and JSON renderers
internal/exitcode     Centralized exit-code policy
internal/app          Orchestrates the full pipeline into a Report
```

Each concern is isolated behind its own package so new validation rules,
security patterns, or output formats can be added without touching
unrelated code.

## Testing

```bash
go test ./...
```

Test suites cover: valid files, malformed syntax, duplicate keys, empty
values, all quoting styles, multiline values, variable references, secret
detection (including that raw secrets never leak into output), schema
validation, `.env.example` comparison, exit codes, JSON output, and CLI
argument parsing edge cases.

## Design principles

- **Never leak secrets.** Masking happens at the point of detection; raw
  values never reach logs, error messages, or JSON output.
- **Never modify the target file.** EnvCheck is a read-only analysis tool.
- **No network dependency.** All checks run fully offline; the only
  external process invoked is `git`, and only for a read-only tracking
  check.
- **Fail predictably.** Exit codes are stable and documented so CI
  pipelines can depend on them.

## License

MIT
