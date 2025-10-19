# 🍎 “Pippin” — a tiny JIRA-ish Go app (agentic build doc)

Goal: a single-binary Go web app (≈400 LOC) that serves a cozy kanban/sprint board on `:8080`. It’s 12-factor (env-driven), dockerable, and deliberately constrained.

---

## 1) Scope (the box we’ll stay inside)

* Accounts: implicit single-tenant (via `ACCOUNT_ID` env). Future multi-tenant is trivial (all rows have `account_id`).
* Projects: max **3 per account**; enforced at write time.
* Tickets:

  * States: `backlog` → `todo` → `in_progress` → `done` (can move both directions by one step per action).
  * Fields: `id`, `project_id`, `title`, `body`, `state`, `assignee`, `created_at`, `updated_at`.
  * Blocking: many-to-many (“ticket A blocked by B”) across projects within the same account.
* Sprints: rolling 1-week (default) or 2-week (via env). Board filters to **current sprint** window, with toggle to show all.
* UI: a single HTML page (server-rendered) with 4 columns; “Backlog” initially collapsed so you primarily see the **three main columns** (Todo / In Progress / Done).
* Backend storage: **SQLite** (simple, local, docker-friendly).
  (We’ll keep storage abstraction thin so swapping to Postgres later is easy.)

---

## 2) Run-time config (12-factor)

| Env var              | Default                                   | Purpose                                  |
| -------------------- | ----------------------------------------- | ---------------------------------------- |
| `PORT`               | `8080`                                    | HTTP port                                |
| `DATABASE_URL`       | `file:pippin.db?_busy_timeout=5000&_fk=1` | SQLite DSN                               |
| `ACCOUNT_ID`         | `demo`                                    | Partition rows per account               |
| `SPRINT_LENGTH_DAYS` | `7`                                       | `7` or `14` only                         |
| `SPRINT_EPOCH`       | `2025-01-01`                              | Date to anchor sprint windows (ISO date) |
| `COZY_THEME`         | `warm`                                    | `warm` or `forest` palette               |

---

## 3) Data model (SQLite)

```sql
-- schema.sql
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  account_id TEXT NOT NULL,
  key TEXT NOT NULL,           -- short handle like "CART", unique per account
  name TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE (account_id, key)
);

CREATE TABLE IF NOT EXISTS tickets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  account_id TEXT NOT NULL,
  project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL CHECK (state IN ('backlog','todo','in_progress','done')),
  assignee TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS blocks (
  blocker_ticket_id INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  blocked_ticket_id INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  account_id TEXT NOT NULL,
  PRIMARY KEY (blocker_ticket_id, blocked_ticket_id),
  CHECK (blocker_ticket_id != blocked_ticket_id)
);
```

**Constraint:** on project creation, reject if account already has 3 projects.

---

## 4) Sprint window logic (tiny, deterministic)

* Parse `SPRINT_EPOCH` as day 0.
* `length := SPRINT_LENGTH_DAYS` (7 or 14).
* `n := floor((today - epoch) / length)`.
* Current sprint = `[epoch + n*length, epoch + (n+1)*length)`.

This lets everyone see “current sprint” the same way, regardless of weekday.

---

## 5) State machine (move rules)

Allowed transitions:

* Forward: `backlog → todo → in_progress → done`
* Backward: adjacent only (e.g., `in_progress → todo`), not jumpy.

Server enforces adjacency; UI “Move Left/Right” buttons compute the next state.

---

## 6) HTTP interface (≃8 routes total)

All paths are under account context implicitly (from `ACCOUNT_ID`).

**HTML**

* `GET /` → redirect `/board`
* `GET /board?sprint=current|all&project=ALL|KEY` → kanban page (HTML)

**JSON (simple)**

* `GET /api/projects` → list
* `POST /api/projects` → `{key,name}` (reject >3)
* `GET /api/tickets?project=KEY&sprint=current|all` → list
* `POST /api/tickets` → `{project_key,title,body,assignee,state?}`
* `POST /api/tickets/:id/move` → `{direction:"left"|"right"}` (enforce adjacency)
* `POST /api/tickets/:id/blocks` → `{blocked_id}` (create edge)
* `DELETE /api/tickets/:id/blocks/:blocked_id` → remove edge

**Notes**

* Keep handlers tiny. Use `database/sql` + `github.com/mattn/go-sqlite3`.
* Use `html/template` for server-rendered board.
* Include a minimal `<style>` block for the cozy theme; no JS framework needed (a dash of vanilla JS for postbacks).

---

## 7) Cozy theme (CSS tokens)

