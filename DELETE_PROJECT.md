# 🗑️ Delete Project Feature

## What Changed

Added a safe way to delete projects when you've reached the 3-project limit!

## ✨ Features

### 1. **Conditional Delete Button**
- Only appears when **all conditions met**:
  - ✅ You have 3 projects (at the limit)
  - ✅ Viewing a specific project (not "All Projects")
- Button position: Left of search box
- Styling: Red button with 🗑️ icon
- Label: "🗑️ Delete Project"

### 2. **Safety Confirmation**
- Displays clear confirmation dialog:
  ```
  Delete project PROJ3?
  
  This will permanently delete the project and ALL its tickets.
  
  This action cannot be undone.
  ```
- User must explicitly confirm
- Cancel = no action taken

### 3. **CASCADE Delete**
- Deletes project from database
- **Automatically deletes all tickets** in that project
- **Automatically deletes all block relationships** involving those tickets
- Uses PostgreSQL CASCADE for safety

### 4. **Smart Redirect**
- After successful deletion:
  - Redirects to "All Projects" view
  - Maintains current sprint filter (current/all)
- "Add Project" button reappears (only 2 projects now)

## 🎯 User Flow

### Step 1: Reach Project Limit
```
Projects: DEMO, PROJ2, PROJ3 (3 total)
Status: "Add Project" button is hidden
```

### Step 2: Select Project to Delete
```
1. Use project dropdown
2. Select "PROJ3"
3. Red "🗑️ Delete Project" button appears
```

### Step 3: Delete with Confirmation
```
1. Click "🗑️ Delete Project"
2. Confirmation dialog appears
3. Click OK to confirm (or Cancel to abort)
4. Project and all tickets deleted
5. Redirected to All Projects view
```

### Step 4: Create New Project
```
Projects: DEMO, PROJ2 (2 total)
Status: "Add Project" button now visible again
```

## 🔒 Safety Features

### Multiple Safeguards
1. ✅ Button only visible at 3-project limit
2. ✅ Button only visible when specific project selected
3. ✅ Explicit confirmation dialog
4. ✅ Clear warning about data loss
5. ✅ Database CASCADE ensures referential integrity
6. ✅ 404 error if project doesn't exist
7. ✅ Account isolation (can only delete own projects)

### What Gets Deleted
- ✅ The project record
- ✅ All tickets in that project
- ✅ All block relationships involving those tickets
- ✅ Nothing from other projects

### What's Protected
- ✅ Other projects remain untouched
- ✅ Account data intact
- ✅ Sprint configuration unchanged

## 📊 Technical Details

### API Endpoint
```
DELETE /api/projects/{key}
```

**Parameters**:
- `key` - Project key (e.g., "PROJ3")

**Response** (200 OK):
```json
{"status": "deleted"}
```

**Response** (404 Not Found):
```json
{"error": "project not found"}
```

### Database CASCADE
The PostgreSQL schema uses `ON DELETE CASCADE`:
```sql
-- In tickets table
project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE

-- In blocks table
blocker_ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE
blocked_ticket_id INTEGER REFERENCES tickets(id) ON DELETE CASCADE
```

This ensures:
1. Delete project → triggers cascade to tickets
2. Delete tickets → triggers cascade to blocks
3. No orphaned records
4. Automatic cleanup

### Button Visibility Logic
```go
{{if and (ge (len .Projects) 3) (ne .Project "ALL")}}
<button class="btn btn-danger" onclick="confirmDeleteProject('{{.Project}}')">
  🗑️ Delete Project
</button>
{{end}}
```

Conditions:
- `ge (len .Projects) 3` - At least 3 projects
- `ne .Project "ALL"` - Specific project selected

### JavaScript Confirmation
```javascript
function confirmDeleteProject(projectKey) {
  const confirmed = confirm('Delete project ' + projectKey + '?...');
  if (!confirmed) return;
  
  fetch('/api/projects/' + projectKey, {method: 'DELETE'})
    .then(r => r.json())
    .then(data => {
      if (data.error) alert('Error: ' + data.error);
      else window.location.href = '/board?...&project=ALL';
    });
}
```

## 🎨 UI Design

### Button Styling
```css
.btn-danger {
  background: #ff6b6b;  /* Red */
  color: #fff;
  font-size: 11px;
  padding: 3px 8px;
}
.btn-danger:hover {
  background: #ff5252;  /* Darker red on hover */
}
```

### Button Position
```
Header Layout:
[🍎] [Projects▼] [+ Add Ticket] [Show All] [🗑️ Delete] [🔍 Search]
                                            ↑
                                     Only when 3 projects
                                     + specific project
```

## 📈 Code Stats

**Before**: 905 lines  
**After**: 960 lines  
**Added**: 55 lines

Breakdown:
- Go handler: ~25 lines (`handleDeleteProject`)
- CSS: ~3 lines (`.btn-danger` styling)
- HTML: ~3 lines (conditional button)
- JavaScript: ~24 lines (`confirmDeleteProject`)

**Go logic**: 432 lines (up from 407)

## 🧪 Testing Scenarios

### Scenario 1: Happy Path
```bash
# Setup: 3 projects
curl http://localhost:8080/api/projects | jq length
# Output: 3

# Delete PROJ3
curl -X DELETE http://localhost:8080/api/projects/PROJ3

# Verify
curl http://localhost:8080/api/projects | jq length
# Output: 2
```

### Scenario 2: Project Not Found
```bash
curl -X DELETE http://localhost:8080/api/projects/INVALID
# Output: {"error":"project not found"}
```

### Scenario 3: Tickets Cascade
```bash
# Before: 5 tickets
curl http://localhost:8080/api/tickets | jq length

# Delete project with 2 tickets
curl -X DELETE http://localhost:8080/api/projects/PROJ2

# After: 3 tickets (2 were deleted)
curl http://localhost:8080/api/tickets | jq length
```

## ✅ Features Summary

- ✅ Delete button at 3-project limit
- ✅ Only visible for specific projects
- ✅ Confirmation dialog with clear warning
- ✅ CASCADE deletes tickets and blocks
- ✅ Smart redirect to All Projects
- ✅ "Add Project" button reappears
- ✅ Red danger styling
- ✅ 🗑️ emoji icon
- ✅ Error handling
- ✅ Account isolation

## 🎯 Use Cases

### Use Case 1: Replace Old Project
```
1. Have: CART, ORCH, STORE (limit reached)
2. Want: Add new "MARKET" project
3. Delete: STORE project (old/unused)
4. Create: MARKET project
```

### Use Case 2: Cleanup Test Data
```
1. Created: TEST, DEMO, SANDBOX for testing
2. Done testing
3. Delete: SANDBOX
4. Keep: Production projects
```

### Use Case 3: Project Pivot
```
1. Project failed: Delete ABANDONED
2. Free up slot
3. Create new direction: NEWAPP
```

## 🚀 Try It!

1. Open http://localhost:8080/board
2. Ensure you have 3 projects (create if needed)
3. Select a specific project from dropdown
4. See red "🗑️ Delete Project" button appear
5. Click it
6. Confirm in dialog
7. Watch project and tickets disappear
8. "Add Project" button returns!

**Perfect for managing your 3-project limit!** 🎯
