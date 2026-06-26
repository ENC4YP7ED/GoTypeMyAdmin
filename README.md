# GoTypeMyAdmin

A modern, fully-typed reimagining of **phpMyAdmin** — rebuilt on a **Go**
backend and a **TypeScript** frontend, wrapped in a true-OLED black, grayscale
design system with **Inter** / **JetBrains Mono** typography and **Font Awesome**
iconography.

No PHP. No jQuery. No framework on the frontend either — every widget is a
hand-built custom component.

```
┌──────────────────────────────────────────────────────────────┐
│  GoTypeMyAdmin · root@127.0.0.1:3306        Home › shop › …    │
├───────────────┬──────────────────────────────────────────────┤
│  ▸ shop    3   │  Browse · Structure · SQL · Export           │
│    ▸ customers │  ┌──────────────────────────────────────┐    │
│    ▸ orders    │  │ #  id   name          balance  …      │    │
│  ▸ mysql   31  │  │ 1  3    Grace Hopper  999.99   …      │    │
│                │  └──────────────────────────────────────┘    │
└───────────────┴──────────────────────────────────────────────┘
```

## Why

phpMyAdmin is indispensable but shows its age: server-rendered PHP, a sprawling
jQuery frontend, and a dated UI. GoTypeMyAdmin keeps the mental model (servers →
databases → tables → rows, plus a raw SQL console) and rebuilds the stack:

- **Backend** — a small, dependency-light Go REST API that owns the MySQL /
  MariaDB connection pools and does all introspection through
  `information_schema`.
- **Frontend** — a single-page TypeScript app with its own reactive core and a
  bespoke component library. The Go binary serves the built assets, so
  deployment is one process.

## Architecture

```
GoTypeMyAdmin/
├── backend/                 Go REST API + static server
│   ├── main.go
│   └── internal/
│       ├── server/          HTTP wiring, SPA fallback
│       ├── session/         live *sql.DB pools keyed by bearer token
│       ├── db/              MySQL/MariaDB introspection & query exec
│       └── api/             REST handlers + auth middleware
└── frontend/                Vite + TypeScript SPA
    └── src/
        ├── core/            reactive signals, hyperscript DOM, popover layer
        ├── components/      the custom component library (see below)
        ├── views/           Connect, AppShell, Home, Database, Table, SQL
        ├── api/             typed fetch client
        ├── state/           tiny global store + router
        └── styles/          design tokens + component/view CSS
```

### Request flow

1. The connect screen `POST /api/connect` with host/port/user/password.
2. The backend opens a verified `*sql.DB` pool, stores it in an in-memory
   session, and returns an opaque **bearer token** (kept in `localStorage`).
3. Every later call sends `Authorization: Bearer <token>`; the backend resolves
   it to the live pool. Idle sessions are reaped after `--session-ttl`.

Credentials are proxied straight to the database and never persisted by the
browser beyond the session token.

## Custom component library

Everything is built from scratch on a ~120-line reactive core (`signal` /
`effect` / `computed`) and an `el()` hyperscript helper — no React/Vue/Svelte.

| Component        | Notes                                                       |
| ---------------- | ----------------------------------------------------------- |
| `Button` / `IconButton` | variants (primary/ghost/danger/subtle), sizes, loading |
| `TextInput` / `TextArea` | leading icon, clear button, password reveal, error state |
| `Select`         | fully custom combobox, optional searchable filter           |
| `Collapse`       | animated accordion section                                  |
| `Menu`           | dropdown menus with submenus, separators, headers, shortcuts |
| `ContextMenu`    | right-click menus on tree rows and grid rows                |
| `Scroller`       | custom overlay scrollbars (auto-hide, draggable thumbs)     |
| `Tabs`           | lazy-rendered, cached panels                                |
| `Modal` / `confirmModal` | focus-safe dialog with action buttons               |
| `Toast`          | stacked notifications (success/error/info/warning)          |
| `Tooltip`        | viewport-aware hover tooltips                               |
| `Tree`           | lazy navigation tree (databases → tables)                   |
| `DataGrid`       | sticky header, sortable columns, NULL styling, cell copy    |
| `CodeEditor`     | SQL editor with line-number gutter + syntax highlighting    |
| `Badge` `Spinner` `EmptyState` `Segmented` `Switch` `Field` | primitives |

## Design system

True-OLED black canvas (`#000000`), a strict 17-step grayscale ramp, and the
only colour reserved for semantic states (success / danger / warning), all
desaturated to sit calmly on black. Tokens live in
`frontend/src/styles/tokens.css`.

