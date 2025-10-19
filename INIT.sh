#!/bin/bash
# 🍎 Pippin - Initialization Script
# Run this script to set up Pippin from scratch

set -e  # Exit on error

echo "🍎 Starting Pippin initialization..."
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
DB_NAME=pippin
DB_USER=pippin
DB_PASSWORD=pippin
DB_PORT=5432
CONTAINER_NAME=pippin-db
NETWORK_NAME=pippin-net

# Step 1: Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker not found. Please install Docker first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker installed${NC}"

if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go not found. Please install Go 1.23+ first.${NC}"
    exit 1
fi
GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✓ Go installed ($GO_VERSION)${NC}"

echo ""

# Step 2: Create Docker network
echo "🌐 Creating Docker network..."
if docker network inspect $NETWORK_NAME &> /dev/null; then
    echo -e "${YELLOW}⚠ Network $NETWORK_NAME already exists, skipping${NC}"
else
    docker network create $NETWORK_NAME
    echo -e "${GREEN}✓ Created network $NETWORK_NAME${NC}"
fi
echo ""

# Step 3: Start PostgreSQL
echo "🐘 Starting PostgreSQL..."
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo -e "${YELLOW}⚠ Container $CONTAINER_NAME already exists${NC}"
    
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo -e "${GREEN}✓ Container $CONTAINER_NAME is already running${NC}"
    else
        echo "Starting existing container..."
        docker start $CONTAINER_NAME
        echo -e "${GREEN}✓ Started container $CONTAINER_NAME${NC}"
    fi
else
    docker run -d --name $CONTAINER_NAME \
        --network $NETWORK_NAME \
        -e POSTGRES_USER=$DB_USER \
        -e POSTGRES_PASSWORD=$DB_PASSWORD \
        -e POSTGRES_DB=$DB_NAME \
        -p $DB_PORT:5432 \
        postgres:17
    
    echo -e "${GREEN}✓ Created and started PostgreSQL container${NC}"
fi

echo "Waiting for PostgreSQL to be ready..."
sleep 8

# Test connection
if docker exec $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME -c "SELECT 1;" &> /dev/null; then
    echo -e "${GREEN}✓ PostgreSQL is ready${NC}"
else
    echo -e "${RED}❌ PostgreSQL connection failed${NC}"
    exit 1
fi
echo ""

# Step 4: Initialize database schema
echo "🗄️  Initializing database schema..."
docker exec -i $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME << 'EOF'
-- Drop existing tables if they exist (fresh start)
DROP TABLE IF EXISTS blocks CASCADE;
DROP TABLE IF EXISTS tickets CASCADE;
DROP TABLE IF EXISTS projects CASCADE;

-- Projects table
CREATE TABLE projects (
  id SERIAL PRIMARY KEY,
  account_id TEXT NOT NULL,
  key TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT now(),
  UNIQUE (account_id, key)
);

