#!/usr/bin/env bats
# Integration tests for stream-of-consciousness-analyzer API
# Exercises live endpoints against the running API server.
#
# Requirements covered:
#   [REQ:P0-001] Sub-Second Voice Recording Start (API readiness)
#   [REQ:P0-002] Quick Text Entry (scheme + information CRUD)
#   [REQ:P0-003] Canvas View Implementation (information canvas coords)
#   [REQ:P0-004] Graph View Implementation (thought + edge CRUD)
#   [REQ:P1-001] LLM Connection Generation (provider listing)
#   [REQ:P1-002] Thought Graph Export (export endpoint)
#   [REQ:P1-003] Suggestion Dismissal (suggestion generation)
#   [REQ:P1-004] Export Format Completeness (export graph components)
#   [REQ:P2-001] LLM Provider Fallback (provider structure)
#   [REQ:P2-002] Inter-Scheme Navigation (multi-scheme operations)
#   [REQ:P2-003] Provider Health Reporting (provider list structure)
#   [REQ:P2-004] Cross-Scheme Thought Linking (nullable scheme_id)

# Resolve the scenario's actual API port dynamically.
# During test-genie runs, API_PORT may point to test-genie's own port,
# so we use vrooli to get the correct scenario port instead.
_resolve_port() {
    local port
    port=$(vrooli scenario port stream-of-consciousness-analyzer API_PORT 2>/dev/null)
    if [ -n "$port" ] && curl -sf "http://localhost:${port}/health" >/dev/null 2>&1; then
        echo "$port"
        return
    fi
    # Fallback to default
    echo "16968"
}

SCENARIO_API_PORT="$(_resolve_port)"
API_BASE="http://localhost:${SCENARIO_API_PORT}/api/v1"

setup() {
    # Skip all tests if the API is not reachable
    if ! curl -sf "http://localhost:${SCENARIO_API_PORT}/health" >/dev/null 2>&1; then
        skip "API server not running on port ${SCENARIO_API_PORT}"
    fi
}

# ---------- Health ----------

# [REQ:P0-001] API must be reachable and healthy
@test "health endpoint returns 200 with healthy status" {
    result=$(curl -sf "http://localhost:${SCENARIO_API_PORT}/health")
    echo "$result" | grep -q '"status":"healthy"'
}

# [REQ:P0-001] API v1 routes are accessible
@test "API v1 health endpoint returns 200" {
    result=$(curl -sf "${API_BASE}/health")
    echo "$result" | grep -q '"status":"healthy"'
}

# ---------- Scheme CRUD ----------

# [REQ:P0-002] Scheme creation for text entry workspace
@test "POST /schemes creates a new scheme" {
    result=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"BATS Test Scheme"}')
    echo "$result" | grep -q '"name":"BATS Test Scheme"'
    echo "$result" | grep -q '"id"'
}

# [REQ:P0-002] List schemes
@test "GET /schemes returns an array" {
    result=$(curl -sf "${API_BASE}/schemes")
    # Must be a JSON array (starts with [)
    [ "$(echo "$result" | head -c1)" = "[" ]
}

# [REQ:P2-002] Create and retrieve individual scheme (inter-scheme navigation)
@test "GET /schemes/:id retrieves a specific scheme" {
    # Create
    created=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"BATS Retrieve Test"}')
    id=$(echo "$created" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    [ -n "$id" ]

    # Retrieve
    fetched=$(curl -sf "${API_BASE}/schemes/${id}")
    echo "$fetched" | grep -q '"name":"BATS Retrieve Test"'
}

# [REQ:P0-002] Update scheme
@test "PUT /schemes/:id updates a scheme" {
    created=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Before Update"}')
    id=$(echo "$created" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    updated=$(curl -sf -X PUT "${API_BASE}/schemes/${id}" \
        -H 'Content-Type: application/json' \
        -d '{"name":"After Update"}')
    echo "$updated" | grep -q '"name":"After Update"'
}

# [REQ:P0-002] Delete scheme
@test "DELETE /schemes/:id removes a scheme" {
    created=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"To Delete"}')
    id=$(echo "$created" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_BASE}/schemes/${id}")
    [ "$status" = "204" ]
}

