# Product Requirements Document: Typing Test

## Executive Summary
**What**: Interactive web application for testing and improving typing speed and accuracy
**Why**: Help users improve keyboard proficiency, essential for productivity in digital work
**Who**: Students, professionals, and anyone seeking to improve typing skills
**Value**: $15K - SaaS potential for educational institutions and corporate training
**Priority**: Medium - Established market need with clear monetization path

## P0 Requirements (Must Have)
- [x] **Core Typing Test**: Real-time typing interface with text prompt and input field
- [x] **WPM Calculation**: Accurate words-per-minute calculation using standard formula
- [x] **Accuracy Tracking**: Real-time error detection and accuracy percentage display
- [x] **Health Check**: API responds to /health endpoint with proper status
- [x] **Lifecycle Management**: setup/develop/test/stop commands work properly

## P1 Requirements (Should Have)
- [x] **Progress Tracking**: Store and display typing history with PostgreSQL
- [x] **Difficulty Levels**: Multiple text difficulty options (easy/medium/hard)
- [x] **Leaderboard**: Global and personal best scores display
- [ ] **AI Coaching**: Personalized tips using Ollama integration (PARTIAL: basic coaching without AI)

## P2 Requirements (Nice to Have)
- [ ] **Custom Texts**: Allow users to practice with their own text
- [ ] **Export Results**: Download typing history as CSV/PDF
- [ ] **Keyboard Heatmap**: Visual representation of typing patterns

## Technical Specifications

### Architecture
- **Frontend**: HTML/CSS/JavaScript with real-time typing capture
- **Backend**: Go API for result processing and data persistence
- **Database**: PostgreSQL for user sessions and history
- **AI**: Ollama for personalized coaching suggestions
- **CLI**: Command-line interface for quick typing tests

### API Endpoints
- `GET /health` - Health check endpoint
- `POST /api/typing-test` - Submit typing test results
- `GET /api/history` - Retrieve typing history
- `GET /api/leaderboard` - Get leaderboard data
- `POST /api/coaching` - Get AI coaching suggestions

### Dependencies
- Go: gin, uuid, pq (PostgreSQL driver)
- UI: Node.js for server, vanilla JS for frontend
- Resources: PostgreSQL (required), Ollama (optional)

## Success Metrics

### Completion Targets
- P0: 100% complete for v1.0 release
- P1: 50% complete for v1.1 enhancement
- P2: Optional for v2.0 roadmap

### Quality Metrics
- API response time < 500ms
- UI input latency < 50ms
- Accuracy calculation precision > 99%
- Test coverage > 80%

### Performance Metrics
- Support 100 concurrent users
- Database queries < 100ms
- Page load time < 2 seconds

## Progress History
- 2025-09-28: 20% → 85% (Fixed dependencies, verified all P0 requirements working, P1 mostly complete)

## Revenue Justification
- **Educational Market**: $50/month per school subscription × 20 schools = $1000/month
- **Corporate Training**: $200/month per company × 5 companies = $1000/month
- **Individual Premium**: $5/month × 100 users = $500/month
- **Total Monthly Recurring**: $2500/month = $30K annual revenue potential