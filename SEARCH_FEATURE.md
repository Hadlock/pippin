# 🔍 Live Search Feature

## What Changed

Added a powerful fzf-inspired fuzzy search that filters tickets in real-time as you type!

## ✨ Features

### 1. **Live Fuzzy Search**
- Real-time filtering as you type
- Fuzzy matching algorithm (inspired by fzf)
- Searches across multiple fields:
  - Ticket ID (e.g., "DEMO-7")
  - Title
  - Project key
  - Assignee name

### 2. **Smart Project Context**
- Respects the project dropdown filter
- Search within "All Projects" or specific project
- Switch projects to narrow search scope

### 3. **Visual Feedback**
- Column counts update dynamically
- Matching tickets stay visible
- Non-matching tickets fade away
- Clear button (×) appears when typing
- Search box in top-right corner

### 4. **Keyboard Shortcuts**
- **`/` key** - Focus search (press / from anywhere)
- **`Escape`** - Close modals (also works for search blur)
- **Clear button (×)** - Quick clear and refocus

## 🎯 Fuzzy Matching Algorithm

The search uses a fuzzy matching algorithm similar to fzf:

```javascript
// Matches characters in order, with bonus scoring for consecutive matches
fuzzyMatch("lgn", "Fix login bug")  // ✅ Matches (l-g-n found)
fuzzyMatch("alc", "alice")           // ✅ Matches (a-l-c found)
fuzzyMatch("db", "Add dashboard")    // ✅ Matches (d-b found)
```

**Scoring**:
- Each character match: +1 point
- Consecutive matches: +2 points (bonus)
- Must match all query characters in order

## 🎨 UI Design

### Search Box
- Located in top-right of header
- Flexible width (max 300px)
- Subtle border, accent-colored focus ring
- Placeholder: "🔍 Search tickets..."
- Clear button appears when text entered

### Search Results
- Cards that don't match get `.search-hidden` class
- Column headers update counts dynamically
- Example: "📝 Todo (3)" → "📝 Todo (1)" when filtered

## 📊 Technical Details

### CSS Classes
- `.search-box` - Container for search input
- `.search-hidden` - Hides non-matching cards
- `.clear-search` - Clear button (hidden when empty)

### JavaScript Functions
- `fuzzyMatch(needle, haystack)` - Core matching algorithm
- `searchTickets()` - Main search handler
- `clearSearch()` - Clears input and shows all
- `updateColumnCounts()` - Updates column header counts

### Data Attributes
All cards now have:
- `data-title` - Ticket title
- `data-project` - Project key
- `data-assignee` - Assignee name
- `data-id` - Ticket ID

### Performance
- Event listener on `input` event (debounced by browser)
- Only queries DOM, no API calls
- Instant results (filters in-memory)

## 🧪 Usage Examples

### Example 1: Search by Title
```
Type: "login"
Result: Shows "Fix login bug" ticket
```

### Example 2: Search by Assignee
```
Type: "alice"
Result: Shows all tickets assigned to alice
```

### Example 3: Search by Project-ID
```
Type: "DEMO-7"
Result: Shows ticket #7 from DEMO project
```

### Example 4: Fuzzy Match
```
Type: "db"
Result: Shows "Add dashboard" and "Refactor database"
```

### Example 5: With Project Filter
```
1. Select "PROJ2" from dropdown
2. Type: "dash"
Result: Only shows "Add dashboard" from PROJ2
```

## 🎮 Keyboard Workflow

```
1. Press `/` to focus search (from anywhere)
2. Type query: "alc"
3. See filtered results instantly
4. Click × to clear (or select and delete)
5. Press Escape to blur
```

## 📈 Code Stats

**Before**: 800 lines  
**After**: 905 lines  
**Added**: 105 lines

Breakdown:
- CSS: ~10 lines (search box styling)
- HTML: ~4 lines (search input markup)
- JavaScript: ~90 lines (fuzzy matching + search logic)
- Data attributes: Added to existing cards

**Go logic**: Still 407 lines ✅ (unchanged)

## 🔍 Search Algorithm Details

### Fuzzy Matching
The algorithm matches characters in order, allowing gaps:

```javascript
fuzzyMatch("fzf", "fuzzy finder")
// Matches: f-u-z-z-y f-i-n-d-e-r
//          ^   ^       ^
// Score: 3 (found all chars)
```

### Consecutive Bonus
```javascript
fuzzyMatch("da", "dashboard")
// Matches: d-a-s-h-b-o-a-r-d
//          ^ ^
// Score: 4 (2 chars × 2 bonus for consecutive)
```

### Case Insensitive
All matching is case-insensitive for better UX.

## ✅ Features Summary

- ✅ Real-time fuzzy search
- ✅ Multi-field search (ID, title, project, assignee)
- ✅ Project filter context awareness
- ✅ Dynamic column count updates
- ✅ Keyboard shortcuts (/ to focus)
- ✅ Clear button with auto-hide
- ✅ No API calls (instant results)
- ✅ Focus ring for accessibility
- ✅ fzf-inspired matching algorithm

## 🚀 Try It!

1. Open http://localhost:8080/board
2. Press `/` to focus search
3. Type "alc" to find alice's tickets
4. Type "dash" to find dashboard ticket
5. Select a project from dropdown
6. Search again to see filtered context
7. Click × to clear and see all tickets

**Perfect for quickly finding tickets in busy boards!** 🎯
