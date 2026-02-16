# Date Night Planner 💕

An AI-powered date planning system that creates personalized romantic experiences for couples by learning their preferences and orchestrating multi-step date activities.

## Overview
Date Night Planner transforms the often stressful task of planning memorable dates into an intelligent, personalized experience. It learns from couple preferences, integrates with local venue information, and creates complete date experiences tailored to each couple's unique tastes.

## Features

### Core Capabilities ✅
- **Personalized Date Suggestions** - AI-powered recommendations based on couple preferences
- **Multi-Step Date Orchestration** - Complete date experiences with activities, venues, and timing
- **Preference Learning** - Improves suggestions over time based on feedback
- **Date Memory Storage** - Save photos, notes, and ratings from completed dates
- **Weather-Aware Planning** - Backup indoor alternatives for outdoor activities
- **Surprise Date Mode** - Plan surprise dates with privacy controls and timed reveals

### Integrations
- **scenario-authenticator** - Multi-tenant authentication for couple privacy
- **contact-book** - Partner preference storage and relationship data
- **local-info-scout** - Real-time venue availability (fallback mode available)
- **research-assistant** - Creative date idea research (optional)

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 16+
- PostgreSQL (optional - runs in degraded mode without)

### Installation & Running
```bash
# Start the scenario (includes setup)
make run

# Or use Vrooli CLI
vrooli scenario run date-night-planner
```

### Access Points
- **UI**: http://localhost:[UI_PORT] (pastel-themed romantic interface)
- **API**: http://localhost:[API_PORT]/health
- **CLI**: `date-night-planner --help`

## Usage

### CLI Examples
```bash
# Get date suggestions
date-night-planner suggest test-couple-123 --budget 100 --type romantic

# Create a date plan
date-night-planner plan test-couple-123 suggestion-1 --date 2025-02-14

# Check status
date-night-planner status
```

### API Endpoints
- `POST /api/v1/dates/suggest` - Generate personalized date suggestions
- `POST /api/v1/dates/plan` - Create and save a complete date plan
- `POST /api/v1/dates/surprise` - Create a surprise date plan with privacy controls
- `GET /api/v1/dates/surprise/{id}` - Retrieve surprise date (with access control)
- `GET /health` - Service health check
- `GET /health/database` - Database connectivity check

### Request Examples
```json
// Get date suggestions
POST /api/v1/dates/suggest
{
  "couple_id": "test-couple-123",
  "date_type": "romantic",
  "budget_max": 100,
  "weather_preference": "flexible"
}

// Create surprise date
POST /api/v1/dates/surprise
{
  "couple_id": "couple-123",
  "planned_by": "partner-1",
  "date_suggestion": {
    "title": "Romantic Sunset Dinner",
    "description": "Special surprise evening",
    "activities": [
      {"type": "romantic", "name": "Secret Restaurant", "duration": "2 hours"}
    ],
    "estimated_cost": 200,
    "estimated_duration": "3 hours"
  },
  "planned_date": "2025-02-14T19:00:00Z",
  "reveal_time": "2025-02-14T17:00:00Z"
}
```

## Architecture

### Tech Stack
- **API**: Go (Gorilla Mux, PostgreSQL driver)
- **UI**: Node.js/Express with vanilla JavaScript (pastel aesthetic)
- **CLI**: Bash script with color output
- **Database**: PostgreSQL with custom schema
- **Workflows**: Orchestrated directly in Go API (n8n removed)
- **AI**: Ollama for preference analysis

### Data Models
- **Couples** - Relationship entities with shared preferences
- **Date Plans** - Planned and completed date activities
- **Preferences** - Learned couple preferences with confidence scoring
- **Date Memories** - Photos and notes from completed dates
- **Activity Suggestions** - Reusable date activity templates

## Testing
```bash
# Run all tests
make test

# Run specific test phases
./test/phases/test-integration.sh
./test/phases/test-performance.sh
```

## Troubleshooting

### Common Issues
1. **Database not connected** - Scenario runs in degraded mode, using fallback suggestions
2. **Port conflicts** - Check `.vrooli/service.json` for port configuration
3. **CLI not found** - Run `./cli/install.sh` to install globally

### Debug Commands
```bash
# Check logs
vrooli scenario logs date-night-planner

# Restart scenario
vrooli scenario stop date-night-planner
vrooli scenario start date-night-planner

# Check resource status
vrooli resource status postgres
```

## Development

### Project Structure
```
date-night-planner/
├── api/               # Go API server
├── cli/               # Bash CLI tool
├── ui/                # Node.js UI server
├── initialization/    # Database schemas
│   └── storage/       # PostgreSQL schemas
├── test/              # Test suites
└── .vrooli/          # Scenario configuration
```

### Contributing
1. Follow existing code patterns and styles
2. Test all changes with `make test`
3. Update PRD.md for requirement changes
4. Maintain pastel aesthetic in UI changes

## Performance
- **Response Time**: < 2s for suggestions
- **Memory Usage**: < 512MB under load
- **Concurrent Users**: Supports 100+ couples
- **Cache TTL**: 30 days for venue data

## Security & Privacy
- Couple data encrypted at rest
- Surprise planning mode with separate encryption
- Individual partner permissions
- Complete audit trail for debugging

## License
MIT - Part of the Vrooli ecosystem