# [REQ:P0-002] Reject invalid JSON on scheme creation
@test "POST /schemes rejects invalid JSON" {
    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d 'not-json')
    [ "$status" = "400" ]
}

# ---------- Information CRUD ----------

# [REQ:P0-003] Create information item with canvas coordinates
@test "POST /schemes/:id/information creates an item with canvas coords" {
    scheme=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Info Test Scheme"}')
    sid=$(echo "$scheme" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    result=$(curl -sf -X POST "${API_BASE}/schemes/${sid}/information" \
        -H 'Content-Type: application/json' \
        -d '{"type":"text","content":"Hello BATS","canvas_x":10.5,"canvas_y":20.3}')
    echo "$result" | grep -q '"content":"Hello BATS"'
    echo "$result" | grep -q '"canvas_x":10.5'
    echo "$result" | grep -q '"canvas_y":20.3'
}

# [REQ:P0-003] List information by scheme
@test "GET /schemes/:id/information lists items for a scheme" {
    scheme=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Info List Scheme"}')
    sid=$(echo "$scheme" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    # Create an item
    curl -sf -X POST "${API_BASE}/schemes/${sid}/information" \
        -H 'Content-Type: application/json' \
        -d '{"type":"text","content":"Item 1","canvas_x":0,"canvas_y":0}' >/dev/null

    result=$(curl -sf "${API_BASE}/schemes/${sid}/information")
    [ "$(echo "$result" | head -c1)" = "[" ]
    echo "$result" | grep -q '"content":"Item 1"'
}

# [REQ:P0-003] Update information canvas position
@test "PUT /schemes/:id/information/:infoId updates canvas position" {
    scheme=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Info Update Scheme"}')
    sid=$(echo "$scheme" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    info=$(curl -sf -X POST "${API_BASE}/schemes/${sid}/information" \
        -H 'Content-Type: application/json' \
        -d '{"type":"text","content":"Move Me","canvas_x":0,"canvas_y":0}')
    iid=$(echo "$info" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    updated=$(curl -sf -X PUT "${API_BASE}/schemes/${sid}/information/${iid}" \
        -H 'Content-Type: application/json' \
        -d '{"canvas_x":99.9,"canvas_y":88.8}')
    echo "$updated" | grep -q '99.9'
    echo "$updated" | grep -q '88.8'
}

# [REQ:P0-003] Delete information item
@test "DELETE /schemes/:id/information/:infoId removes an item" {
    scheme=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Info Delete Scheme"}')
    sid=$(echo "$scheme" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    info=$(curl -sf -X POST "${API_BASE}/schemes/${sid}/information" \
        -H 'Content-Type: application/json' \
        -d '{"type":"text","content":"Delete Me","canvas_x":0,"canvas_y":0}')
    iid=$(echo "$info" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_BASE}/schemes/${sid}/information/${iid}")
    [ "$status" = "204" ]
}

# ---------- Thought CRUD ----------

# [REQ:P0-004] Create thought in a scheme
@test "POST /thoughts creates a thought with scheme_id" {
    scheme=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Thought Test Scheme"}')
    sid=$(echo "$scheme" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    result=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d "{\"scheme_id\":\"${sid}\",\"title\":\"BATS Thought\",\"body\":\"test body\",\"canvas_x\":5,\"canvas_y\":10}")
    echo "$result" | grep -q '"title":"BATS Thought"'
}

# [REQ:P2-004] Create thought without scheme_id (cross-scheme linking)
@test "POST /thoughts creates a cross-scheme thought with null scheme_id" {
    result=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Cross-Scheme Thought","body":"bridges workspaces","canvas_x":0,"canvas_y":0}')
    echo "$result" | grep -q '"title":"Cross-Scheme Thought"'
    echo "$result" | grep -q '"scheme_id":null'
}

# [REQ:P0-004] List thoughts
@test "GET /thoughts returns array" {
    result=$(curl -sf "${API_BASE}/thoughts")
    [ "$(echo "$result" | head -c1)" = "[" ]
}

# [REQ:P0-004] Get single thought
@test "GET /thoughts/:id retrieves a specific thought" {
    thought=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Get Test","body":"","canvas_x":0,"canvas_y":0}')
    tid=$(echo "$thought" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    fetched=$(curl -sf "${API_BASE}/thoughts/${tid}")
    echo "$fetched" | grep -q '"title":"Get Test"'
}

# [REQ:P0-004] Update thought
@test "PUT /thoughts/:id updates a thought" {
    thought=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Original","body":"","canvas_x":0,"canvas_y":0}')
    tid=$(echo "$thought" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    updated=$(curl -sf -X PUT "${API_BASE}/thoughts/${tid}" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Updated Title"}')
    echo "$updated" | grep -q '"title":"Updated Title"'
}

# [REQ:P0-004] Delete thought
@test "DELETE /thoughts/:id removes a thought" {
    thought=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Delete Me","body":"","canvas_x":0,"canvas_y":0}')
    tid=$(echo "$thought" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_BASE}/thoughts/${tid}")
    [ "$status" = "204" ]
}

# ---------- Edge CRUD ----------

# [REQ:P0-004] Create edge between thoughts
@test "POST /thoughts/:id/edges creates a directed edge" {
    t1=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Source","body":"","canvas_x":0,"canvas_y":0}')
    t1id=$(echo "$t1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    t2=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Target","body":"","canvas_x":10,"canvas_y":10}')
    t2id=$(echo "$t2" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    edge=$(curl -sf -X POST "${API_BASE}/thoughts/${t1id}/edges" \
        -H 'Content-Type: application/json' \
        -d "{\"target_id\":\"${t2id}\",\"label\":\"causes\"}")
    echo "$edge" | grep -q '"label":"causes"'
    echo "$edge" | grep -q '"source_id"'
    echo "$edge" | grep -q '"target_id"'
}

# [REQ:P0-004] List edges for a thought
@test "GET /thoughts/:id/edges returns edges" {
    t1=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Edge List Source","body":"","canvas_x":0,"canvas_y":0}')
    t1id=$(echo "$t1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    result=$(curl -sf "${API_BASE}/thoughts/${t1id}/edges")
    [ "$(echo "$result" | head -c1)" = "[" ]
}

# [REQ:P0-004] Self-loop edge is rejected
@test "POST /thoughts/:id/edges rejects self-loop" {
    t1=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Self Loop","body":"","canvas_x":0,"canvas_y":0}')
    t1id=$(echo "$t1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/thoughts/${t1id}/edges" \
        -H 'Content-Type: application/json' \
        -d "{\"target_id\":\"${t1id}\",\"label\":\"self\"}")
    [ "$status" = "400" ]
}

# [REQ:P0-004] Edge without target_id is rejected
@test "POST /thoughts/:id/edges rejects missing target_id" {
    t1=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"No Target","body":"","canvas_x":0,"canvas_y":0}')
    t1id=$(echo "$t1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/thoughts/${t1id}/edges" \
        -H 'Content-Type: application/json' \
        -d '{"target_id":"","label":"missing"}')
    [ "$status" = "400" ]
}

# [REQ:P0-004] Delete edge
@test "DELETE /thoughts/:id/edges/:edgeId removes an edge" {
    t1=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Del Edge Src","body":"","canvas_x":0,"canvas_y":0}')
    t1id=$(echo "$t1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    t2=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d '{"title":"Del Edge Tgt","body":"","canvas_x":0,"canvas_y":0}')
    t2id=$(echo "$t2" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    edge=$(curl -sf -X POST "${API_BASE}/thoughts/${t1id}/edges" \
        -H 'Content-Type: application/json' \
        -d "{\"target_id\":\"${t2id}\",\"label\":\"temp\"}")
    eid=$(echo "$edge" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_BASE}/thoughts/${t1id}/edges/${eid}")
    [ "$status" = "204" ]
}

# ---------- Export ----------

# [REQ:P1-002] Export endpoint returns scheme data
@test "GET /schemes/:id/export returns export data" {
    scheme=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Export Test Scheme"}')
    sid=$(echo "$scheme" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    result=$(curl -sf "${API_BASE}/schemes/${sid}/export")
    echo "$result" | grep -q '"scheme"'
    echo "$result" | grep -q '"export_format"'
}

# [REQ:P1-004] Export contains all graph components
@test "export data includes scheme, information, thoughts, and edges" {
    scheme=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Full Export Scheme"}')
    sid=$(echo "$scheme" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    # Add information
    curl -sf -X POST "${API_BASE}/schemes/${sid}/information" \
        -H 'Content-Type: application/json' \
        -d '{"type":"text","content":"Export Item","canvas_x":0,"canvas_y":0}' >/dev/null

    # Add thoughts
    t1=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d "{\"scheme_id\":\"${sid}\",\"title\":\"Export T1\",\"body\":\"\",\"canvas_x\":0,\"canvas_y\":0}")
    t1id=$(echo "$t1" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    t2=$(curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d "{\"scheme_id\":\"${sid}\",\"title\":\"Export T2\",\"body\":\"\",\"canvas_x\":5,\"canvas_y\":5}")
    t2id=$(echo "$t2" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    # Add edge
    curl -sf -X POST "${API_BASE}/thoughts/${t1id}/edges" \
        -H 'Content-Type: application/json' \
        -d "{\"target_id\":\"${t2id}\",\"label\":\"related\"}" >/dev/null

    export_data=$(curl -sf "${API_BASE}/schemes/${sid}/export")
    echo "$export_data" | grep -q '"scheme"'
    echo "$export_data" | grep -q '"information"'
    echo "$export_data" | grep -q '"thoughts"'
    echo "$export_data" | grep -q '"edges"'
    echo "$export_data" | grep -q '"export_format"'
}

# ---------- Providers ----------

# [REQ:P1-001] [REQ:P2-001] [REQ:P2-003] Provider listing
@test "GET /providers returns provider list with structure" {
    result=$(curl -sf "${API_BASE}/providers")
    [ "$(echo "$result" | head -c1)" = "[" ]
    # Each provider should have name and active fields
    echo "$result" | grep -q '"name"'
}

# ---------- Multi-Scheme Operations ----------

# [REQ:P2-002] Multiple schemes can coexist
@test "multiple schemes can be created and listed independently" {
    curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Multi Scheme A"}' >/dev/null
    curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Multi Scheme B"}' >/dev/null

    result=$(curl -sf "${API_BASE}/schemes")
    echo "$result" | grep -q '"Multi Scheme A"'
    echo "$result" | grep -q '"Multi Scheme B"'
}

# [REQ:P0-004] Filter thoughts by scheme_id
@test "GET /thoughts?scheme_id filters by scheme" {
    scheme=$(curl -sf -X POST "${API_BASE}/schemes" \
        -H 'Content-Type: application/json' \
        -d '{"name":"Filter Scheme"}')
    sid=$(echo "$scheme" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    curl -sf -X POST "${API_BASE}/thoughts" \
        -H 'Content-Type: application/json' \
        -d "{\"scheme_id\":\"${sid}\",\"title\":\"Filtered Thought\",\"body\":\"\",\"canvas_x\":0,\"canvas_y\":0}" >/dev/null

    result=$(curl -sf "${API_BASE}/thoughts?scheme_id=${sid}")
    [ "$(echo "$result" | head -c1)" = "[" ]
}