- **UI font:** Inter (self-hosted via `@fontsource`)
- **Mono font:** JetBrains Mono (data, SQL, identifiers)
- **Icons:** Font Awesome Free

## Features

- Connect to any MySQL / MariaDB server
- Server overview: version, uptime, database/table/size rollups
- Navigate databases & tables in a lazy sidebar tree
- **Browse** table rows with pagination, page-size control and column sorting
- **Insert / edit / duplicate / delete rows** with a generated form (NULL &
  DEFAULT toggles, auto-increment awareness, primary-key-based row identity)
- **Structure** view: columns, keys, defaults, indexes
- **SQL console** (global, per-database, or per-table) with syntax highlighting,
  `Ctrl/Cmd+Enter` to run, timing and affected-row reporting
- **Import** a `.sql` script: multi-statement execution with comment handling
  and per-statement error reporting
- **Export** a table as **SQL / CSV / JSON**, or dump a whole database to SQL
- **Users & privileges**: list accounts, view grants, create users with a
  privilege preset & scope, drop users
- Create / drop databases and drop tables, with confirmation dialogs
- Right-click context menus throughout

## Download a prebuilt binary

Each release ships a **self-contained single file per platform** — the frontend
is embedded into the binary, so there's nothing else to install. Grab the
archive for your OS/CPU from the [Releases](../../releases) page, extract, and
run:

```bash
tar xzf gotypemyadmin_*_linux_amd64.tar.gz   # or unzip the .zip on Windows
./gotypemyadmin -addr :8088                   # → open http://localhost:8088
```

Releases are cross-compiled for **every Go-supported OS/CPU target** (~45),
including:

| OS | Architectures |
| -- | ------------- |
| **Linux** | amd64, arm64, arm, 386, ppc64le, ppc64, riscv64, s390x, mips/mips64(le), loong64 |
| **macOS** (darwin) | amd64 (Intel), arm64 (Apple Silicon) |
| **Windows** | amd64, arm64, 386 |
| **FreeBSD / OpenBSD / NetBSD / DragonFly** | amd64, arm64, arm, 386, … |
| **Solaris / illumos / AIX** | amd64, ppc64 |
| **Android** | amd64, arm64, arm, 386 |

Verify downloads against `SHA256SUMS` in the release.

### Build releases yourself

```bash
make release          # cross-compiles every target into ./dist-bin (+ SHA256SUMS)
# or a subset:
TARGETS="linux/amd64 darwin/arm64 windows/amd64" make release
```

`scripts/build-release.sh` builds the frontend once, embeds it, then loops the
Go matrix with `CGO_ENABLED=0` (no C toolchain needed — the MySQL driver is pure
Go). A tagged push (`git tag v0.1.0 && git push --tags`) runs the same matrix in
CI and attaches the archives to a GitHub Release.

## Running from source

Requirements: **Go 1.22+** and **Node 18+**.

```bash
# 1. install frontend deps
make install

# 2a. production-style: build the SPA and serve everything from Go on :8088
make run
#    → open http://localhost:8088

# 2b. one self-contained binary for your machine (frontend embedded)
make build-embed && ./bin/gotypemyadmin

# 2c. or hot-reload dev: two terminals
make dev-backend      # Go API on :8088
make dev-frontend     # Vite on :5173 (proxies /api → :8088)
```

Need a database to point at? Spin up a throwaway one:

```bash
make test-db          # MariaDB on 127.0.0.1:13306  (root / secret)
# ...connect with host 127.0.0.1, port 13306, user root, password secret
make test-db-stop
```

### Configuration

| Flag / env             | Default              | Purpose                          |
| ---------------------- | -------------------- | -------------------------------- |
| `-addr` / `GTMA_ADDR`  | `:8088`              | listen address                   |
| `-static` / `GTMA_STATIC` | `../frontend/dist`| built frontend directory         |
| `-session-ttl`         | `2h`                 | idle lifetime of a DB session    |
| `-allow-hosts` / `GTMA_ALLOW_HOSTS` | _(empty = any)_ | comma-separated allowlist of DB hosts clients may connect to |
| `-version`             | —                    | print version + platform and exit |

## REST API

