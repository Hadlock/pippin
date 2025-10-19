# 🍎 Pippin - Tiny JIRA Clone

A minimal, single-binary Go web app (~960 LOC) that serves a cozy kanban board for simple project tracking.

## ✨ What Does This Do?

### Core Features
- **Project Management** - Track up to 3 projects (deliberately constrained for simplicity)
- **Kanban Board** - Visual workflow: Backlog → Todo → In Progress → Done
- **Drag & Drop** - Move tickets between columns with your mouse
- **Sprint View** - Filter tickets by current sprint (7 or 14 day cycles)
- **Live Search** - Fuzzy search across tickets (fzf-inspired)
- **Blocking** - Track dependencies between tickets across projects
- **Cozy Themes** - Warm (peachy) or Forest (green) color palettes

### User Interface
- **Add Tickets** - Hero button with modal form (title, description, assignee, state)
- **Add Projects** - Subtle button (max 3 projects, auto-hides at limit)
- **Delete Projects** - Red danger button (appears at 3 projects, CASCADE deletes tickets)
- **Search Box** - Real-time fuzzy filtering (press `/` to focus)
- **Project Filter** - Dropdown to view specific project or all projects
- **Sprint Toggle** - Switch between current sprint and all tickets

### Ticket Features
- **States**: Backlog, Todo, In Progress, Done
- **Move Left/Right** - Adjacent-only transitions (enforces workflow)
- **Drag & Drop** - Visual feedback, validates state transitions
- **Assignees** - Track who's working on what
- **Blocking** - Mark tickets as blocked by others (⚠️ badges)
- **Auto-Timestamps** - Created/updated dates handled automatically

### Technical
- **Single Binary** - Entire app in one Go executable
- **PostgreSQL Backend** - Reliable, production-ready storage
- **12-Factor App** - Fully configurable via environment variables
- **Stateless** - Can be horizontally scaled
- **Minimal Dependencies** - Just Go stdlib + PostgreSQL driver
- **CASCADE Deletes** - Database handles referential integrity
- **Server-Rendered** - No JavaScript framework needed (~100 lines of vanilla JS)

---

## 🚀 Quick Start (From Scratch)

**Easy Mode:** Use the initialization script!

```bash
# Make it executable
chmod +x INIT.sh

# Run it
./INIT.sh
```

The script will:
1. ✅ Check prerequisites (Docker, Go)
2. 🌐 Create Docker network
3. 🐘 Start PostgreSQL container
4. 🗄️ Initialize database schema
5. 🎯 Optionally add demo data
6. 📦 Install Go dependencies
7. 🔨 Build the binary
8. ⚙️ Create .env configuration
9. 🚀 Create run.sh script

Then start the app:
```bash
./run.sh
```

Open **http://localhost:8080/board** in your browser!

---

## 📖 Manual Setup (Step by Step)

### Prerequisites
- Docker & Docker Compose (for PostgreSQL)
- Go 1.23+ (for building the app)

### Step 1: Start PostgreSQL

```bash
# Create network
docker network create pippin-net

# Start PostgreSQL
docker run -d --name pippin-db \
  --network pippin-net \
  -e POSTGRES_USER=pippin \
  -e POSTGRES_PASSWORD=pippin \
  -e POSTGRES_DB=pippin \
  -p 5432:5432 \
  postgres:17
```

### Step 2: Initialize Database Schema

```bash
# Wait for PostgreSQL to be ready
sleep 5

# Create tables
docker exec -i pippin-db psql -U pippin -d pippin << 'EOF'
-- Projects table
CREATE TABLE IF NOT EXISTS projects (
  id SERIAL PRIMARY KEY,
  account_id TEXT NOT NULL,
  key TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT now(),
  UNIQUE (account_id, key)
);

-- Tickets table
CREATE TABLE IF NOT EXISTS tickets (
  id SERIAL PRIMARY KEY,
  account_id TEXT NOT NULL,
  project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  body TEXT DEFAULT '',
  state TEXT CHECK (state IN ('backlog','todo','in_progress','done')),
  assignee TEXT DEFAULT '',
  created_at TIMESTAMP DEFAULT now(),
  updated_at TIMESTAMP DEFAULT now()
);

-- Blocking relationships table
CREATE TABLE IF NOT EXISTS blocks (
  blocker_ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE,
  blocked_ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE,
  account_id TEXT NOT NULL,
  PRIMARY KEY (blocker_ticket_id, blocked_ticket_id),
  CHECK (blocker_ticket_id != blocked_ticket_id)
);
EOF
```

