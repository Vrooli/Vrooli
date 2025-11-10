# Travel Map Filler

## Purpose
Interactive world map application for tracking and visualizing your travel history. Provides a visual way to track where you've been, achievements unlocked, and semantic search for travel memories.

## Core Features
- Visual world map with visited countries/cities highlighted
- Travel achievements system (first country, explorer, world traveler, etc.)
- Semantic search for travel memories using embeddings
- Statistics dashboard (countries visited, distance traveled, world coverage %)
- Travel bucket list management
- Photo memories and notes for each location

## Dependencies
- **n8n**: Workflow automation for travel processing and achievement calculation
- **PostgreSQL**: Store travel records, achievements, and user data
- **Qdrant**: Vector database for semantic search of travel memories
- **Ollama** (optional): Generate travel insights and recommendations
- **Redis** (optional): Cache map data for fast loading

## How It Works
1. Users add travel locations via UI or CLI
2. n8n workflows process the location:
   - Generate embeddings for semantic search
   - Calculate achievements and statistics
   - Store in PostgreSQL and Qdrant
3. UI displays interactive map with pins and heat maps
4. Search finds similar travel experiences using vector similarity

## Integration Points
- Can be used by `itinerary-tracker` for trip planning
- Provides API for other scenarios to query travel history
- Achievements system can integrate with gamification scenarios

## UI Style
**Theme**: Adventure explorer aesthetic
- Interactive world map with smooth zoom/pan
- Pins and heat maps showing travel density
- Achievement badges with fun icons
- Stats dashboard with progress bars
- Photo gallery with location memories
- Clean, modern design with travel-inspired colors (blues, earth tones)

## CLI Commands
```bash
# Add a new travel location
travel-map-filler add "Paris, France" --date "2024-06-15" --type vacation

# Search travel memories
travel-map-filler search "beach sunset"

# View statistics
travel-map-filler stats

# List all travels
travel-map-filler list --year 2024
```

## API Endpoints
- `POST /api/travels` - Add new travel location
- `GET /api/travels` - List all travels with filtering
- `GET /api/travels/search` - Semantic search
- `GET /api/stats` - Get travel statistics
- `GET /api/achievements` - List earned achievements
- `GET /api/map/data` - Get map visualization data

## N8N Workflows
1. **travel-tracker.json** - Process new travel entries, calculate achievements
2. **travel-search.json** - Semantic search using embeddings
3. **travel-list.json** - Generate travel lists with filters

## Testing
```bash
# Test via CLI
travel-map-filler add "Tokyo, Japan" --type business
travel-map-filler stats

# Test via API
curl -X POST http://localhost:8760/api/travels \
  -H "Content-Type: application/json" \
  -d '{"location": "Tokyo, Japan", "date": "2024-11-01", "type": "business"}'
```

## Future Enhancements
- Integration with photo services for automatic geotagging
- Social features to share travel maps with friends
- AI-powered travel recommendations based on history
- Carbon footprint tracking for travels
- Integration with flight/hotel booking data
