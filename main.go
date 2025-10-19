package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

var (
	db  *sql.DB
	tpl *template.Template
	cfg struct {
		Port         string
		DatabaseURL  string
		AccountID    string
		SprintLength int
		SprintEpoch  time.Time
		CozyTheme    string
	}
)

type Project struct {
	ID        int       `json:"id"`
	AccountID string    `json:"account_id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Ticket struct {
	ID         int       `json:"id"`
	AccountID  string    `json:"account_id"`
	ProjectID  int       `json:"project_id"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	State      string    `json:"state"`
	Assignee   string    `json:"assignee"`
	Comments   string    `json:"comments"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ProjectKey string    `json:"project_key,omitempty"`
	BlockedBy  []string  `json:"blocked_by,omitempty"`
}

type Comment struct {
	Timestamp time.Time `json:"timestamp"`
	Text      string    `json:"text"`
}

func main() {
	loadConfig()
	initDB()
	initSchema()
	initTemplate()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/board", http.StatusFound)
	})
	mux.HandleFunc("GET /board", handleBoard)
	mux.HandleFunc("GET /api/projects", handleGetProjects)
	mux.HandleFunc("POST /api/projects", handleCreateProject)
	mux.HandleFunc("DELETE /api/projects/{key}", handleDeleteProject)
	mux.HandleFunc("GET /api/tickets", handleGetTickets)
	mux.HandleFunc("GET /api/tickets/{id}", handleGetTicket)
	mux.HandleFunc("POST /api/tickets", handleCreateTicket)
	mux.HandleFunc("PATCH /api/tickets/{id}", handleUpdateTicket)
	mux.HandleFunc("POST /api/tickets/{id}/move", handleMoveTicket)
	mux.HandleFunc("POST /api/tickets/{id}/comments", handleAddComment)
	mux.HandleFunc("POST /api/tickets/{id}/blocks", handleAddBlock)
	mux.HandleFunc("DELETE /api/tickets/{id}/blocks/{blocked_id}", handleDeleteBlock)
	mux.HandleFunc("GET /api/settings", handleGetSettings)
	mux.HandleFunc("POST /api/settings", handleUpdateSettings)
	mux.HandleFunc("GET /api/export", handleExport)

	log.Printf("🍎 Pippin starting on :%s (account=%s, theme=%s)", cfg.Port, cfg.AccountID, cfg.CozyTheme)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}

func loadConfig() {
	cfg.Port = getEnv("PORT", "8080")
	cfg.DatabaseURL = getEnv("DATABASE_URL", "postgres://pippin:pippin@localhost:5432/pippin?sslmode=disable")
	cfg.AccountID = getEnv("ACCOUNT_ID", "demo")
	cfg.SprintLength = getEnvInt("SPRINT_LENGTH_DAYS", 7)
	cfg.CozyTheme = getEnv("COZY_THEME", "warm")
	epochStr := getEnv("SPRINT_EPOCH", "2025-01-01")
	var err error
	cfg.SprintEpoch, err = time.Parse("2006-01-02", epochStr)
	if err != nil {
		log.Fatalf("invalid SPRINT_EPOCH: %v", err)
	}
}

func initDB() {
	var err error
	db, err = sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}
}

func currentSprintRange(now time.Time) (time.Time, time.Time) {
	days := int(now.Sub(cfg.SprintEpoch).Hours() / 24)
	n := days / cfg.SprintLength
	start := cfg.SprintEpoch.AddDate(0, 0, n*cfg.SprintLength)
	end := start.AddDate(0, 0, cfg.SprintLength)
	return start, end
}

func handleBoard(w http.ResponseWriter, r *http.Request) {
	sprint := r.URL.Query().Get("sprint")
	if sprint == "" {
		sprint = "current"
	}
	projectFilter := r.URL.Query().Get("project")
	if projectFilter == "" {
		projectFilter = "ALL"
	}

	tickets := queryTickets(projectFilter, sprint)
	projects, _ := queryProjects()

	data := struct {
		Theme    string
		Sprint   string
		Project  string
		Projects []Project
		Backlog  []Ticket
		Todo     []Ticket
		InProg   []Ticket
		Done     []Ticket
	}{
		Theme:    cfg.CozyTheme,
		Sprint:   sprint,
		Project:  projectFilter,
		Projects: projects,
	}

	for _, t := range tickets {
		switch t.State {
		case "backlog":
			data.Backlog = append(data.Backlog, t)
		case "todo":
			data.Todo = append(data.Todo, t)
		case "in_progress":
			data.InProg = append(data.InProg, t)
		case "done":
			data.Done = append(data.Done, t)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tpl.Execute(w, data)
}

func handleGetProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := queryProjects()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, projects)
}

func handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM projects WHERE account_id=$1", cfg.AccountID).Scan(&count)
	if count >= 3 {
		writeJSON(w, 400, map[string]string{"error": "project limit reached"})
		return
	}

	var id int
	err := db.QueryRow(`INSERT INTO projects (account_id,key,name) VALUES ($1,$2,$3) RETURNING id`,
		cfg.AccountID, req.Key, req.Name).Scan(&id)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 201, map[string]int{"id": id})
}

func handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		writeJSON(w, 400, map[string]string{"error": "project key required"})
		return
	}

	// Delete project (CASCADE will delete tickets and blocks)
	result, err := db.Exec("DELETE FROM projects WHERE account_id=$1 AND key=$2", cfg.AccountID, key)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeJSON(w, 404, map[string]string{"error": "project not found"})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "deleted"})
}

func handleGetTickets(w http.ResponseWriter, r *http.Request) {
	sprint := r.URL.Query().Get("sprint")
	project := r.URL.Query().Get("project")
	tickets := queryTickets(project, sprint)
	writeJSON(w, 200, tickets)
}

func handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectKey string `json:"project_key"`
		Title      string `json:"title"`
		Body       string `json:"body"`
		Assignee   string `json:"assignee"`
		State      string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}
	if req.State == "" {
		req.State = "backlog"
	}

	var projectID int
	err := db.QueryRow("SELECT id FROM projects WHERE account_id=$1 AND key=$2", cfg.AccountID, req.ProjectKey).Scan(&projectID)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "project not found"})
		return
	}

	var id int
	err = db.QueryRow(`INSERT INTO tickets (account_id,project_id,title,body,state,assignee) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		cfg.AccountID, projectID, req.Title, req.Body, req.State, req.Assignee).Scan(&id)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 201, map[string]int{"id": id})
}

func handleMoveTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Direction string `json:"direction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	var state string
	err := db.QueryRow("SELECT state FROM tickets WHERE id=$1 AND account_id=$2", id, cfg.AccountID).Scan(&state)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "ticket not found"})
		return
	}

	newState := adjacentState(state, req.Direction)
	if newState == "" {
		writeJSON(w, 400, map[string]string{"error": "invalid move"})
		return
	}

	_, err = db.Exec("UPDATE tickets SET state=$1, updated_at=now() WHERE id=$2 AND account_id=$3", newState, id, cfg.AccountID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, map[string]string{"state": newState})
}

func handleAddBlock(w http.ResponseWriter, r *http.Request) {
	blocker := r.PathValue("id")
	var req struct {
		BlockedID int `json:"blocked_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	_, err := db.Exec("INSERT INTO blocks (blocker_ticket_id,blocked_ticket_id,account_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING",
		blocker, req.BlockedID, cfg.AccountID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 201, map[string]string{"status": "ok"})
}

func handleDeleteBlock(w http.ResponseWriter, r *http.Request) {
	blocker := r.PathValue("id")
	blocked := r.PathValue("blocked_id")
	_, err := db.Exec("DELETE FROM blocks WHERE blocker_ticket_id=$1 AND blocked_ticket_id=$2 AND account_id=$3",
		blocker, blocked, cfg.AccountID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func handleGetTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var t Ticket
	err := db.QueryRow(`SELECT t.id, t.account_id, t.project_id, t.title, t.body, t.state, t.assignee, COALESCE(t.comments,''), t.created_at, t.updated_at, p.key
		FROM tickets t JOIN projects p ON t.project_id=p.id 
		WHERE t.id=$1 AND t.account_id=$2`, id, cfg.AccountID).Scan(
		&t.ID, &t.AccountID, &t.ProjectID, &t.Title, &t.Body, &t.State, &t.Assignee, &t.Comments, &t.CreatedAt, &t.UpdatedAt, &t.ProjectKey)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "ticket not found"})
		return
	}
	t.BlockedBy = queryBlockers(t.ID)
	writeJSON(w, 200, t)
}

func handleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Title    string `json:"title"`
		Body     string `json:"body"`
		Assignee string `json:"assignee"`
		State    string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	_, err := db.Exec(`UPDATE tickets SET title=$1, body=$2, assignee=$3, state=$4, updated_at=now() 
		WHERE id=$5 AND account_id=$6`,
		req.Title, req.Body, req.Assignee, req.State, id, cfg.AccountID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 200, map[string]string{"status": "updated"})
}

func handleAddComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	// Get existing comments
	var existing string
	db.QueryRow("SELECT COALESCE(comments,'') FROM tickets WHERE id=$1 AND account_id=$2", id, cfg.AccountID).Scan(&existing)

	// Append new comment with timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	newComment := fmt.Sprintf("[%s] %s", timestamp, req.Comment)

	var updated string
	if existing == "" {
		updated = newComment
	} else {
		updated = existing + "\n" + newComment
	}

	_, err := db.Exec("UPDATE tickets SET comments=$1, updated_at=now() WHERE id=$2 AND account_id=$3", updated, id, cfg.AccountID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, 201, map[string]string{"status": "comment added"})
}

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings := map[string]interface{}{
		"sprint_length_days": cfg.SprintLength,
		"sprint_epoch":       cfg.SprintEpoch.Format("2006-01-02"),
		"cozy_theme":         cfg.CozyTheme,
		"account_id":         cfg.AccountID,
	}
	writeJSON(w, 200, settings)
}

func handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SprintLength int    `json:"sprint_length_days"`
		SprintEpoch  string `json:"sprint_epoch"`
		CozyTheme    string `json:"cozy_theme"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid json"})
		return
	}

	// Note: This only updates runtime config, not env vars
	// In production, you'd want to persist these to a config file or database
	if req.SprintLength > 0 {
		cfg.SprintLength = req.SprintLength
	}
	if req.SprintEpoch != "" {
		if t, err := time.Parse("2006-01-02", req.SprintEpoch); err == nil {
			cfg.SprintEpoch = t
		}
	}
	if req.CozyTheme != "" {
		cfg.CozyTheme = req.CozyTheme
	}

	writeJSON(w, 200, map[string]string{"status": "settings updated (runtime only)"})
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	projects, _ := queryProjects()
	tickets := queryTickets("ALL", "all")

	export := map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"account_id":  cfg.AccountID,
		"projects":    projects,
		"tickets":     tickets,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=pippin-export-%s.json", time.Now().Format("2006-01-02")))
	json.NewEncoder(w).Encode(export)
}

func initSchema() {
	// Add comments column if it doesn't exist
	_, err := db.Exec(`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS comments TEXT DEFAULT ''`)
	if err != nil {
		log.Printf("schema migration warning: %v", err)
	}
}

func queryProjects() ([]Project, error) {
	rows, err := db.Query("SELECT id,account_id,key,name,created_at FROM projects WHERE account_id=$1 ORDER BY created_at", cfg.AccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		rows.Scan(&p.ID, &p.AccountID, &p.Key, &p.Name, &p.CreatedAt)
		projects = append(projects, p)
	}
	return projects, nil
}

func queryTickets(projectFilter, sprint string) []Ticket {
	query := `SELECT t.id, t.account_id, t.project_id, t.title, t.body, t.state, t.assignee, COALESCE(t.comments,''), t.created_at, t.updated_at, p.key
		FROM tickets t JOIN projects p ON t.project_id=p.id WHERE t.account_id=$1`
	args := []interface{}{cfg.AccountID}

	if projectFilter != "" && projectFilter != "ALL" {
		query += " AND p.key=$2"
		args = append(args, projectFilter)
	}

	if sprint == "current" {
		start, end := currentSprintRange(time.Now())
		query += fmt.Sprintf(" AND t.created_at >= '%s' AND t.created_at < '%s'", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}

	query += " ORDER BY t.created_at"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("query tickets: %v", err)
		return nil
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var t Ticket
		rows.Scan(&t.ID, &t.AccountID, &t.ProjectID, &t.Title, &t.Body, &t.State, &t.Assignee, &t.Comments, &t.CreatedAt, &t.UpdatedAt, &t.ProjectKey)
		t.BlockedBy = queryBlockers(t.ID)
		tickets = append(tickets, t)
	}
	return tickets
}

func queryBlockers(ticketID int) []string {
	rows, err := db.Query(`SELECT t.id, p.key FROM blocks b 
		JOIN tickets t ON b.blocker_ticket_id=t.id 
		JOIN projects p ON t.project_id=p.id
		WHERE b.blocked_ticket_id=$1 AND b.account_id=$2`, ticketID, cfg.AccountID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var blockers []string
	for rows.Next() {
		var id int
		var key string
		rows.Scan(&id, &key)
		blockers = append(blockers, fmt.Sprintf("T-%d (%s)", id, key))
	}
	return blockers
}

func adjacentState(current, direction string) string {
	states := []string{"backlog", "todo", "in_progress", "done"}
	idx := -1
	for i, s := range states {
		if s == current {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ""
	}
	if direction == "right" && idx < len(states)-1 {
		return states[idx+1]
	}
	if direction == "left" && idx > 0 {
		return states[idx-1]
	}
	return ""
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func initTemplate() {
	tpl = template.Must(template.New("board").Parse(`<!DOCTYPE html>
