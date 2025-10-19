# 🍎 Pippin - Tiny JIRA Clone

A minimal, single-binary Go web app (~400 LOC) that serves a cozy kanban board for simple project tracking.

## Features

- **3 Projects Max** - Deliberately constrained for simplicity
- **Kanban Board** - Backlog → Todo → In Progress → Done
- **Sprint View** - Rolling 1-week or 2-week sprints
- **Blocking** - Track ticket dependencies across projects
- **Cozy Themes** - Warm or Forest color palettes
- **12-Factor App** - Fully configurable via environment variables

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL (or use the provided Docker setup)

### Local Development

```bash
# Start PostgreSQL with PostgREST (optional)
docker network create pippin-net
docker run -d --name pippin-db \
  --network pippin-net \
  -e POSTGRES_USER=pippin \
  -e POSTGRES_PASSWORD=pippin \
  -e POSTGRES_DB=pippin \
  -p 5432:5432 \
  postgres:17

# Initialize schema (see DATABASE.md)
docker exec -i pippin-db psql -U pippin -d pippin < schema.sql

# Run the app
make run
# or
go run main.go
```

Open http://localhost:8080/board to see your kanban board!

## Configuration

All configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port |
| `DATABASE_URL` | `postgres://pippin:pippin@localhost:5432/pippin?sslmode=disable` | PostgreSQL connection string |
| `ACCOUNT_ID` | `demo` | Tenant identifier |
| `SPRINT_LENGTH_DAYS` | `7` | Sprint duration (7 or 14) |
| `SPRINT_EPOCH` | `2025-01-01` | Sprint start date |
| `COZY_THEME` | `warm` | UI theme (warm or forest) |

## API Endpoints

### Projects
- `GET /api/projects` - List all projects
- `POST /api/projects` - Create project (max 3 per account)

### Tickets
- `GET /api/tickets?sprint=current&project=ALL` - List tickets
- `POST /api/tickets` - Create ticket
- `POST /api/tickets/:id/move` - Move ticket left/right
- `POST /api/tickets/:id/blocks` - Add blocking relationship
- `DELETE /api/tickets/:id/blocks/:blocked_id` - Remove block

### Board
- `GET /board?sprint=current&project=ALL` - View kanban board

## Example Usage

```bash
# Create a ticket
curl -X POST http://localhost:8080/api/tickets \
  -H 'Content-Type: application/json' \
  -d '{
    "project_key":"CART",
    "title":"Choose wheel size",
    "body":"12 vs 14 inch",
    "assignee":"jane",
    "state":"backlog"
  }'

# Move ticket forward
curl -X POST http://localhost:8080/api/tickets/1/move \
  -H 'Content-Type: application/json' \
  -d '{"direction":"right"}'

# Add blocking relationship
curl -X POST http://localhost:8080/api/tickets/1/blocks \
  -H 'Content-Type: application/json' \
  -d '{"blocked_id":3}'
```

## Project Structure

```
pippin/
├── main.go           # ~400 LOC - entire application
├── Makefile          # convenience commands
├── README.md         # this file
├── CLAUDE.md         # design spec
└── DATABASE.md       # database setup guide
```

## State Machine

Tickets flow through states with adjacent-only transitions:

```
backlog ⟷ todo ⟷ in_progress ⟷ done
```

You can only move one step at a time in either direction.

## Architecture

- **Single binary** - No dependencies beyond Go stdlib + pq driver
- **PostgreSQL** - Simple, reliable storage
- **Server-rendered HTML** - No JS framework needed
- **Minimal JavaScript** - ~20 LOC for move buttons and backlog toggle
- **Embedded templates** - Everything in one file

## License

MIT
