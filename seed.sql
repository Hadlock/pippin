-- Seed data for Pippin
-- The Three Apple Projects

INSERT INTO projects (account_id, key, name) VALUES
  ('demo', 'CART', 'Apple Cart — build cart'),
  ('demo', 'ORCH', 'Apple Orchard — grow & maintain'),
  ('demo', 'STORE', 'Apple Store — sell apples')
ON CONFLICT (account_id, key) DO NOTHING;

-- Sample tickets
INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'Design cart frame', 'Wood vs. metal', 'todo', 'jane'
FROM projects p WHERE p.key='CART' AND p.account_id='demo'
ON CONFLICT DO NOTHING;

INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'Soil testing', 'Check pH & nutrients', 'in_progress', 'lee'
FROM projects p WHERE p.key='ORCH' AND p.account_id='demo'
ON CONFLICT DO NOTHING;

INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'POS setup', 'Pick a simple POS', 'backlog', 'sam'
FROM projects p WHERE p.key='STORE' AND p.account_id='demo'
ON CONFLICT DO NOTHING;

INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'Choose wheel size', '12 vs 14 inch', 'backlog', 'jane'
FROM projects p WHERE p.key='CART' AND p.account_id='demo'
ON CONFLICT DO NOTHING;

INSERT INTO tickets (account_id, project_id, title, body, state, assignee)
SELECT 'demo', p.id, 'Build website', 'Simple storefront', 'todo', 'alex'
FROM projects p WHERE p.key='STORE' AND p.account_id='demo'
ON CONFLICT DO NOTHING;

-- Example blocking relationship: Cart frame must be designed before POS setup
INSERT INTO blocks (blocker_ticket_id, blocked_ticket_id, account_id)
SELECT t1.id, t2.id, 'demo'
FROM tickets t1
JOIN projects p1 ON p1.id = t1.project_id
CROSS JOIN tickets t2
JOIN projects p2 ON p2.id = t2.project_id
WHERE p1.key='CART' AND t1.title='Design cart frame'
  AND p2.key='STORE' AND t2.title='POS setup'
  AND t1.account_id='demo' AND t2.account_id='demo'
ON CONFLICT DO NOTHING;
