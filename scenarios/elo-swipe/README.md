# Elo Swipe - Universal Ranking Engine

Transform any list into a precisely ranked priority order through simple binary comparisons. Uses chess-style Elo ratings to make subjective prioritization objective and reusable.

## 🎯 What It Does

Elo Swipe adds a **permanent ranking capability** to Vrooli. Any scenario that needs to prioritize items - from bug fixes to investment opportunities - can leverage this engine. Every swipe captures human preferences, teaching the system what matters most.

## 🚀 Quick Start

```bash
# Start the scenario
vrooli scenario run elo-swipe

# Create a new list via CLI
elo-swipe create-list --name "Feature Priorities" --items-file items.json

# Or use the web UI
open http://localhost:36850
```

## 💡 Use Cases

- **Development**: Prioritize issues, features, technical debt
- **Product Management**: Rank user stories, feature requests
- **Decision Making**: Compare options with many trade-offs
- **Team Consensus**: Aggregate rankings from multiple users
- **AI Training**: Build preference models for future automation

## 🎮 Interface Options

### Web UI (Tinder-style)
- Swipe left/right or click cards to choose
- Keyboard shortcuts: ← → for selection, Space to skip
- Real-time progress tracking
- Visual rankings with confidence scores

### CLI Interactive Mode
```bash
elo-swipe swipe --list <list-id>
```

### API Integration
```bash
# Create list
curl -X POST http://localhost:30400/api/v1/lists \
  -H "Content-Type: application/json" \
  -d '{"name":"My List","items":[{"content":"Item 1"},{"content":"Item 2"}]}'

# Get rankings (JSON format - default)
curl http://localhost:30400/api/v1/lists/<list-id>/rankings

# Export rankings as CSV
curl http://localhost:30400/api/v1/lists/<list-id>/rankings?format=csv > rankings.csv
```

## 🧮 How It Works

1. **Elo Rating System**: Each item starts at 1500 rating
2. **Binary Comparisons**: Users choose winners in head-to-head matchups
3. **Smart Pairing**: Algorithm selects optimal pairs to minimize comparisons
4. **Convergence**: ~n*log(n) comparisons produce stable rankings
5. **Confidence Scores**: Track certainty based on comparison count

## 🔧 Architecture

- **UI**: Playful swipe interface with smooth animations
- **API**: Go server with PostgreSQL persistence
- **CLI**: Full-featured command-line interface
- **Algorithm**: Standard Elo with K-factor=32

## 📊 Permanent Capability

This scenario becomes a **universal preference engine** for Vrooli:
- Other scenarios can submit lists for human ranking
- Rankings persist and improve over time
- Preference patterns become training data
- Future agents learn to predict priorities

## 🎨 Design Philosophy

**Fun meets Function**: Gamified interface makes tedious prioritization engaging. Professional enough for enterprise use, playful enough to actually want to use.

## 🔌 Integration Examples

```javascript
// From another scenario
const response = await fetch('http://localhost:30400/api/v1/lists', {
  method: 'POST',
  body: JSON.stringify({
    name: 'Bug Priorities',
    items: bugs.map(bug => ({ content: bug }))
  })
});

const { list_id } = await response.json();
// Direct user to ranking interface
window.open(`http://localhost:36850?list=${list_id}`);
```

## 🚦 Status

**Current State: 95% Complete** - All P0 + 4/5 P1 requirements verified

- ✅ Core Elo algorithm
- ✅ Swipe UI (with iframe-bridge integration)
- ✅ PostgreSQL persistence
- ✅ CLI interface
- ✅ Multi-list support
- ✅ Export to JSON/CSV
- ✅ Progress tracking
- ✅ Confidence scores
- ✅ Smart pairing algorithm (with AI fallback)
- ✅ Phased test suite (smoke/unit/integration)
- ✅ Go unit tests for core logic
- ⏳ Undo/skip during swiping (P1 - planned)
- ⏳ Team consensus features (P2)
- ⏳ Preference learning (P2)

## 🔗 Dependencies

- PostgreSQL (required): Persistent storage
- Redis (optional): Performance caching
- Ollama (optional): AI-enhanced comparisons

## 🧪 Testing

Run the comprehensive test suite:

```bash
# Via Makefile (recommended)
make test

# Or directly
./test/run-tests.sh
```

Test phases:
- **Smoke Tests**: API health, CLI status, database connectivity
- **Unit Tests**: Smart pairing logic, AI response parsing
- **Integration Tests**: Full workflows, CSV/JSON export

All tests include auto-port detection for reliable execution.

---

**Permanent Intelligence**: Every ranking makes Vrooli smarter about priorities.

**Last Updated**: 2025-10-03