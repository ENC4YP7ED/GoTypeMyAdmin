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

## Running it

Requirements: **Go 1.22+** and **Node 18+**.

```bash
# 1. install frontend deps
make install

# 2a. production-style: build the SPA and serve everything from Go on :8088
make run
#    → open http://localhost:8088

# 2b. or hot-reload dev: two terminals
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

## Status & roadmap

A working tool covering the most-used phpMyAdmin workflows: browse & edit data,
SQL console, import/export, schema browsing, and user management. Natural next
steps: relation/foreign-key visualization, bulk row operations, CSV *import*,
saved queries/bookmarks, and multi-engine support (PostgreSQL/SQLite behind the
same API).

## License

For personal/educational use. phpMyAdmin is a trademark of its respective
owners; GoTypeMyAdmin is an independent reimplementation, not affiliated.
