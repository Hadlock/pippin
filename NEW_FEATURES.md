# 🎉 Pippin New Features

## Overview

Three major feature additions have been implemented:

1. **⚙️ Settings Modal** - Configure sprint settings and export data
2. **💬 Comments System** - Add comments to tickets
3. **🔍 Ticket View Modal** - Detailed ticket view with full editing capabilities

---

## 1. ⚙️ Settings Modal

### Access
Click the **⚙️** gear icon button in the header (top-right corner)

### Features

#### Sprint Configuration
- **Sprint Length**: Choose between 7 days (weekly) or 14 days (bi-weekly)
- **Sprint Epoch**: Set the sprint start date (ISO format: YYYY-MM-DD)
- **Theme**: Switch between Warm (peachy) and Forest (green) themes

#### Export Data
- **Export to JSON**: Download all projects and tickets as a JSON file
- File naming: `pippin-export-YYYY-MM-DD.json`
- Contains:
  - Export timestamp
  - Account ID
  - All projects with metadata
  - All tickets with comments, states, timestamps, etc.

### API Endpoints

**Get Settings**
```bash
GET /api/settings
```

Response:
```json
{
  "sprint_length_days": 7,
  "sprint_epoch": "2025-01-01",
  "cozy_theme": "warm",
  "account_id": "demo"
}
```

**Update Settings**
```bash
POST /api/settings
Content-Type: application/json

{
  "sprint_length_days": 14,
  "sprint_epoch": "2025-01-01",
  "cozy_theme": "forest"
}
```

**Export Data**
```bash
GET /api/export
```

Downloads JSON file with all data.

### Notes
- ⚠️ Settings changes only affect the current runtime session
- Changes are not persisted to environment variables
- For permanent changes, update your `.env` file or environment variables
- Page refresh may be required to see theme changes

---

## 2. 💬 Comments System

### Database Schema

A new `comments` column has been added to the `tickets` table:

```sql
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS comments TEXT DEFAULT '';
```

- **Type**: TEXT (unlimited length)
- **Format**: Newline-separated comments with timestamps
- **Example**:
  ```
  [2025-10-18 18:30:00] Initial comment here
  [2025-10-18 18:45:00] Another comment added later
  ```

### How to Use

1. **View Comments**: Click on any ticket title to open the Ticket View Modal
2. **Add Comment**: 
   - Scroll to the "Comments" section
   - Type your comment in the text area
   - Click **"💬 Save Comment"**
3. **Comment Format**: Each comment is automatically timestamped

### API Endpoint

**Add Comment to Ticket**
```bash
POST /api/tickets/{id}/comments
Content-Type: application/json

{
  "comment": "This is my comment text"
}
```

Response:
```json
{
  "status": "comment added"
}
```

### Features
- **Timestamps**: Automatically added in format `[YYYY-MM-DD HH:MM:SS]`
- **Append-only**: Comments are appended, never overwritten
- **No deletion**: Once added, comments persist (edit ticket directly in DB if needed)
- **Real-time display**: Comments appear immediately after saving

---

## 3. 🔍 Ticket View Modal

### Access
Click on **any ticket title** from the kanban board (underlined text with dotted decoration)

### Features

#### Metadata Display (Read-Only)
- **Project Key**: e.g., CART, ORCH, STORE
- **Created**: Original creation timestamp
- **Updated**: Last modification timestamp  
- **Blocked By**: List of blocking tickets (if any)

#### Editable Fields
- **Title** (required)
- **Description** (body text)
- **Assignee** (username)
- **State** (dropdown: Backlog, Todo, In Progress, Done)

#### Comments Section
- **View All Comments**: Chronologically ordered with timestamps
- **Add New Comment**: Text area for new comments
- **Save Comment**: Button to add comment (reloads ticket data)

#### Actions
- **💬 Save Comment**: Adds new comment without closing modal
- **💾 Save Changes**: Updates ticket fields and closes modal
- **Cancel**: Closes modal without saving changes

### Visual Design
- **Clickable Titles**: Dotted underline, changes to solid on hover
- **Modal Size**: Larger than other modals (max-width: 600px)
- **Comments List**: 
  - Scrollable (max-height: 200px)
  - Each comment in its own styled box
  - Timestamp in bold, muted color
  - Comment text below timestamp