### Step 3: Build & Run Pippin

```bash
# Install Go dependencies
go mod tidy

# Build the binary
go build -o pippin main.go

# Run the app
./pippin
```

The app will start on **http://localhost:8080**

---

## 🔧 Configuration

All configuration via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port |
| `DATABASE_URL` | `postgres://pippin:pippin@localhost:5432/pippin?sslmode=disable` | PostgreSQL connection string |
| `ACCOUNT_ID` | `demo` | Tenant identifier |
| `SPRINT_LENGTH_DAYS` | `7` | Sprint duration (7 or 14) |
| `SPRINT_EPOCH` | `2025-01-01` | Sprint start date (ISO format) |
| `COZY_THEME` | `warm` | UI theme (warm or forest) |

Example `.env` file:
```bash
PORT=8080
DATABASE_URL=postgres://pippin:pippin@localhost:5432/pippin?sslmode=disable
ACCOUNT_ID=demo
SPRINT_LENGTH_DAYS=7
SPRINT_EPOCH=2025-01-01
COZY_THEME=warm
```

---

## 📚 API Endpoints

### Projects
- `GET /api/projects` - List all projects
- `POST /api/projects` - Create project (max 3)
  ```json
  {"key": "PROJ", "name": "Project Name"}
  ```
- `DELETE /api/projects/{key}` - Delete project + tickets

### Tickets
- `GET /api/tickets?project=KEY&sprint=current|all` - List tickets
- `POST /api/tickets` - Create ticket
  ```json
  {
    "project_key": "PROJ",
    "title": "Ticket title",
    "body": "Description",
    "assignee": "username",
    "state": "backlog"
  }
  ```
- `POST /api/tickets/{id}/move` - Move left/right
  ```json
  {"direction": "left|right"}
  ```
- `POST /api/tickets/{id}/blocks` - Add blocking relationship
  ```json
  {"blocked_id": 5}
  ```
- `DELETE /api/tickets/{id}/blocks/{blocked_id}` - Remove block

### Board
- `GET /board?project=KEY&sprint=current|all` - Kanban view

---

## 🎯 Usage Guide

### Creating Projects

1. Click **"+ Add Project"** button (only visible when < 3 projects)
2. Enter project key (e.g., `CART`, `STORE`) - auto-converts to uppercase
3. Enter project name (e.g., "Apple Cart")
4. Click **"Create Project"**

### Adding Tickets

1. Click **"+ Add Ticket"** button (hero button, always visible)
2. Select project from dropdown
3. Fill in:
   - **Title** (required) - Brief description
   - **Description** - Additional details
   - **Assignee** - Who's working on it
   - **Initial State** - Backlog, Todo, In Progress, or Done
4. Click **"Create Ticket"**

### Moving Tickets

**Method 1: Drag & Drop**
- Click and drag any ticket card
- Drop on adjacent column (only adjacent moves allowed)
- Visual feedback shows valid drop zones

**Method 2: Arrow Buttons**
- Click **←** or **→** buttons on ticket cards
- Moves ticket to adjacent state

### Searching Tickets

- Click search box or press **`/`** key
- Type fuzzy query (e.g., "alc" finds "alice")
- Searches: ticket ID, title, project key, assignee
- Results filter in real-time
- Click **×** to clear

### Deleting Projects

1. Ensure you have 3 projects (button only shows at limit)
2. Select specific project from dropdown
3. Click red **"🗑️ Delete Project"** button
4. Confirm deletion in dialog
5. Project and all its tickets are deleted (CASCADE)

---

## ⌨️ Keyboard Shortcuts

