# Product Requirements Document (PRD)

> **Template Version**: 2.0.0
> **Canonical Reference**: [scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md](../prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md)

## 🎯 Overview

**Travel Map Filler** is an interactive world map application that enables users to visualize and track their travel history, earn achievements, and manage their bucket list. This scenario provides a gamified approach to travel documentation with semantic search capabilities and rich data visualization.

**Purpose**: Add a permanent capability for visualizing, tracking, and gaining insights from personal travel data through interactive maps, statistics, and AI-powered search.

**Primary Users**:
- Frequent travelers wanting to document and visualize their journey
- Travel bloggers needing visual content for their stories
- Travel agencies looking to understand client travel patterns

**Deployment Surfaces**: CLI, API, UI

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Travel data CRUD | Add, edit, delete, and view travel locations with dates, types, notes, and duration
- [ ] OT-P0-002 | Interactive map visualization | Display travels on a Leaflet.js world map with zoom, pan, and color-coded pins by travel type
- [ ] OT-P0-003 | Location geocoding | Automatically resolve location names to latitude/longitude coordinates
- [ ] OT-P0-004 | Basic statistics | Display real-time counts for countries, cities, continents visited
- [ ] OT-P0-005 | Data persistence | PostgreSQL storage for all travel data with proper schema and migrations
- [ ] OT-P0-006 | Go API backend | RESTful API with standard endpoints for all CRUD operations

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Achievement system | Automatic achievement calculation and display for milestones like First Steps, Explorer, World Traveler, Continent Collector
- [ ] OT-P1-002 | Travel analytics | Distance calculations, world coverage percentage, time-based patterns, travels per year
- [ ] OT-P1-003 | Semantic search | Vector-based search through travel notes using Qdrant with relevance scoring
- [ ] OT-P1-004 | Bucket list management | Add, prioritize, and convert dream destinations to actual travels with budget and date tracking
- [ ] OT-P1-005 | Heat map overlay | Visual density layer showing travel concentration areas
- [ ] OT-P1-006 | Photo attachments | Support for uploading and viewing travel photos
- [ ] OT-P1-007 | Data export | Export travel data as JSON/CSV and generate shareable statistics summaries
- [ ] OT-P1-008 | Embedded automation pipeline | API handles embedding, achievement, and persistence orchestration without a dedicated workflow engine

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | AI travel insights | Ollama integration for predictive recommendations and pattern recognition
- [ ] OT-P2-002 | Social features | Share travel maps with friends and multi-user travel planning
- [ ] OT-P2-003 | Booking integration | Connect with flight/hotel platforms for seamless trip planning
- [ ] OT-P2-004 | Carbon footprint tracking | Environmental awareness metrics for travel impact
- [ ] OT-P2-005 | Photo AI tagging | Automated photo categorization and analysis
- [ ] OT-P2-006 | Offline PWA mode | Progressive Web App with offline capability for viewing existing data

## 🧱 Tech Direction Snapshot

**API Stack**: Go with Gin/Echo framework for RESTful endpoints, standard middleware for logging and error handling.

**UI Stack**: Responsive HTML5/CSS3/JavaScript frontend with Leaflet.js for mapping. Progressive Web App capabilities for mobile usage.

**Data Storage**:
- PostgreSQL for relational travel data (locations, trips, users)
- Qdrant for vector embeddings (semantic search)
- Redis for caching map data and frequent queries

**Integration Strategy**:
- Direct API automation for embeddings, achievements, and storage coordination
- Ollama for AI-powered insights (post-launch)
- External geocoding service for location resolution

**Non-Goals**:
- Real-time collaboration on travel planning (P2 future)
- Native mobile apps (PWA sufficient for MVP)
- Social network features beyond basic sharing

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- PostgreSQL (primary data store)
- Qdrant (vector search)
- Redis (caching layer)

**Scenario Dependencies**: None (standalone scenario)

**Risks**:
- Geocoding API rate limits may throttle batch imports – mitigate with caching and fallback strategies
- Vector database latency for large datasets – optimize indexing and query strategies
- Map rendering performance varies by browser – test across devices, degrade gracefully

**Launch Sequencing**:
1. Deploy PostgreSQL schema and seed data
2. Start API backend with health checks
3. Verify the embedded automation pipeline handles embeddings and achievements
4. Launch UI with basic map and CRUD forms
5. Post-launch: add achievements, analytics, semantic search

## 🎨 UX & Branding

**Visual Palette**: Adventure-themed design with earth tones (browns, greens, blues). Clean, minimal interface that puts the map front and center. Standard map controls (zoom, reset, layer toggles) follow familiar patterns.

**Typography & Motion**: Sans-serif fonts for readability at all sizes. Smooth animations at 60fps minimum for pin placement, map transitions, and modal popups.

**Accessibility**: WCAG 2.1 Level AA compliance. Color-coded pins must have pattern/icon alternatives for colorblind users. Keyboard navigation for all map interactions. Screen reader support for statistics and achievement announcements.

**Voice & Personality**: Encouraging and adventurous without being overwhelming. Achievement unlocks use positive, non-competitive language. Error messages are helpful and actionable.

**Mobile Experience**: Responsive design prioritized for smartphone usage. Touch-friendly controls with appropriate hit targets. Simplified UI for smaller screens without losing core functionality.

## 📎 Appendix

**Performance Targets**:
- Map loads within 3 seconds on standard connections
- Search results displayed within 2 seconds
- API response times < 200ms (95th percentile)
- 99.5% uptime for core functionality

**Success Metrics**:
- Average session duration > 5 minutes
- >90% of travels include location notes
- >70% of users unlock at least one achievement
- 60% monthly active user retention rate

**Reference Links**:
- Leaflet.js documentation: https://leafletjs.com/
- Qdrant vector search: https://qdrant.tech/