| Method & path | Purpose |
| ------------- | ------- |
| `POST /api/connect` | open a connection, return a session token |
| `POST /api/disconnect` | close the session |
| `GET /api/server/info` | version, uptime, charset |
| `GET /api/databases` | list schemas with counts/sizes |
| `POST /api/databases` | create a database |
| `DELETE /api/databases/{db}` | drop a database |
| `GET /api/databases/{db}/tables` | list tables/views |
| `GET /api/databases/{db}/tables/{t}/columns` | column structure |
| `GET /api/databases/{db}/tables/{t}/indexes` | indexes |
| `GET /api/databases/{db}/tables/{t}/rows` | paginated browse (`limit`,`offset`,`orderBy`,`dir`) |
| `GET /api/databases/{db}/tables/{t}/ddl` | `SHOW CREATE TABLE` |
| `GET /api/databases/{db}/tables/{t}/export?format=` | export table as sql/csv/json |
| `GET /api/databases/{db}/export` | dump a whole database to SQL |
| `DELETE /api/databases/{db}/tables/{t}` | drop a table |
| `POST /api/databases/{db}/tables/{t}/insert` | insert a row |
| `POST /api/databases/{db}/tables/{t}/update` | update a row (by identity) |
| `POST /api/databases/{db}/tables/{t}/delete` | delete a row (by identity) |
| `GET /api/users` | list accounts |
| `POST /api/users` | create a user (+ optional grant) |
| `GET /api/users/{user}/{host}/grants` | `SHOW GRANTS` |
| `DELETE /api/users/{user}/{host}` | drop a user |
| `POST /api/query` | run arbitrary SQL `{database, sql}` |
| `POST /api/import` | run a multi-statement SQL script |

## Security

GoTypeMyAdmin was audited against the recurring phpMyAdmin vulnerability classes
(the [PMASA](https://www.phpmyadmin.net/security/) advisories) and related CVEs.
How each class is handled here:

| phpMyAdmin class (examples) | GoTypeMyAdmin |
| --------------------------- | ------------- |
| **XSS** from crafted table/db/column names or data — the single largest PMASA category (e.g. PMASA-2025-1, -2025-2, -2023-1) | The DOM layer (`core/dom.ts`) writes text as `textContent` by default; the only two `innerHTML` sinks (SQL highlighters) HTML-escape first. A strict **Content-Security-Policy** (`script-src 'self'`, no `unsafe-inline`) is sent as defense-in-depth. |
| **SQL injection** (PMASA-2020-x: username/search/user-accounts) | Identifiers go through backtick-quoting (`QuoteIdent`); row reads/writes use bound `?` parameters; `ORDER BY` direction is whitelisted. The `GRANT` builder — which cannot bind parameters — validates privileges against an allowlist and canonicalizes the scope. |
| **CSRF** (CVE-2019-12616) | Auth is a `Bearer` token in the `Authorization` header (not an ambient cookie), so cross-site requests can't carry it. |
| **SSRF / arbitrary-server proxy** (PMASA-2017-6; cf. `AllowArbitraryServer`) | The connect host can be restricted with `-allow-hosts`; the server logs a warning when it's unset. |
| **Resource exhaustion / DoS** | Request bodies capped (64 MiB), ad-hoc query results capped (100k rows, flagged `truncated`), browse paging capped (1000/page). |
| **Path traversal / LFI** | Static files served via `http.Dir` + cleaned paths; no user-controlled file reads. |
| **RCE via `preg_replace /e`, file include, deserialization** | Not applicable — no `eval`, no dynamic includes, no PHP. |
| **Clickjacking / MIME sniffing** | `X-Frame-Options: DENY`, `frame-ancestors 'none'`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`. |

**Operator responsibilities (not handled by the app):**

- **Terminate TLS** in front of it (a reverse proxy). Connection credentials are
  proxied to the database in the request body — never run it over plain HTTP on
  an untrusted network.
- **Set `-allow-hosts`** to the database hosts you actually use, to avoid the
  server being usable as an internal port-scanner.
- Consider **rate-limiting `/api/connect`** at the proxy to blunt credential
  brute-forcing. The session token lives in `localStorage`; the strict CSP +
  escaping keep it out of reach of injected script, but treat the origin as
  trusted.

## Status & roadmap

A working tool covering the most-used phpMyAdmin workflows: browse & edit data,
SQL console, import/export, schema browsing, and user management. Natural next
steps: relation/foreign-key visualization, bulk row operations, CSV *import*,
saved queries/bookmarks, and multi-engine support (PostgreSQL/SQLite behind the
same API).

## License

For personal/educational use. phpMyAdmin is a trademark of its respective
owners; GoTypeMyAdmin is an independent reimplementation, not affiliated.