-- Tickets table
CREATE TABLE tickets (
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
CREATE TABLE blocks (
  blocker_ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE,
  blocked_ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE,
  account_id TEXT NOT NULL,
  PRIMARY KEY (blocker_ticket_id, blocked_ticket_id),
  CHECK (blocker_ticket_id != blocked_ticket_id)
);

-- Verify tables created
SELECT 'Created ' || COUNT(*) || ' tables' AS result
FROM information_schema.tables 
WHERE table_name IN ('projects', 'tickets', 'blocks');
EOF

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Database schema initialized${NC}"
else
    echo -e "${RED}❌ Schema initialization failed${NC}"
    exit 1
fi
echo ""

# Step 5: Ask about demo data
echo "🎯 Would you like to add demo data? (y/N)"
read -r ADD_DEMO

if [[ $ADD_DEMO =~ ^[Yy]$ ]]; then
    echo "Adding demo data..."
    docker exec -i $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME << 'EOF'
-- Create demo projects
INSERT INTO projects (account_id, key, name) VALUES
  ('demo', 'CART', 'Apple Cart — build cart'),
  ('demo', 'ORCH', 'Apple Orchard — grow & maintain'),
  ('demo', 'STORE', 'Apple Store — sell apples');

-- Add demo tickets
INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'Design cart frame', 'Wood vs metal decision', 'todo', 'jane'
FROM projects p WHERE p.key='CART';

INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'Build wheels', 'Need 4 sturdy wheels', 'backlog', 'bob'
FROM projects p WHERE p.key='CART';

INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'Soil testing', 'Check pH & nutrients', 'in_progress', 'lee'
FROM projects p WHERE p.key='ORCH';

INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'Prune trees', 'Spring pruning schedule', 'backlog', 'lee'
FROM projects p WHERE p.key='ORCH';

INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'POS setup', 'Pick a simple POS system', 'backlog', 'sam'
FROM projects p WHERE p.key='STORE';

-- Show summary
SELECT 
  'Created ' || 
  (SELECT COUNT(*) FROM projects) || ' projects, ' ||
  (SELECT COUNT(*) FROM tickets) || ' tickets' AS summary;
EOF
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Demo data added${NC}"
    else
        echo -e "${RED}❌ Demo data insertion failed${NC}"
    fi
else
    echo "Skipping demo data"
fi
echo ""

# Step 6: Install Go dependencies
echo "📦 Installing Go dependencies..."
if [ -f "go.mod" ]; then
    go mod tidy
    echo -e "${GREEN}✓ Dependencies installed${NC}"
else
    echo -e "${YELLOW}⚠ go.mod not found, skipping${NC}"
fi
echo ""

# Step 7: Build the application
echo "🔨 Building Pippin..."
if [ -f "main.go" ]; then
    go build -o pippin main.go
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Build successful${NC}"
    else
        echo -e "${RED}❌ Build failed${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ main.go not found${NC}"
    exit 1
fi
echo ""

# Step 8: Configure environment
echo "⚙️  Setting up environment..."
cat > .env << 'EOF'
# Pippin Environment Configuration

# Server
PORT=8080

# Database
DATABASE_URL=postgres://pippin:pippin@localhost:5432/pippin?sslmode=disable

# Account (multi-tenant support)
ACCOUNT_ID=demo

# Sprint settings
SPRINT_LENGTH_DAYS=7        # 7 or 14 day sprints
SPRINT_EPOCH=2025-01-01     # Sprint start date (ISO format)

# Theme
COZY_THEME=warm             # warm or forest
EOF
echo -e "${GREEN}✓ Created .env file${NC}"
echo ""

# Step 9: Create run script
echo "🚀 Creating run script..."
cat > run.sh << 'EOF'
#!/bin/bash
# Load environment variables
set -a
source .env
set +a

# Run Pippin
./pippin
EOF
chmod +x run.sh
echo -e "${GREEN}✓ Created run.sh${NC}"
echo ""

# Step 10: Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}🎉 Pippin initialization complete!${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📊 What was created:"
echo "  • PostgreSQL container: $CONTAINER_NAME (port $DB_PORT)"
echo "  • Database: $DB_NAME with tables: projects, tickets, blocks"
if [[ $ADD_DEMO =~ ^[Yy]$ ]]; then
    echo "  • Demo data: 3 projects, 5 tickets"
fi
echo "  • Binary: ./pippin"
echo "  • Config: .env"
echo "  • Runner: ./run.sh"
echo ""
echo "🚀 To start Pippin:"
echo "  ./run.sh"
echo ""
echo "🌐 Then open in browser:"
echo "  http://localhost:8080/board"
echo ""
echo "🛠️  Useful commands:"
echo "  • View logs:     docker logs -f $CONTAINER_NAME"
echo "  • Stop DB:       docker stop $CONTAINER_NAME"
echo "  • Start DB:      docker start $CONTAINER_NAME"
echo "  • DB shell:      docker exec -it $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME"
echo "  • Rebuild:       go build -o pippin main.go"
echo ""
echo "📖 Configuration:"
echo "  Edit .env to customize settings (port, theme, sprint length, etc.)"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}Happy tracking! 🍎${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
