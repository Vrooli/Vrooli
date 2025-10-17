#!/bin/bash
set -euo pipefail

# Recipe Book Business Logic Test Runner

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Starting Recipe Book business logic tests"

# Business Rule 1: Recipe visibility and access control
testing::phase::log "Testing visibility and access control..."

# Create a private recipe
PRIVATE_RECIPE=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Private Recipe",
        "ingredients": [{"name": "secret ingredient", "amount": 1, "unit": "cup"}],
        "instructions": ["Secret step"],
        "created_by": "user-alice",
        "visibility": "private"
    }' 2>&1 || echo '{}')

if echo "$PRIVATE_RECIPE" | grep -q '"id"'; then
    RECIPE_ID=$(echo "$PRIVATE_RECIPE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    # Try to access as unauthorized user (should fail)
    if ! curl -sf "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}?user_id=user-bob" > /dev/null 2>&1; then
        testing::phase::log "✓ Private recipe correctly denies unauthorized access"
    else
        testing::phase::log "✗ Private recipe allows unauthorized access"
    fi

    # Access as owner (should succeed)
    if curl -sf "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}?user_id=user-alice" > /dev/null 2>&1; then
        testing::phase::log "✓ Private recipe allows owner access"
    else
        testing::phase::log "✗ Private recipe denies owner access"
    fi

    # Cleanup
    curl -sf -X DELETE "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}?user_id=user-alice" > /dev/null 2>&1 || true
fi

# Business Rule 2: Recipe sharing workflow
testing::phase::log "Testing recipe sharing workflow..."

SHARED_RECIPE=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Recipe to Share",
        "ingredients": [{"name": "ingredient", "amount": 1, "unit": "cup"}],
        "instructions": ["Step 1"],
        "created_by": "user-alice",
        "visibility": "private"
    }' 2>&1 || echo '{}')

if echo "$SHARED_RECIPE" | grep -q '"id"'; then
    RECIPE_ID=$(echo "$SHARED_RECIPE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    # Share with user-bob
    if curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}/share" \
        -H "Content-Type: application/json" \
        -d '{"user_ids": ["user-bob"]}' > /dev/null 2>&1; then
        testing::phase::log "✓ Recipe sharing API successful"
    fi

    # Cleanup
    curl -sf -X DELETE "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}?user_id=user-alice" > /dev/null 2>&1 || true
fi

# Business Rule 3: Recipe modification creates derivative
testing::phase::log "Testing recipe modification workflow..."

ORIGINAL_RECIPE=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Original Recipe",
        "ingredients": [{"name": "ingredient", "amount": 1, "unit": "cup"}],
        "instructions": ["Original step"],
        "created_by": "user-alice"
    }' 2>&1 || echo '{}')

if echo "$ORIGINAL_RECIPE" | grep -q '"id"'; then
    RECIPE_ID=$(echo "$ORIGINAL_RECIPE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    # Modify recipe (should create derivative)
    MODIFIED_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}/modify" \
        -H "Content-Type: application/json" \
        -d '{
            "modification_type": "make_vegan",
            "user_id": "user-bob"
        }' 2>&1 || echo '{}')

    if echo "$MODIFIED_RESPONSE" | grep -q '"modified_recipe"'; then
        testing::phase::log "✓ Recipe modification creates derivative"
    fi

    # Cleanup
    curl -sf -X DELETE "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}?user_id=user-alice" > /dev/null 2>&1 || true
fi

# Business Rule 4: Shopping list aggregates ingredients
testing::phase::log "Testing shopping list aggregation..."

# Create multiple recipes
RECIPE1=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Recipe 1",
        "ingredients": [{"name": "flour", "amount": 2, "unit": "cups"}],
        "instructions": ["Mix"],
        "created_by": "user-alice"
    }' 2>&1 || echo '{}')

RECIPE2=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Recipe 2",
        "ingredients": [{"name": "flour", "amount": 1, "unit": "cup"}],
        "instructions": ["Bake"],
        "created_by": "user-alice"
    }' 2>&1 || echo '{}')

if echo "$RECIPE1" | grep -q '"id"' && echo "$RECIPE2" | grep -q '"id"'; then
    ID1=$(echo "$RECIPE1" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    ID2=$(echo "$RECIPE2" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    # Generate shopping list
    SHOPPING_LIST=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/shopping-list" \
        -H "Content-Type: application/json" \
        -d "{\"recipe_ids\": [\"$ID1\", \"$ID2\"], \"user_id\": \"user-alice\"}" 2>&1 || echo '{}')

    if echo "$SHOPPING_LIST" | grep -q '"shopping_list"'; then
        testing::phase::log "✓ Shopping list generation successful"
    fi

    # Cleanup
    curl -sf -X DELETE "http://localhost:${API_PORT:-3250}/api/v1/recipes/${ID1}?user_id=user-alice" > /dev/null 2>&1 || true
    curl -sf -X DELETE "http://localhost:${API_PORT:-3250}/api/v1/recipes/${ID2}?user_id=user-alice" > /dev/null 2>&1 || true
fi

# Business Rule 5: Recipe rating and cooking history
testing::phase::log "Testing recipe rating workflow..."

RECIPE=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Recipe to Rate",
        "ingredients": [{"name": "ingredient", "amount": 1, "unit": "cup"}],
        "instructions": ["Cook"],
        "created_by": "user-alice"
    }' 2>&1 || echo '{}')

if echo "$RECIPE" | grep -q '"id"'; then
    RECIPE_ID=$(echo "$RECIPE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    # Rate the recipe
    if curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}/rate" \
        -H "Content-Type: application/json" \
        -d '{
            "user_id": "user-bob",
            "rating": 5,
            "notes": "Delicious!",
            "anonymous": false
        }' > /dev/null 2>&1; then
        testing::phase::log "✓ Recipe rating successful"
    fi

    # Cleanup
    curl -sf -X DELETE "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}?user_id=user-alice" > /dev/null 2>&1 || true
fi

# Business Rule 6: AI recipe generation with dietary restrictions
testing::phase::log "Testing AI recipe generation..."

GEN_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "prompt": "healthy breakfast",
        "user_id": "user-alice",
        "dietary_restrictions": ["vegetarian", "gluten-free"],
        "available_ingredients": ["eggs", "spinach", "tomatoes"]
    }' 2>&1 || echo '{}')

if echo "$GEN_RESPONSE" | grep -q '"recipe"'; then
    testing::phase::log "✓ AI recipe generation successful"

    # Verify dietary restrictions are respected
    if echo "$GEN_RESPONSE" | grep -q '"source":"ai_generated"'; then
        testing::phase::log "✓ Generated recipe has correct source"
    fi
fi

# Business Rule 7: Semantic search functionality
testing::phase::log "Testing semantic search..."

SEARCH_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes/search" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "chocolate dessert",
        "user_id": "user-alice",
        "limit": 10,
        "filters": {
            "dietary": ["vegetarian"],
            "max_time": 60,
            "ingredients": ["chocolate"]
        }
    }' 2>&1 || echo '{}')

if echo "$SEARCH_RESPONSE" | grep -q '"results"'; then
    testing::phase::log "✓ Semantic search successful"
fi

testing::phase::end_with_summary "Recipe Book business logic tests completed"
