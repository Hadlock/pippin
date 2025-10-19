Perfect — PostgREST makes a great fast mock backend.
Here’s the quickest dev setup so your Go “Pippin” agent can develop and test against it.

---

### 🧱 1. Start a PostgREST + Postgres combo

```bash
# create a network for isolation
docker network create pippin-net

# postgres container
docker run -d --name pippin-db \
  --network pippin-net \
  -e POSTGRES_USER=pippin \
  -e POSTGRES_PASSWORD=pippin \
  -e POSTGRES_DB=pippin \
  -p 5432:5432 \
  postgres:17

# wait a few seconds for pg to init
sleep 5

# postgrest container
docker run -d --name pippin-api \
  --network pippin-net \
  -e PGRST_DB_URI="postgres://pippin:pippin@pippin-db:5432/pippin" \
  -e PGRST_DB_SCHEMA="public" \
  -e PGRST_DB_ANON_ROLE="pippin" \
  -e PGRST_OPENAPI_SERVER_PROXY_URI="http://localhost:8081" \
  -p 8081:3000 \
  postgrest/postgrest:latest
```

That exposes:

* Postgres on `localhost:5432`
* PostgREST API on `http://localhost:8081`

---

### 🧭 2. Initialize schema

Attach to the DB:

```bash
docker exec -it pippin-db psql -U pippin -d pippin
```

Then paste in:

```sql
CREATE TABLE projects (
  id SERIAL PRIMARY KEY,
  account_id TEXT NOT NULL,
  key TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE tickets (
  id SERIAL PRIMARY KEY,
  account_id TEXT NOT NULL,
  project_id INTEGER REFERENCES projects(id),
  title TEXT NOT NULL,
  body TEXT,
  state TEXT CHECK (state IN ('backlog','todo','in_progress','done')),
  assignee TEXT,
  created_at TIMESTAMP DEFAULT now(),
  updated_at TIMESTAMP DEFAULT now()
);

CREATE TABLE blocks (
  blocker_ticket_id INTEGER REFERENCES tickets(id),
  blocked_ticket_id INTEGER REFERENCES tickets(id),
  account_id TEXT NOT NULL,
  PRIMARY KEY (blocker_ticket_id, blocked_ticket_id)
);
```

Add some seed data:

```sql
INSERT INTO projects (account_id,key,name) VALUES
 ('demo','CART','Apple Cart — build cart'),
 ('demo','ORCH','Apple Orchard — grow & maintain'),
 ('demo','STORE','Apple Store — sell apples');
```

Now you can hit:

```
curl http://localhost:8081/projects
curl http://localhost:8081/tickets
```

You’ll get JSON responses directly from PostgREST.

---

### 🧩 3. Point your Go app at it

In `Pippin`:

```bash
export DATABASE_URL="postgres://pippin:pippin@localhost:5432/pippin?sslmode=disable"
export API_BASE="http://localhost:8081"
```

You can have your agent choose either:

* **Direct SQL** (through `DATABASE_URL`)
* **REST** (through `API_BASE`, ideal for mocking while coding frontend)

---

### 🧰 4. Optional pgAdmin or TablePlus connection

If you want to watch data live:

```
Host: localhost
Port: 5432
User: pippin
Pass: pippin
DB: pippin
```

---

### ✅ 5. Verify PostgREST is alive

```bash
curl http://localhost:8081/
# → returns the OpenAPI spec JSON
```

---

Would you like me to give you a **minimal PostgREST “schema.sql”** tailored for the Pippin app (including views and role policies for safe anon reads/writes)? That’ll make the PostgREST instance act like a fully RESTful backend your Go frontend can immediately consume.