```css
:root[data-theme="warm"]{
  --bg: #fff8f0; --panel:#fff2e1; --ink:#3b2e2a; --muted:#8b6f64;
  --todo:#ffe3b0; --doing:#ffd1a1; --done:#c7f9cc; --backlog:#e9ecef;
  --accent:#ffb86b; --border:#e2c6b6; --card:#fffaf5;
}
:root[data-theme="forest"]{
  --bg:#f6fff8; --panel:#e9f5ec; --ink:#1b3a2f; --muted:#446a5a;
  --todo:#dff3e3; --doing:#cbe8d6; --done:#b8e0c2; --backlog:#eaf2ee;
  --accent:#7acb9f; --border:#cfe6d7; --card:#f8fffb;
}
body{background:var(--bg);color:var(--ink);font:14px/1.4 system-ui;margin:0}
header{padding:12px 16px;background:var(--panel);border-bottom:1px solid var(--border)}
.board{display:grid;grid-template-columns:260px 1fr 1fr 1fr;gap:12px;padding:12px}
.col{background:var(--card);border:1px solid var(--border);border-radius:12px;overflow:hidden}
.col h3{margin:0;padding:8px 10px;background:var(--panel);border-bottom:1px solid var(--border)}
.col[data-state="backlog"]{background:var(--backlog)}
.col[data-state="todo"]{background:var(--todo)}
.col[data-state="in_progress"]{background:var(--doing)}
.col[data-state="done"]{background:var(--done)}
.card{margin:8px;border:1px dashed var(--border);border-radius:10px;padding:8px;background:#fff8;backdrop-filter:saturate(120%) blur(2px)}
.small{color:var(--muted);font-size:12px}
button{border:1px solid var(--border);background:var(--accent);color:var(--ink);border-radius:999px;padding:4px 10px;margin-right:6px;cursor:pointer}
```

Backlog column starts collapsed (CSS + small JS to toggle), so the “three main columns” are foreground.

---

## 8) File layout (tiny)

```
pippin/
  main.go            # ~400 LOC (handlers, templates, db, sprint math)
  schema.sql
  seed.sql           # optional: test data
  Dockerfile
  Makefile           # quality-of-life targets
  README.md
```

---

## 9) Implementation sketch (what fits in ~400 LOC)

* **Globals:** `db *sql.DB`, `tpl *template.Template`, `cfg struct{…}`, `now func() time.Time`.
* **Init:**

  * parse env → `cfg`
  * open SQLite with `DATABASE_URL`
  * run `schema.sql`
  * parse and compile HTML template (inline in `main.go` to keep files small)
* **Helpers:**

  * `currentSprintRange(now, epoch, length) (start,end time.Time)`
  * `queryTickets(projectKey *string, sprint string) []Ticket`
  * `move(ticketID, dir)` with adjacency check & `updated_at`.
  * `toggleBlock(blocker, blocked, on bool)`
* **HTTP:**

  * mux via `http.NewServeMux()`
  * small JSON helpers: `writeJSON(w,status,any)` and `readJSON(r,&v)`
  * minimal security headers
* **Template:**

  * One template with 4 columns; each ticket shows title, assignee, blockers, and two small buttons (← / →).
  * Submits to `/api/tickets/:id/move` with `fetch()` (10 lines of inline JS).

---

## 10) Seed test data (the three apple projects)

```sql
-- seed.sql (idempotent-ish)
INSERT OR IGNORE INTO projects (account_id,key,name) VALUES
 ('demo','CART','Apple Cart — build cart'),
 ('demo','ORCH','Apple Orchard — grow & maintain'),
 ('demo','STORE','Apple Store — sell apples');

-- A few tickets
INSERT INTO tickets (account_id,project_id,title,body,state,assignee)
SELECT 'demo', p.id, 'Design cart frame','Wood vs. metal', 'todo','jane'
FROM projects p WHERE p.key='CART' AND p.account_id='demo';

INSERT INTO tickets (account_id,project_id,title,body,state,assignee)
SELECT 'demo', p.id, 'Soil testing','Check pH & nutrients', 'in_progress','lee'
FROM projects p WHERE p.key='ORCH' AND p.account_id='demo';

INSERT INTO tickets (account_id,project_id,title,body,state,assignee)
SELECT 'demo', p.id, 'POS setup','Pick a simple POS', 'backlog','sam'
FROM projects p WHERE p.key='STORE' AND p.account_id='demo';
```

Example block:

```sql
-- Make STORE:POS depend on CART:Design frame
INSERT INTO blocks (blocker_ticket_id, blocked_ticket_id, account_id)
SELECT t1.id, t2.id, 'demo'
FROM tickets t1, tickets t2
JOIN projects p1 ON p1.id=t1.project_id
JOIN projects p2 ON p2.id=t2.project_id
WHERE p1.key='CART' AND t1.title='Design cart frame'
  AND p2.key='STORE' AND t2.title='POS setup';
```

---

## 11) Dockerfile (CGO-enabled for sqlite3)

```dockerfile
# Dockerfile
FROM golang:1.23-bookworm AS build
WORKDIR /app
COPY . .
RUN apt-get update && apt-get install -y build-essential ca-certificates && rm -rf /var/lib/apt/lists/*
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /bin/pippin ./...

FROM debian:bookworm-slim
RUN useradd -m -u 10001 app
WORKDIR /app
COPY --from=build /bin/pippin /usr/local/bin/pippin
COPY schema.sql seed.sql ./
ENV PORT=8080 DATABASE_URL="file:/app/pippin.db?_busy_timeout=5000&_fk=1" ACCOUNT_ID=demo SPRINT_LENGTH_DAYS=7 SPRINT_EPOCH=2025-01-01 COZY_THEME=warm
EXPOSE 8080
USER app
CMD ["/usr/local/bin/pippin"]
```

---

## 12) Makefile QoL

```makefile
run:
	ACCOUNT_ID=demo SPRINT_LENGTH_DAYS=7 go run .

db:
	sqlite3 pippin.db < schema.sql
	-sqlite3 pippin.db < seed.sql || true

docker:
	docker build -t pippin:latest .

up:
	docker run --rm -p 8080:8080 -v $$PWD:/app pippin:latest
```

---

## 13) Example flows (acceptance checkpoints)

1. **First boot**

* `schema.sql` applied; visiting `http://localhost:8080/board` shows empty board (or seeded data if `seed.sql` was run).

2. **Enforce ≤3 projects**

```bash
curl -sX POST localhost:8080/api/projects -d '{"key":"X","name":"Extra"}' -H 'Content-Type: application/json'
# If already 3, returns 400 {"error":"project limit reached"}
```

3. **Create a ticket**

```bash
curl -sX POST localhost:8080/api/tickets \
 -H 'Content-Type: application/json' \
 -d '{"project_key":"CART","title":"Choose wheel size","body":"12 vs 14 in","assignee":"jane","state":"backlog"}'
```

4. **Block across projects**

```bash
curl -sX POST localhost:8080/api/tickets/11/blocks -H 'Content-Type: application/json' -d '{"blocked_id": 7}'
```

5. **Move right (enforces adjacency)**

```bash
curl -sX POST localhost:8080/api/tickets/7/move -d '{"direction":"right"}' -H 'Content-Type: application/json'
```

6. **Board view**

* `GET /board?sprint=current&project=ALL` (backlog collapsed; warm theme)
* Click “Backlog” to expand; toggle “Show all tickets” to include non-current entries.

---

## 14) Guardrails & simplicity tricks to hit ~400 LOC

* One file `main.go`; avoid ORM. Use 4–5 small SQL queries with `?` args.
* `template.Must(template.New("board").Parse(boardHTML))` with inline CSS.
* Tiny router with `http.ServeMux`; parameter parsing by splitting URL path.
* Validation helpers (max project count; state adjacency).
* Minimal JS (~20 LOC) for move buttons + backlog collapse.

---

## 15) “Done” criteria

* Runs with `go run .` and with Docker.
* Creates/reads SQLite, applies schema at startup.
* Enforces 3-project limit.
* Board renders with cozy theme; backlog collapsed by default.
* Tickets move left/right with rules; timestamps update.
* Blocks appear as “⚠ blocked by T-123 (ORCH)” badges, and moving **into** `in_progress` is allowed even if blocked (we only **warn**, not hard-prevent, to keep app simple). If you want strictness: flip a flag in code to forbid moving forward when blocked.

---

## 16) Nice-to-haves (only if still under 400 LOC)

* `?project=KEY` filter dropdown in header.
* `?sprint=all` toggle link.
* Simple search box (title contains).

---

## 17) Next steps (what I’ll implement in `main.go`)

1. Parse env → config.
2. Open SQLite; run `schema.sql`; (optionally) run `seed.sql` if db empty.
3. Implement `currentSprintRange`.
4. Implement handlers:

   * `GET /board` (query tickets, group by state, render)
   * 6 JSON endpoints (projects list/create; tickets list/create; move; block add/del)
5. Inline template + CSS + tiny JS.
6. Build Docker image and verify.

If you want, I can now draft the single-file `main.go` that sticks to ~400 LOC with the above behavior and the cozy theme baked in.