### API Endpoints

**Get Single Ticket**
```bash
GET /api/tickets/{id}
```

Response:
```json
{
  "id": 1,
  "account_id": "demo",
  "project_id": 1,
  "project_key": "CART",
  "title": "Design cart frame",
  "body": "Wood vs metal decision",
  "state": "todo",
  "assignee": "jane",
  "comments": "[2025-10-18 18:30:00] Initial comment",
  "created_at": "2025-10-18T18:00:00Z",
  "updated_at": "2025-10-18T18:30:00Z",
  "blocked_by": ["T-3 (ORCH)"]
}
```

**Update Ticket**
```bash
PATCH /api/tickets/{id}
Content-Type: application/json

{
  "title": "Updated title",
  "body": "Updated description",
  "assignee": "bob",
  "state": "in_progress"
}
```

Response:
```json
{
  "status": "updated"
}
```

### Notes
- **Created/Updated timestamps**: Cannot be manually edited (auto-managed by DB)
- **State changes**: Can change to any state (not restricted to adjacent states in modal)
- **Real-time**: Changes appear immediately after page reload
- **Validation**: Title is required, other fields are optional

---

## Technical Implementation

### Database Migration
The `initSchema()` function runs on startup and adds the `comments` column if it doesn't exist:

```go
func initSchema() {
    _, err := db.Exec(`ALTER TABLE tickets ADD COLUMN IF NOT EXISTS comments TEXT DEFAULT ''`)
    if err != nil {
        log.Printf("schema migration warning: %v", err)
    }
}
```

### New Handler Functions

1. **`handleGetTicket`**: Fetch single ticket details
2. **`handleUpdateTicket`**: Update ticket fields (PATCH)
3. **`handleAddComment`**: Append comment to ticket
4. **`handleGetSettings`**: Return current settings
5. **`handleUpdateSettings`**: Update runtime settings
6. **`handleExport`**: Generate JSON export

### JavaScript Functions

#### Settings Modal
- `showSettingsModal()` - Fetch and display current settings
- `hideSettingsModal()` - Close settings modal
- `saveSettings()` - POST updated settings to API
- `exportData()` - Trigger JSON download

#### Ticket View Modal
- `showTicketView(ticketId)` - Fetch and display ticket details
- `hideTicketView()` - Close ticket view modal
- `saveComment()` - POST new comment to API
- `saveTicketUpdates()` - PATCH ticket changes to API

### CSS Additions

```css
.ticket-title {
  cursor: pointer;
  text-decoration: underline;
  text-decoration-style: dotted;
}
.ticket-title:hover {
  text-decoration-style: solid;
  color: var(--accent);
}
.comments-list {
  /* Styled scrollable container */
}
.comment-item {
  /* Individual comment styling */
}
.ticket-meta {
  /* Metadata display box */
}
```

---

## Usage Examples

### Example 1: Export All Data

1. Click **⚙️** in header
2. Scroll to "Export Data" section
3. Click **"📥 Export to JSON"**
4. File downloads: `pippin-export-2025-10-18.json`

```json
{
  "exported_at": "2025-10-18T18:45:00Z",
  "account_id": "demo",
  "projects": [
    {"id": 1, "key": "CART", "name": "Apple Cart"},
    {"id": 2, "key": "ORCH", "name": "Apple Orchard"}
  ],
  "tickets": [
    {
      "id": 1,
      "title": "Design cart frame",
      "state": "todo",
      "comments": "[2025-10-18 18:30:00] Decided on metal frame",
      ...
    }
  ]
}
```

### Example 2: Add Comment to Ticket

1. Click on ticket title "Design cart frame"
2. Ticket View Modal opens
3. Scroll to Comments section
4. Type: "Checked with supplier - metal is cheaper"
5. Click **"💬 Save Comment"**
6. Comment appears with timestamp: `[2025-10-18 18:50:00] Checked with supplier - metal is cheaper`

### Example 3: Update Ticket Details

