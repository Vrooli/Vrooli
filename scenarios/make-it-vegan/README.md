# Make It Vegan - Your Plant-Based Food Companion

## Overview
Make It Vegan is an intelligent food analysis tool that helps users identify whether foods are vegan, understand non-vegan ingredients, and discover delicious plant-based alternatives. Perfect for vegans, those transitioning to plant-based diets, or anyone cooking for vegan friends.

## Why It's Useful
- **Instant Ingredient Analysis**: Scan ingredient lists to identify non-vegan components
- **Smart Alternative Suggestions**: Get contextual replacements that maintain flavor and texture
- **Recipe Conversion**: Transform traditional recipes into vegan versions
- **Nutritional Insights**: Understand protein, B12, and other nutrient considerations
- **Restaurant Menu Helper**: Identify vegan options and request modifications

## Dependencies & Integration
Leverages Vrooli's ecosystem:
- **Ollama**: For ingredient analysis via direct API calls
- **N8n Workflows**: Orchestrates ingredient checking and recipe generation
- **PostgreSQL**: Stores ingredient database and user preferences
- **Redis**: Caches common queries for instant responses
- **Shared Workflows**: Uses project-level Ollama workflow for reliable AI reasoning

## UX Design Philosophy
**Friendly & Vibrant**: A warm, welcoming design that celebrates plant-based living:
- Fresh green and earth tone color palette with playful vegetable illustrations
- Animated ingredient scanner with fun "veggie facts" while processing
- Visual ingredient breakdown showing vegan/non-vegan components
- Recipe cards with appetizing food photography
- Achievement badges for trying new vegan foods
- Mobile-first design for grocery store scanning

The interface feels like a helpful friend rather than a strict judge, encouraging exploration of plant-based options.

## Key Features
1. **Ingredient Scanner**: Paste or photo-scan ingredient lists
2. **Alternative Finder**: Get specific substitutes (e.g., "vegan egg for baking")
3. **Recipe Veganizer**: Convert any recipe to plant-based
4. **Brand Database**: Quick lookup of common products
5. **Meal Planning**: Weekly vegan meal suggestions
6. **Shopping Lists**: Auto-generated lists with store locations

## Technical Architecture
- **API**: Lightweight Go API for fast ingredient lookups
- **Database**: PostgreSQL with comprehensive ingredient taxonomy
- **Caching**: Redis for instant common queries
- **AI Integration**: Ollama for contextual understanding

## Future Enhancements
Could integrate with:
- `nutrition-tracker`: For complete dietary analysis
- `personal-relationship-manager`: Remember dietary preferences of friends
- `local-info-scout`: Find vegan restaurants and stores nearby
- `recipe-generator` (future): Create custom vegan recipes

## CLI Usage
```bash
make-it-vegan check "milk, eggs, flour, sugar"
make-it-vegan substitute "cheese" --context "pizza topping"
make-it-vegan veganize recipe.txt
make-it-vegan brands --category "ice cream"
```

## API Endpoints
- `POST /api/check` - Analyze ingredients for vegan status
- `POST /api/substitute` - Find vegan alternatives
- `POST /api/veganize` - Convert recipe to vegan
- `GET /api/products` - List common non-vegan ingredients
- `GET /api/nutrition` - Get vegan nutritional guidance
- `GET /health` - Service health check

## Environment Variables

### Required Variables (Provided by Lifecycle)
- `API_PORT` - API server port (auto-assigned: 15000-19999)
- `UI_PORT` - Web UI port (auto-assigned: 35000-39999)

### Optional Variables (Graceful Degradation)
- `REDIS_URL` - Redis cache connection (optional, no caching without it)
- `POSTGRES_URL` - PostgreSQL database (optional, uses in-memory fallback)

### CLI Variables
- `MAKE_IT_VEGAN_API_URL` - Override API URL (optional, auto-detects from scenario status)

**Note**: The scenario uses graceful degradation - all optional resources can be unavailable without breaking core functionality.