<html data-theme="{{.Theme}}">
<head>
<meta charset="utf-8">
<title>🍎 Pippin Board</title>
<style>
:root[data-theme="warm"]{
  --bg:#fff8f0;--panel:#fff2e1;--ink:#3b2e2a;--muted:#8b6f64;
  --todo:#ffe3b0;--doing:#ffd1a1;--done:#c7f9cc;--backlog:#e9ecef;
  --accent:#ffb86b;--border:#e2c6b6;--card:#fffaf5;
}
:root[data-theme="forest"]{
  --bg:#f6fff8;--panel:#e9f5ec;--ink:#1b3a2f;--muted:#446a5a;
  --todo:#dff3e3;--doing:#cbe8d6;--done:#b8e0c2;--backlog:#eaf2ee;
  --accent:#7acb9f;--border:#cfe6d7;--card:#f8fffb;
}
body{background:var(--bg);color:var(--ink);font:14px/1.4 system-ui;margin:0}
header{padding:12px 16px;background:var(--panel);border-bottom:1px solid var(--border);display:flex;gap:12px;align-items:center}
.board{display:grid;grid-template-columns:260px 1fr 1fr 1fr;gap:12px;padding:12px}
.col{background:var(--card);border:1px solid var(--border);border-radius:12px;overflow:hidden}
.col h3{margin:0;padding:8px 10px;background:var(--panel);border-bottom:1px solid var(--border);font-size:14px;display:flex;justify-content:space-between;align-items:center}
.col[data-state="backlog"]{background:var(--backlog)}
.col[data-state="todo"]{background:var(--todo)}
.col[data-state="in_progress"]{background:var(--doing)}
.col[data-state="done"]{background:var(--done)}
.card{margin:8px;border:1px dashed var(--border);border-radius:10px;padding:8px;background:#fff8;backdrop-filter:saturate(120%) blur(2px);cursor:move;user-select:none}
.card:hover{border-style:solid;box-shadow:0 2px 8px rgba(0,0,0,0.1)}
.card.dragging{opacity:0.5;transform:rotate(2deg)}
.col.drag-over{box-shadow:inset 0 0 0 3px var(--accent);transform:scale(1.02);transition:all 0.2s}
.col-content{min-height:100px}
.small{color:var(--muted);font-size:12px}
button,.btn{border:1px solid var(--border);background:var(--accent);color:var(--ink);border-radius:999px;padding:4px 10px;margin-right:6px;cursor:pointer;text-decoration:none;font-size:12px}
button:hover,.btn:hover{opacity:0.8}
.btn-hero{background:var(--accent);font-weight:600;padding:6px 16px;font-size:14px}
.btn-subtle{background:transparent;color:var(--muted);font-size:11px;padding:3px 8px}
.btn-subtle:hover{background:var(--panel)}
.btn-danger{background:#ff6b6b;color:#fff;font-size:11px;padding:3px 8px}
.btn-danger:hover{background:#ff5252}
.badge{background:#ff6b6b;color:#fff;padding:2px 6px;border-radius:4px;font-size:11px;margin-right:4px}
#backlog-col.collapsed .card{display:none}
select{padding:4px 8px;border:1px solid var(--border);background:var(--card);color:var(--ink);border-radius:4px}
.modal{display:none;position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.5);z-index:1000;align-items:center;justify-content:center}
.modal.show{display:flex}
.modal-content{background:var(--card);border:2px solid var(--border);border-radius:16px;padding:24px;max-width:500px;width:90%;box-shadow:0 8px 32px rgba(0,0,0,0.2)}
.modal h2{margin:0 0 16px 0;font-size:18px}
.form-group{margin-bottom:12px}
.form-group label{display:block;margin-bottom:4px;font-size:12px;font-weight:600;color:var(--muted)}
.form-group input,.form-group textarea,.form-group select{width:100%;padding:8px;border:1px solid var(--border);border-radius:8px;background:var(--bg);color:var(--ink);font:14px system-ui;box-sizing:border-box}
.form-group textarea{min-height:80px;resize:vertical}
.form-actions{display:flex;gap:8px;margin-top:16px;justify-content:flex-end}
.hidden{display:none!important}
.search-box{flex:1;max-width:300px;margin-left:auto;position:relative}
.search-box input{width:100%;padding:6px 30px 6px 10px;border:1px solid var(--border);border-radius:8px;background:var(--bg);color:var(--ink);font:13px system-ui;box-sizing:border-box}
.search-box input:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 2px rgba(255,184,107,0.2)}
.search-box .clear-search{position:absolute;right:8px;top:50%;transform:translateY(-50%);background:none;border:none;color:var(--muted);cursor:pointer;padding:0;font-size:16px;display:none}
.search-box input:not(:placeholder-shown) + .clear-search{display:block}
.card.search-hidden{display:none!important}
.ticket-title{cursor:pointer;text-decoration:underline;text-decoration-style:dotted}
.ticket-title:hover{text-decoration-style:solid;color:var(--accent)}
.comments-list{background:var(--bg);border:1px solid var(--border);border-radius:8px;padding:12px;max-height:200px;overflow-y:auto;margin-bottom:12px;font-size:12px;white-space:pre-wrap}
.comment-item{margin-bottom:8px;padding-bottom:8px;border-bottom:1px solid var(--border)}
.comment-item:last-child{border-bottom:none;margin-bottom:0;padding-bottom:0}
.comment-timestamp{color:var(--muted);font-size:11px;font-weight:600}
.ticket-meta{background:var(--panel);border:1px solid var(--border);border-radius:8px;padding:12px;margin-bottom:12px;font-size:12px}
.ticket-meta-item{display:flex;justify-content:space-between;margin-bottom:6px}
.ticket-meta-item:last-child{margin-bottom:0}
.ticket-meta-label{color:var(--muted);font-weight:600}
</style>
</head>
<body>
<header>
  <h1 style="margin:0;font-size:18px">🍎 Pippin</h1>
  <select id="project-filter" onchange="location.href='?sprint={{.Sprint}}&project='+this.value">
    <option value="ALL" {{if eq .Project "ALL"}}selected{{end}}>All Projects</option>
    {{range .Projects}}<option value="{{.Key}}" {{if eq $.Project .Key}}selected{{end}}>{{.Key}}</option>{{end}}
  </select>
  <button class="btn btn-hero" onclick="showAddTicketModal()">+ Add Ticket</button>
  {{if lt (len .Projects) 3}}
  <button class="btn btn-subtle" onclick="showAddProjectModal()">+ Add Project</button>
  {{end}}
  {{if eq .Sprint "current"}}
  <a href="?sprint=all&project={{.Project}}" class="btn">Show All Tickets</a>
  {{else}}
  <a href="?sprint=current&project={{.Project}}" class="btn">Current Sprint</a>
  {{end}}
  {{if and (ge (len .Projects) 3) (ne .Project "ALL")}}
  <button class="btn btn-danger" onclick="confirmDeleteProject('{{.Project}}')">🗑️ Delete Project</button>
  {{end}}
  <div class="search-box">
    <input type="text" id="search-input" placeholder="🔍 Search tickets..." autocomplete="off">
    <button class="clear-search" onclick="clearSearch()">✕</button>
  </div>
  <button class="btn btn-subtle" onclick="showSettingsModal()" title="Settings">⚙️</button>
</header>
<div class="board">
  <div class="col" data-state="backlog" id="backlog-col">
    <h3>
      <span>📋 Backlog ({{len .Backlog}})</span>
      <button onclick="toggleBacklog()" style="font-size:10px">Toggle</button>
    </h3>
    <div class="col-content">
    {{range .Backlog}}
    <div class="card" draggable="true" data-id="{{.ID}}" data-state="backlog" data-title="{{.Title}}" data-project="{{.ProjectKey}}" data-assignee="{{.Assignee}}">
      <strong>{{.ProjectKey}}-{{.ID}}</strong> <span class="ticket-title" onclick="showTicketView({{.ID}})">{{.Title}}</span>
      <div class="small">{{.Assignee}}</div>
      {{range .BlockedBy}}<span class="badge">⚠ {{.}}</span>{{end}}
      <div style="margin-top:6px">
        <button onclick="move({{.ID}},'right')">→</button>
      </div>
    </div>
    {{end}}
    </div>
  </div>
  
  <div class="col" data-state="todo">
    <h3>📝 Todo ({{len .Todo}})</h3>
    <div class="col-content">
    {{range .Todo}}
    <div class="card" draggable="true" data-id="{{.ID}}" data-state="todo" data-title="{{.Title}}" data-project="{{.ProjectKey}}" data-assignee="{{.Assignee}}">
      <strong>{{.ProjectKey}}-{{.ID}}</strong> <span class="ticket-title" onclick="showTicketView({{.ID}})">{{.Title}}</span>
      <div class="small">{{.Assignee}}</div>
      {{range .BlockedBy}}<span class="badge">⚠ {{.}}</span>{{end}}
      <div style="margin-top:6px">
        <button onclick="move({{.ID}},'left')">←</button>
        <button onclick="move({{.ID}},'right')">→</button>
      </div>
    </div>
    {{end}}
    </div>
  </div>
  
  <div class="col" data-state="in_progress">
    <h3>🔧 In Progress ({{len .InProg}})</h3>
    <div class="col-content">
    {{range .InProg}}
    <div class="card" draggable="true" data-id="{{.ID}}" data-state="in_progress" data-title="{{.Title}}" data-project="{{.ProjectKey}}" data-assignee="{{.Assignee}}">
      <strong>{{.ProjectKey}}-{{.ID}}</strong> <span class="ticket-title" onclick="showTicketView({{.ID}})">{{.Title}}</span>
      <div class="small">{{.Assignee}}</div>
      {{range .BlockedBy}}<span class="badge">⚠ {{.}}</span>{{end}}
      <div style="margin-top:6px">
        <button onclick="move({{.ID}},'left')">←</button>
        <button onclick="move({{.ID}},'right')">→</button>
      </div>
    </div>
    {{end}}
    </div>
  </div>
  
  <div class="col" data-state="done">
    <h3>✅ Done ({{len .Done}})</h3>
    <div class="col-content">
    {{range .Done}}
    <div class="card" draggable="true" data-id="{{.ID}}" data-state="done" data-title="{{.Title}}" data-project="{{.ProjectKey}}" data-assignee="{{.Assignee}}">
      <strong>{{.ProjectKey}}-{{.ID}}</strong> <span class="ticket-title" onclick="showTicketView({{.ID}})">{{.Title}}</span>
      <div class="small">{{.Assignee}}</div>
      <div style="margin-top:6px">
        <button onclick="move({{.ID}},'left')">←</button>
      </div>
    </div>
    {{end}}
    </div>
  </div>
</div>

<!-- Add Ticket Modal -->
<div id="ticket-modal" class="modal">
  <div class="modal-content">
    <h2>Add New Ticket</h2>
    <form id="ticket-form" onsubmit="submitTicket(event)">
      <div class="form-group">
        <label>Project *</label>
        <select id="ticket-project" required>
          <option value="">Select a project...</option>
          {{range .Projects}}<option value="{{.Key}}">{{.Key}} - {{.Name}}</option>{{end}}
        </select>
      </div>
      <div class="form-group">
        <label>Title *</label>
        <input type="text" id="ticket-title" required placeholder="Brief description">
      </div>
      <div class="form-group">
        <label>Description</label>
        <textarea id="ticket-body" placeholder="Additional details..."></textarea>
      </div>
      <div class="form-group">
        <label>Assignee</label>
        <input type="text" id="ticket-assignee" placeholder="Username">
      </div>
      <div class="form-group">
        <label>Initial State</label>
        <select id="ticket-state">
          <option value="backlog">Backlog</option>
          <option value="todo">Todo</option>
          <option value="in_progress">In Progress</option>
          <option value="done">Done</option>
        </select>
      </div>
      <div class="form-actions">
        <button type="button" class="btn btn-subtle" onclick="hideAddTicketModal()">Cancel</button>
        <button type="submit" class="btn btn-hero">Create Ticket</button>
      </div>
    </form>
  </div>
</div>

<!-- Add Project Modal -->
<div id="project-modal" class="modal">
  <div class="modal-content">
    <h2>Add New Project</h2>
    <form id="project-form" onsubmit="submitProject(event)">
      <div class="form-group">
        <label>Project Key * (e.g., CART, STORE)</label>
        <input type="text" id="project-key" required pattern="[A-Za-z0-9]+" placeholder="PROJ" maxlength="10" style="text-transform:uppercase">
      </div>
      <div class="form-group">
        <label>Project Name *</label>
        <input type="text" id="project-name" required placeholder="Full project name">
      </div>
      <div class="form-actions">
        <button type="button" class="btn btn-subtle" onclick="hideAddProjectModal()">Cancel</button>
        <button type="submit" class="btn btn-hero">Create Project</button>
      </div>
    </form>
  </div>
</div>

<!-- Settings Modal -->
<div id="settings-modal" class="modal">
  <div class="modal-content">
    <h2>⚙️ Settings</h2>
    <div style="margin-bottom:20px">
      <h3 style="margin:0 0 8px 0;font-size:14px">Sprint Configuration</h3>
      <div class="form-group">
        <label>Sprint Length (days)</label>
        <select id="settings-sprint-length">
          <option value="7">7 days (weekly)</option>
          <option value="14">14 days (bi-weekly)</option>
        </select>
      </div>
      <div class="form-group">
        <label>Sprint Epoch (start date)</label>
        <input type="date" id="settings-sprint-epoch">
      </div>
      <div class="form-group">
        <label>Theme</label>
        <select id="settings-theme">
          <option value="warm">Warm (peachy)</option>
          <option value="forest">Forest (green)</option>
        </select>
      </div>
      <button onclick="saveSettings()" class="btn btn-hero">Save Settings</button>
      <p style="font-size:11px;color:var(--muted);margin-top:8px">
        ⚠️ Settings only affect current session (not persisted to env vars)
      </p>
    </div>
    <div style="border-top:1px solid var(--border);padding-top:16px">
      <h3 style="margin:0 0 8px 0;font-size:14px">Export Data</h3>
      <p style="font-size:12px;margin-bottom:12px;color:var(--muted)">
        Download all projects and tickets as JSON
      </p>
      <button onclick="exportData()" class="btn">📥 Export to JSON</button>
    </div>
    <div class="form-actions" style="margin-top:20px">
      <button class="btn btn-subtle" onclick="hideSettingsModal()">Close</button>
    </div>
  </div>
</div>

<!-- Ticket View Modal -->
<div id="ticket-view-modal" class="modal">
  <div class="modal-content" style="max-width:600px">
    <h2 id="ticket-view-title">Loading...</h2>
    <div class="ticket-meta" id="ticket-view-meta"></div>
    
    <div class="form-group">
      <label>Title *</label>
      <input type="text" id="ticket-view-title-input" required>
    </div>
    <div class="form-group">
      <label>Description</label>
      <textarea id="ticket-view-body"></textarea>
    </div>
    <div class="form-group">
      <label>Assignee</label>
      <input type="text" id="ticket-view-assignee">
    </div>
    <div class="form-group">
      <label>State</label>
      <select id="ticket-view-state">
        <option value="backlog">Backlog</option>
        <option value="todo">Todo</option>
        <option value="in_progress">In Progress</option>
        <option value="done">Done</option>
      </select>
    </div>
    
    <div style="border-top:1px solid var(--border);padding-top:16px;margin-top:16px">
      <h3 style="margin:0 0 8px 0;font-size:14px">Comments</h3>
      <div class="comments-list" id="ticket-view-comments">
        <div style="color:var(--muted);font-style:italic">No comments yet</div>
      </div>
      <div class="form-group">
        <textarea id="ticket-view-new-comment" placeholder="Add a comment..." rows="2"></textarea>
      </div>
      <button onclick="saveComment()" class="btn">💬 Save Comment</button>
    </div>
    
    <div class="form-actions" style="margin-top:20px">
      <button class="btn btn-subtle" onclick="hideTicketView()">Cancel</button>
      <button class="btn btn-hero" onclick="saveTicketUpdates()">💾 Save Changes</button>
    </div>
  </div>
</div>

<script>
const stateOrder = ['backlog', 'todo', 'in_progress', 'done'];
let draggedCard = null;

function move(id,dir){
  fetch('/api/tickets/'+id+'/move',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({direction:dir})})
  .then(r=>r.json()).then(()=>location.reload());
}

function toggleBacklog(){
  document.getElementById('backlog-col').classList.toggle('collapsed');
}

// Modal functions
function showAddTicketModal() {
  const modal = document.getElementById('ticket-modal');
  const projectSelect = document.getElementById('ticket-project');
  const filter = document.getElementById('project-filter').value;
  
  // Pre-select project from filter if not ALL
  if (filter !== 'ALL') {
    projectSelect.value = filter;
  }
  
  modal.classList.add('show');
}

function hideAddTicketModal() {
  document.getElementById('ticket-modal').classList.remove('show');
  document.getElementById('ticket-form').reset();
}

function showAddProjectModal() {
  document.getElementById('project-modal').classList.add('show');
}

function hideAddProjectModal() {
  document.getElementById('project-modal').classList.remove('show');
  document.getElementById('project-form').reset();
}

function submitTicket(e) {
  e.preventDefault();
  const data = {
    project_key: document.getElementById('ticket-project').value,
    title: document.getElementById('ticket-title').value,
    body: document.getElementById('ticket-body').value,
    assignee: document.getElementById('ticket-assignee').value,
    state: document.getElementById('ticket-state').value
  };
  
  fetch('/api/tickets', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(data)
  })
  .then(r => r.json())
  .then(result => {
    if (result.error) {
      alert('Error: ' + result.error);
    } else {
      location.reload();
    }
  })
  .catch(err => alert('Error creating ticket: ' + err));
}

