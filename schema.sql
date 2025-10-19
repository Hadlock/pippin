-- Pippin Database Schema
-- PostgreSQL

CREATE TABLE IF NOT EXISTS projects (
  id SERIAL PRIMARY KEY,
  account_id TEXT NOT NULL,
  key TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT now(),
  UNIQUE (account_id, key)
);

CREATE TABLE IF NOT EXISTS tickets (
  id SERIAL PRIMARY KEY,
  account_id TEXT NOT NULL,
  project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  body TEXT DEFAULT '',
  state TEXT NOT NULL CHECK (state IN ('backlog','todo','in_progress','done')),
  assignee TEXT DEFAULT '',
  created_at TIMESTAMP DEFAULT now(),
  updated_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS blocks (
  blocker_ticket_id INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  blocked_ticket_id INTEGER NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  account_id TEXT NOT NULL,
  PRIMARY KEY (blocker_ticket_id, blocked_ticket_id),
  CHECK (blocker_ticket_id != blocked_ticket_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_projects_account ON projects(account_id);
CREATE INDEX IF NOT EXISTS idx_tickets_account ON tickets(account_id);
CREATE INDEX IF NOT EXISTS idx_tickets_project ON tickets(project_id);
CREATE INDEX IF NOT EXISTS idx_tickets_state ON tickets(state);
CREATE INDEX IF NOT EXISTS idx_blocks_account ON blocks(account_id);