- **`/`** - Focus search box
- **`Escape`** - Close modals / blur search
- **Click outside** - Close modals

---

## 🎨 Themes

### Warm Theme (Default)
Peachy, cozy colors with soft orange accents

### Forest Theme
Green, natural palette with mint accents

Change theme:
```bash
export COZY_THEME=forest
./pippin
```

---

## 📊 State Machine

Tickets flow through states with adjacent-only transitions:

```
Backlog ⟷ Todo ⟷ In Progress ⟷ Done
```

Only adjacent moves are allowed to enforce workflow discipline.

---

## 🏗️ Architecture

### Tech Stack
- **Backend**: Go 1.23+ with `database/sql` + `lib/pq`
- **Database**: PostgreSQL 17
- **Frontend**: Server-rendered HTML with embedded CSS/JS (~100 LOC vanilla JavaScript)
- **Deployment**: Single binary, stateless, 12-factor

### File Structure
```
pippin/
├── main.go              # Complete app (~960 lines)
├── INIT.sh              # Initialization script
├── README.md            # This file
├── CLAUDE.md            # Original design spec
├── DATABASE.md          # Database setup guide
├── go.mod               # Go dependencies
└── Makefile             # Build shortcuts
```

### Database Schema
- **projects** - 3 max per account
- **tickets** - Unlimited, linked to projects
- **blocks** - Many-to-many relationships
- All with CASCADE delete for safety

---

## 🧪 Testing

```bash
# Test project limit
curl -X POST http://localhost:8080/api/projects \
  -H 'Content-Type: application/json' \
  -d '{"key":"TEST","name":"Test Project"}'

# Create ticket
curl -X POST http://localhost:8080/api/tickets \
  -H 'Content-Type: application/json' \
  -d '{"project_key":"TEST","title":"Test ticket","assignee":"bob","state":"todo"}'

# Move ticket
curl -X POST http://localhost:8080/api/tickets/1/move \
  -H 'Content-Type: application/json' \
  -d '{"direction":"right"}'

# Delete project
curl -X DELETE http://localhost:8080/api/projects/TEST
```

---

## 🐛 Troubleshooting

### Database Connection Failed
```bash
# Check PostgreSQL is running
docker ps | grep pippin-db

# Check connection
docker exec -it pippin-db psql -U pippin -d pippin -c "SELECT 1;"
```

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080

# Kill it
kill -9 <PID>
```

### Foreign Key Errors
Database uses CASCADE deletes. If you see errors:
```bash
# Verify CASCADE is set
docker exec -i pippin-db psql -U pippin -d pippin -c "\d tickets"
# Should show: ON DELETE CASCADE
```

---

## 🚀 Production Deployment

### Recommended Setup
1. **Reverse proxy** (nginx/Caddy) with HTTPS
2. **Authentication** (OAuth, basic auth, etc.)
3. **PostgreSQL** with backups
4. **Environment variables** for configuration
5. **Monitoring** (logs, metrics)
6. **Rate limiting** if exposed publicly

### Docker Compose Example

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  db:
    image: postgres:17
    environment:
      POSTGRES_USER: pippin
      POSTGRES_PASSWORD: pippin
      POSTGRES_DB: pippin
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  app:
    build: .
    environment:
      PORT: 8080
      DATABASE_URL: "postgres://pippin:pippin@db:5432/pippin?sslmode=disable"
      ACCOUNT_ID: demo
      SPRINT_LENGTH_DAYS: 7
      SPRINT_EPOCH: "2025-01-01"
      COZY_THEME: warm
    ports:
      - "8080:8080"
    depends_on:
      - db

volumes:
  pgdata:
```

Run:
```bash
docker-compose up -d
```

---

## 📝 License

MIT

---

## 🍎 Credits

Built following the CLAUDE.md spec - a tiny JIRA-ish Go app for simple project tracking.

**Features:**
- ~960 lines of Go (including HTML template)
- Single binary deployment
- PostgreSQL backend
- Cozy, minimal UI
- Perfect for small teams (1-10 people, up to 3 projects)

---

**Happy tracking! 🍎**