function submitProject(e) {
  e.preventDefault();
  const data = {
    key: document.getElementById('project-key').value.toUpperCase(),
    name: document.getElementById('project-name').value
  };
  
  fetch('/api/projects', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(data)
  })
  .then(r => r.json())
  .then(result => {
    if (result.error) {
      alert('Error: ' + result.error);
    } else {
      location.reload();
    }
  })
  .catch(err => alert('Error creating project: ' + err));
}

// Close modals on outside click
document.addEventListener('click', (e) => {
  if (e.target.classList.contains('modal')) {
    e.target.classList.remove('show');
  }
});

// Close modals on Escape key
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    document.querySelectorAll('.modal').forEach(m => m.classList.remove('show'));
  }
});

// Delete project with confirmation
function confirmDeleteProject(projectKey) {
  const confirmed = confirm(
    'Delete project ' + projectKey + '?\n\n' +
    'This will permanently delete the project and ALL its tickets.\n\n' +
    'This action cannot be undone.'
  );
  
  if (!confirmed) return;
  
  fetch('/api/projects/' + projectKey, {
    method: 'DELETE',
    headers: {'Content-Type': 'application/json'}
  })
  .then(r => r.json())
  .then(data => {
    if (data.error) {
      alert('Error: ' + data.error);
    } else {
      // Redirect to all projects view after deletion
      window.location.href = '/board?sprint=' + (new URLSearchParams(window.location.search).get('sprint') || 'current') + '&project=ALL';
    }
  })
  .catch(err => alert('Error deleting project: ' + err));
}

