# 🎯 Picker Wheel

## What It Is
A fun, interactive decision-making tool that brings the excitement of a spinning wheel to any choice. Perfect for those moments when you can't decide - let fate take the wheel!

## Why It's Useful
- **Decision Fatigue Relief**: Eliminate the stress of choosing between equally good options
- **Group Activities**: Perfect for team decisions, game nights, or classroom activities  
- **Fair & Random**: Unbiased selection with customizable probability weights
- **Historical Tracking**: See patterns in your decisions over time
- **Shareable Wheels**: Save and share custom wheels for recurring decisions

## UX Style
**Arcade Casino Vibes** - Bright neon colors, animated spinning effects, and satisfying sound effects create an exciting experience reminiscent of classic arcade games and casino wheels. The interface balances fun animations with practical functionality.

## Key Features
- **Preset Wheels**: Quick access to common decision wheels (dinner, yes/no, D20, etc.)
- **Custom Wheels**: Build your own wheels with unlimited options
- **Weighted Options**: Adjust probability for each option
- **History Tracking**: View past spins and identify patterns
- **Multiple Themes**: Neon arcade, retro casino, minimalist, dark mode
- **Sound & Confetti**: Celebratory effects for each spin

## Scenario Dependencies
- **Shared Resources**: Uses ollama for intelligent suggestions via shared workflows
- **Database**: PostgreSQL for storing wheels and history
- **Automation**: N8n for workflow orchestration

## Integration Points
This scenario demonstrates:
- Clean, themed JavaScript UI without heavy frameworks
- Real-time animations and visual effects
- Local storage with database backup
- Simple API for wheel management
- CLI tool for command-line spinning

## Revenue Potential
- **B2C SaaS**: $5-10/month for premium features (unlimited saves, analytics, team wheels)
- **Educational License**: $200-500 per school/organization
- **API Access**: $50-100/month for developers wanting to integrate decision wheels
- **White Label**: $5K-10K for custom branded versions

## Future Enhancements
- AI-powered option suggestions based on context
- Multiplayer spinning for group decisions
- Integration with calendar for scheduled decisions
- Export results to spreadsheets
- Voice-activated spinning
- Mobile app with shake-to-spin

## API Documentation

### Health Check
```bash
GET /health
```
Returns service health status.

**Response:**
```json
{
  "status": "healthy",
  "service": "picker-wheel-api",
  "version": "1.0.0"
}
```

### Spin Wheel
```bash
POST /api/spin
Content-Type: application/json
```

**Request:**
```json
{
  "wheel_id": "yes-or-no",
  "session_id": "optional-session-id",
  "options": [
    {"label": "Option 1", "color": "#FF6B6B", "weight": 1.0},
    {"label": "Option 2", "color": "#4ECDC4", "weight": 0.5}
  ]
}
```

**Response:**
```json
{
  "result": "Option 1",
  "wheel_id": "yes-or-no",
  "session_id": "session_123",
  "timestamp": "2025-10-03T08:00:00Z"
}
```

### List Wheels
```bash
GET /api/wheels
```

**Response:**
```json
[
  {
    "id": "dinner-decider",
    "name": "Dinner Decider",
    "description": "Can't decide what to eat?",
    "options": [...],
    "theme": "food",
    "created_at": "2025-10-03T08:00:00Z",
    "times_used": 42
  }
]
```

### Create Wheel
```bash
POST /api/wheels
Content-Type: application/json
```

**Request:**
```json
{
  "name": "My Custom Wheel",
  "description": "Optional description",
  "options": [
    {"label": "Option 1", "color": "#FF6B6B", "weight": 1.0},
    {"label": "Option 2", "color": "#4ECDC4", "weight": 1.0}
  ],
  "theme": "neon"
}
```

**Response:**
```json
{
  "id": "wheel_1234567890",
  "name": "My Custom Wheel",
  ...
}
```

### Get Wheel Details
```bash
GET /api/wheels/{id}
```

### Delete Wheel
```bash
DELETE /api/wheels/{id}
```

### Get Spin History
```bash
GET /api/history
```

**Response:**
```json
[
  {
    "result": "Pizza 🍕",
    "wheel_id": "dinner-decider",
    "session_id": "session_123",
    "timestamp": "2025-10-03T08:00:00Z"
  }
]
```

## CLI Usage

```bash
# Spin a preset wheel
./cli/picker-wheel --wheel yes-or-no

# Spin with custom options
./cli/picker-wheel --options "Yes,No,Maybe"

# Spin with weights
./cli/picker-wheel --options "Yes:2,No:1,Maybe:0.5"
```

## Technical Notes
- Lightweight Go API for fast response times
- SVG-based wheel rendering for smooth animations
- WebSocket support for real-time multiplayer (future)
- Responsive design works on all devices
- Weighted probability selection using normalized random distribution
- PostgreSQL persistence with automatic in-memory fallback
- History tracking with session support