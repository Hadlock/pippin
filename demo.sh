#!/bin/bash
# Pippin Demo Script

echo "🍎 Pippin - Quick Demo"
echo "====================="
echo ""

# Check if server is running
if ! curl -s http://localhost:8080/api/projects > /dev/null 2>&1; then
    echo "❌ Server not running. Start it with: make run"
    exit 1
fi

echo "1️⃣  Listing Projects (max 3):"
curl -s http://localhost:8080/api/projects | jq -r '.[] | "   \(.key): \(.name)"'
echo ""

echo "2️⃣  Listing Tickets:"
curl -s http://localhost:8080/api/tickets | jq -r '.[] | "   [\(.state)] \(.project_key)-\(.id): \(.title) (@\(.assignee))"'
echo ""

echo "3️⃣  Testing Project Limit:"
RESULT=$(curl -s -X POST http://localhost:8080/api/projects \
  -H 'Content-Type: application/json' \
  -d '{"key":"TEST","name":"Test Project"}')
echo "   $RESULT"
echo ""

echo "4️⃣  Creating a new ticket:"
NEW_TICKET=$(curl -s -X POST http://localhost:8080/api/tickets \
  -H 'Content-Type: application/json' \
  -d '{
    "project_key":"CART",
    "title":"Paint the cart",
    "body":"Choose a nice color",
    "assignee":"bob",
    "state":"backlog"
  }')
TICKET_ID=$(echo $NEW_TICKET | jq -r '.id')
echo "   Created ticket #$TICKET_ID"
echo ""

echo "5️⃣  Moving ticket #$TICKET_ID from backlog → todo:"
MOVE_RESULT=$(curl -s -X POST http://localhost:8080/api/tickets/$TICKET_ID/move \
  -H 'Content-Type: application/json' \
  -d '{"direction":"right"}')
echo "   New state: $(echo $MOVE_RESULT | jq -r '.state')"
echo ""

echo "6️⃣  Adding a blocking relationship:"
BLOCK_RESULT=$(curl -s -X POST http://localhost:8080/api/tickets/$TICKET_ID/blocks \
  -H 'Content-Type: application/json' \
  -d '{"blocked_id":3}')
echo "   Status: $(echo $BLOCK_RESULT | jq -r '.status')"
echo ""

echo "✅ Demo complete!"
echo ""
echo "View the board at: http://localhost:8080/board"