// Fuzzy search functionality (fzf-inspired)
function fuzzyMatch(needle, haystack) {
  needle = needle.toLowerCase();
  haystack = haystack.toLowerCase();
  let nIdx = 0;
  let hIdx = 0;
  let score = 0;
  let lastMatchIdx = -1;
  
  while (nIdx < needle.length && hIdx < haystack.length) {
    if (needle[nIdx] === haystack[hIdx]) {
      // Bonus for consecutive matches
      if (lastMatchIdx === hIdx - 1) score += 2;
      else score += 1;
      lastMatchIdx = hIdx;
      nIdx++;
    }
    hIdx++;
  }
  
  return nIdx === needle.length ? score : 0;
}

function searchTickets() {
  const query = document.getElementById('search-input').value.trim();
  const projectFilter = document.getElementById('project-filter').value;
  const cards = document.querySelectorAll('.card');
  
  if (query === '') {
    // Show all cards
    cards.forEach(card => card.classList.remove('search-hidden'));
    updateColumnCounts();
    return;
  }
  
  cards.forEach(card => {
    const title = card.dataset.title || '';
    const project = card.dataset.project || '';
    const assignee = card.dataset.assignee || '';
    const id = card.dataset.id || '';
    
    // Respect project filter
    if (projectFilter !== 'ALL' && project !== projectFilter) {
      card.classList.add('search-hidden');
      return;
    }
    
    // Search in title, project key, assignee, and ID
    const searchText = project + '-' + id + ' ' + title + ' ' + assignee;
    const score = fuzzyMatch(query, searchText);
    
    if (score > 0) {
      card.classList.remove('search-hidden');
    } else {
      card.classList.add('search-hidden');
    }
  });
  
  updateColumnCounts();
}

