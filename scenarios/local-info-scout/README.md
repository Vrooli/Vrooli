# Local Info Scout

## Overview
Local Info Scout is an intelligent location-based information discovery tool that helps users find nearby places, services, and points of interest through natural language queries and interactive maps.

### Architecture
The codebase follows a **modular architecture** for maintainability and testability:
- **api/main.go** (839 lines): HTTP handlers, routing, and application startup
- **api/database.go** (195 lines): PostgreSQL operations with connection pooling
- **api/cache.go** (110 lines): Redis caching with graceful degradation
- **api/nlp.go** (131 lines): Natural language query parsing (Ollama integration)
- **api/multisource.go** (515 lines): Multi-source data aggregation (OSM, SearXNG, etc.)
- **api/recommendations.go** (420 lines): Personalized recommendation engine

## Current Status
✅ **Working Features:**
- Health check endpoint (4ms response, compliant with v2.0 schema including readiness field)
- Search API with natural language processing
- Smart filtering with category-aware thresholds and relevance scoring
- Categories API (9 categories)
- Place details API (`/api/places/:id`)
- Discovery API with time-based recommendations and trending places
- **Multi-source data aggregation** - OpenStreetMap, LocalDB, SearXNG, Mock data
- **Personalized recommendations** - User profiles, search history, favorite/hidden categories
- **Trending & Popular places** - Analytics on searches and saved places
- PostgreSQL persistence with search logging and analytics
- Full-featured CLI tool
- **Secure CORS** - Origin validation, no wildcard (0 security vulnerabilities)
- Comprehensive test suite (all 5 phases passing)
- Ollama integration for natural language parsing
- Real-time data integration structure (SearXNG ready)

⚠️ **Known Limitations:**
- Redis caching currently unavailable (requires infrastructure permissions - see PROBLEMS.md)
- Scenario functions correctly without caching; P0 requirements 100% operational

⚠️ **Pending Features:**
- Route planning (P2)
- Social features for sharing discoveries (P2)

## Purpose
This scenario enables local discovery by:
- Finding restaurants, stores, services based on specific criteria
- Providing real-time availability and hours information
- Offering personalized recommendations based on preferences
- Showing distance, ratings, and reviews from multiple sources

## Key Features
- **Natural Language Search**: "Find vegan restaurants within 2 miles"
- **Multi-Source Aggregation**: Combines data from maps, reviews, and local directories
- **Smart Filtering**: Filter by distance, ratings, price, hours, accessibility
- **Real-Time Updates**: Current hours, availability, and wait times
- **Discovery Mode**: Explore categories like "hidden gems" or "new openings"

## Dependencies
- **Shared Workflows**: 
  - `ollama.json` - For understanding natural language queries
  - `rate-limiter.json` - For managing API rate limits
- **Resources**: ollama, n8n, postgres, redis, browserless

## UX Design
Clean, map-focused interface with exploration-friendly design:
- Interactive map as the centerpiece
- Card-based results with photos and key info
- Smooth animations and transitions
- Mobile-first responsive design
- Light theme with accent colors for categories (green for parks, orange for food, etc.)

## Use Cases
- Finding specific services: "Where can I buy cat bowls nearby?"
- Restaurant discovery: "Vegan restaurants open now"
- Emergency needs: "24-hour pharmacy near me"
- Activity planning: "Kid-friendly activities this weekend"
- Local exploration: "Historic sites within walking distance"

## Security
The API implements proper CORS restrictions:
- Only allows requests from specific origins (localhost:3000, localhost:3001)
- Configurable via `ALLOWED_ORIGIN` environment variable
- No wildcard (`*`) CORS that could expose the API to unauthorized domains
- All security audits pass with 0 vulnerabilities

## API Endpoints
- `GET /health` - Service health check
- `POST /api/search` - Location-based search with natural language support
  ```json
  {
    "query": "vegan restaurants within 2 miles",
    "lat": 40.7128,
    "lon": -74.0060,
    "radius": 5,
    "category": "restaurant",
    "min_rating": 4.0,
    "max_price": 3,
    "open_now": true
  }
  ```
- `GET /api/categories` - List available categories
- `GET /api/places/:id` - Get place details
- `POST /api/discover` - Get hidden gems and new openings
  ```json
  {
    "lat": 40.7128,
    "lon": -74.0060
  }
  ```
- `POST /api/recommendations` - Get personalized recommendations (requires `X-User-ID` header)
- `GET /api/profile` - Get user profile and preferences (requires `X-User-ID` header)
- `PUT /api/profile` - Update user preferences (requires `X-User-ID` header)
- `POST /api/places/save` - Save a place to favorites (requires `X-User-ID` header)
- `GET /api/trending` - Get trending searches and popular places (optional `?category=` param)

## CLI Usage
```bash
# Install CLI (if not already built)
cd cli && go build -o local-info-scout .

# Search for places
./local-info-scout "vegan restaurants"
./local-info-scout --query="coffee shops" --radius=2

# List categories
./local-info-scout --categories

# Get help
./local-info-scout --help
```

## Running the Scenario
```bash
# Start the scenario
make run

# Check status
make status

# Run tests
make test                      # Full test suite (all 5 phases)
cd cli && bats local-info-scout.bats  # CLI-specific tests (23 tests)

# View logs
make logs

# Stop the scenario
make stop
```

## Integration Points
- Powers location features in other scenarios
- Can be called by `personal-relationship-manager` for gift shopping
- Integrates with `travel-map-filler` for tourist destinations
- Used by `morning-vision-walk` for finding walking routes