1. Click on ticket title
2. Change Title to "Build cart frame with metal"
3. Change Assignee to "bob"
4. Change State to "in_progress"
5. Click **"💾 Save Changes"**
6. Modal closes, page refreshes
7. Ticket now shows in "In Progress" column with new title

### Example 4: Change Sprint Settings

1. Click **⚙️** in header
2. Change Sprint Length to "14 days (bi-weekly)"
3. Change Sprint Epoch to "2025-01-06"
4. Click **"Save Settings"**
5. Alert: "Settings updated! Refresh page to see changes."
6. Refresh page
7. Sprint filtering now uses 14-day cycles starting from Jan 6

---

## Keyboard Shortcuts

- **`/`** - Focus search box (existing)
- **`Escape`** - Close any open modal (new + existing)
- **Click outside modal** - Close modal (new + existing)

---

## Testing

### Test Settings Modal

```bash
# Get current settings
curl http://localhost:8080/api/settings

# Update settings
curl -X POST http://localhost:8080/api/settings \
  -H 'Content-Type: application/json' \
  -d '{"sprint_length_days":14,"sprint_epoch":"2025-01-06","cozy_theme":"forest"}'

# Export data
curl http://localhost:8080/api/export -o export.json
```

### Test Comments

```bash
# Get ticket details
curl http://localhost:8080/api/tickets/1

# Add comment
curl -X POST http://localhost:8080/api/tickets/1/comments \
  -H 'Content-Type: application/json' \
  -d '{"comment":"This is a test comment"}'

# Verify comment was added
curl http://localhost:8080/api/tickets/1 | jq '.comments'
```

### Test Ticket Updates

```bash
# Update ticket
curl -X PATCH http://localhost:8080/api/tickets/1 \
  -H 'Content-Type: application/json' \
  -d '{
    "title":"Updated title",
    "body":"New description",
    "assignee":"alice",
    "state":"in_progress"
  }'

# Verify changes
curl http://localhost:8080/api/tickets/1 | jq '{title,state,assignee}'
```

---

## Migration Notes

### Existing Installations

The `comments` column is automatically added on first run after upgrading:

1. Stop Pippin: `pkill -f pippin`
2. Build new version: `go build -o pippin main.go`
3. Run: `./pippin`
4. On startup, the `initSchema()` function adds the column

### Existing Tickets

- All existing tickets will have empty `comments` field (`''`)
- No data loss - all existing fields remain unchanged
- Comments can be added immediately after upgrade

### Database Query Update

The `queryTickets()` function now includes `COALESCE(t.comments,'')` to handle NULL values safely.

---

## Security Considerations

### Settings Updates
- Runtime-only changes (not persisted)
- No authentication required (same as existing endpoints)
- Consider adding authentication if exposing publicly

### Comments
- No sanitization currently implemented
- Comments are stored as plain text
- HTML/JavaScript in comments will be rendered as text (safe)
- Consider adding Markdown support in future

### Export
- Contains ALL data for account
- No filtering by project or date
- Consider adding authentication for production use

---

## Future Enhancements

### Possible Additions
- [ ] Markdown rendering in comments and descriptions
- [ ] Comment editing/deletion
- [ ] User mentions in comments (@username)
- [ ] Comment reactions (👍, ❤️, etc.)
- [ ] Settings persistence to database
- [ ] Export filtering (by project, date range)
- [ ] Import from JSON
- [ ] Ticket history/audit log
- [ ] Attachment support
- [ ] Rich text editor for descriptions

### Not Planned
- ❌ Real-time comment updates (WebSocket)
- ❌ Comment notifications
- ❌ Comment threading/replies
- ❌ User authentication (deploy behind proxy)

---

## Summary

✅ **Settings Modal**: Configure sprint settings and export data to JSON

✅ **Comments System**: Add timestamped comments to tickets with append-only storage

✅ **Ticket View Modal**: Click ticket titles to view/edit all fields, see metadata, manage comments

**Total Lines Added**: ~300 LOC (Go handlers + HTML/CSS + JavaScript)

**Database Changes**: 1 column added (`comments TEXT`)

**API Endpoints Added**: 6 new endpoints

**User Experience**: Significantly enhanced with detailed ticket management

---

**Happy tracking! 🍎**