function clearSearch() {
  document.getElementById('search-input').value = '';
  searchTickets();
  document.getElementById('search-input').focus();
}

function updateColumnCounts() {
  ['backlog', 'todo', 'in_progress', 'done'].forEach(state => {
    const col = document.querySelector('.col[data-state="' + state + '"]');
    if (!col) return;
    const visible = col.querySelectorAll('.card:not(.search-hidden)').length;
    const header = col.querySelector('h3 span');
    if (header) {
      const text = header.textContent.split('(')[0].trim();
      header.textContent = text + ' (' + visible + ')';
    }
  });
}

// Real-time search
document.addEventListener('DOMContentLoaded', () => {
  const searchInput = document.getElementById('search-input');
  if (searchInput) {
    searchInput.addEventListener('input', searchTickets);
    // Focus search on / key
    document.addEventListener('keydown', (e) => {
      if (e.key === '/' && !e.target.matches('input,textarea')) {
        e.preventDefault();
        searchInput.focus();
      }
    });
  }
});

function isAdjacentState(from, to) {
  const fromIdx = stateOrder.indexOf(from);
  const toIdx = stateOrder.indexOf(to);
  return Math.abs(fromIdx - toIdx) === 1;
}

function getDirection(from, to) {
  const fromIdx = stateOrder.indexOf(from);
  const toIdx = stateOrder.indexOf(to);
  return toIdx > fromIdx ? 'right' : 'left';
}

document.addEventListener('DOMContentLoaded',()=>{
  // Don't auto-collapse backlog anymore - let users see all their tickets
  
  // Add drag event listeners to all cards
  document.querySelectorAll('.card').forEach(card => {
    card.addEventListener('dragstart', (e) => {
      draggedCard = card;
      card.classList.add('dragging');
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/html', card.innerHTML);
    });
    
    card.addEventListener('dragend', (e) => {
      card.classList.remove('dragging');
      document.querySelectorAll('.col').forEach(col => {
        col.classList.remove('drag-over');
      });
    });
  });
  
  // Add drop zone listeners to all columns
  document.querySelectorAll('.col').forEach(col => {
    col.addEventListener('dragover', (e) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      col.classList.add('drag-over');
    });
    
    col.addEventListener('dragleave', (e) => {
      if (e.target === col) {
        col.classList.remove('drag-over');
      }
    });
    
    col.addEventListener('drop', (e) => {
      e.preventDefault();
      col.classList.remove('drag-over');
      
      if (!draggedCard) return;
      
      const ticketId = draggedCard.dataset.id;
      const fromState = draggedCard.dataset.state;
      const toState = col.dataset.state;
      
      // Check if move is valid (adjacent states only)
      if (!isAdjacentState(fromState, toState)) {
        alert('Can only move to adjacent states! (' + fromState + ' → ' + toState + ' not allowed)');
        return;
      }
      
      const direction = getDirection(fromState, toState);
      
      // Make the API call
      fetch('/api/tickets/'+ticketId+'/move', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({direction: direction})
      })
      .then(r => r.json())
      .then(data => {
        if (data.error) {
          alert('Error: ' + data.error);
        } else {
          location.reload();
        }
      })
      .catch(err => {
        alert('Error moving ticket: ' + err);
      });
    });
  });
});

// Settings Modal functions
function showSettingsModal() {
  fetch('/api/settings')
    .then(r => r.json())
    .then(data => {
      document.getElementById('settings-sprint-length').value = data.sprint_length_days;
      document.getElementById('settings-sprint-epoch').value = data.sprint_epoch;
      document.getElementById('settings-theme').value = data.cozy_theme;
      document.getElementById('settings-modal').classList.add('show');
    });
}

function hideSettingsModal() {
  document.getElementById('settings-modal').classList.remove('show');
}

function saveSettings() {
  const sprintLength = parseInt(document.getElementById('settings-sprint-length').value);
  const sprintEpoch = document.getElementById('settings-sprint-epoch').value;
  const theme = document.getElementById('settings-theme').value;
  
  fetch('/api/settings', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      sprint_length_days: sprintLength,
      sprint_epoch: sprintEpoch,
      cozy_theme: theme
    })
  })
  .then(r => r.json())
  .then(data => {
    alert('Settings updated! Refresh page to see changes.');
    hideSettingsModal();
  })
  .catch(err => alert('Error saving settings: ' + err));
}

function exportData() {
  window.location.href = '/api/export';
}

// Ticket View Modal functions
let currentTicketId = null;

function showTicketView(ticketId) {
  currentTicketId = ticketId;
  
  fetch('/api/tickets/' + ticketId)
    .then(r => r.json())
    .then(ticket => {
      document.getElementById('ticket-view-title').textContent = ticket.project_key + '-' + ticket.id;
      
      // Populate metadata
      const meta = document.getElementById('ticket-view-meta');
      meta.innerHTML = 
        '<div class="ticket-meta-item"><span class="ticket-meta-label">Project:</span><span>' + ticket.project_key + '</span></div>' +
        '<div class="ticket-meta-item"><span class="ticket-meta-label">Created:</span><span>' + new Date(ticket.created_at).toLocaleString() + '</span></div>' +
        '<div class="ticket-meta-item"><span class="ticket-meta-label">Updated:</span><span>' + new Date(ticket.updated_at).toLocaleString() + '</span></div>' +
        (ticket.blocked_by && ticket.blocked_by.length > 0 ? 
          '<div class="ticket-meta-item"><span class="ticket-meta-label">Blocked by:</span><span>' + ticket.blocked_by.join(', ') + '</span></div>' : '');
      
      // Populate form fields
      document.getElementById('ticket-view-title-input').value = ticket.title;
      document.getElementById('ticket-view-body').value = ticket.body || '';
      document.getElementById('ticket-view-assignee').value = ticket.assignee || '';
      document.getElementById('ticket-view-state').value = ticket.state;
      
      // Display comments
      const commentsDiv = document.getElementById('ticket-view-comments');
      if (ticket.comments && ticket.comments.trim() !== '') {
        const comments = ticket.comments.split('\\n');
        commentsDiv.innerHTML = comments.map(c => {
          const match = c.match(/^\\[(.+?)\\] (.+)$/);
          if (match) {
            return '<div class="comment-item"><div class="comment-timestamp">' + match[1] + '</div><div>' + match[2] + '</div></div>';
          }
          return '<div class="comment-item">' + c + '</div>';
        }).join('');
      } else {
        commentsDiv.innerHTML = '<div style="color:var(--muted);font-style:italic">No comments yet</div>';
      }
      
      document.getElementById('ticket-view-new-comment').value = '';
      document.getElementById('ticket-view-modal').classList.add('show');
    })
    .catch(err => alert('Error loading ticket: ' + err));
}

function hideTicketView() {
  document.getElementById('ticket-view-modal').classList.remove('show');
  currentTicketId = null;
}

function saveComment() {
  const comment = document.getElementById('ticket-view-new-comment').value.trim();
  if (!comment) {
    alert('Please enter a comment');
    return;
  }
  
  fetch('/api/tickets/' + currentTicketId + '/comments', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({comment: comment})
  })
  .then(r => r.json())
  .then(data => {
    if (data.error) {
      alert('Error: ' + data.error);
    } else {
      // Reload ticket to show new comment
      showTicketView(currentTicketId);
    }
  })
  .catch(err => alert('Error saving comment: ' + err));
}

function saveTicketUpdates() {
  const title = document.getElementById('ticket-view-title-input').value.trim();
  const body = document.getElementById('ticket-view-body').value;
  const assignee = document.getElementById('ticket-view-assignee').value;
  const state = document.getElementById('ticket-view-state').value;
  
  if (!title) {
    alert('Title is required');
    return;
  }
  
  fetch('/api/tickets/' + currentTicketId, {
    method: 'PATCH',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      title: title,
      body: body,
      assignee: assignee,
      state: state
    })
  })
  .then(r => r.json())
  .then(data => {
    if (data.error) {
      alert('Error: ' + data.error);
    } else {
      hideTicketView();
      location.reload();
    }
  })
  .catch(err => alert('Error saving changes: ' + err));
}

// Close modals on Escape key or click outside
document.addEventListener('click', (e) => {
  if (e.target.classList.contains('modal')) {
    e.target.classList.remove('show');
  }
});

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    document.querySelectorAll('.modal').forEach(m => m.classList.remove('show'));
  }
});
</script>
</body>
</html>`))
}
