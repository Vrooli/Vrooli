# Product Requirements Document (PRD)

> **Template Version**: 2.0.0
> **Canonical Reference**: PRD Control Tower
> **Status**: Static Operational Charter

## 🎯 Overview

**Purpose**: Interactive application for tracking and forecasting fall foliage peaks across North America, enabling data-driven autumn travel planning.

**Primary Users**:
- Nature enthusiasts planning seasonal trips
- Photographers seeking peak foliage conditions
- Travel planners and tourism boards
- Regional tourism partnerships

**Deployment Surfaces**:
- Web UI with interactive maps (Leaflet.js)
- REST API for foliage data and predictions
- CLI for lifecycle management and testing

**Permanent Capability**: Vrooli gains real-time foliage tracking and AI-powered peak prediction as a reusable scenario, demonstrating weather integration, map visualization, and crowd-sourced reporting patterns.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Health Check Endpoint | API responds to /health with status 200
- [x] OT-P0-002 | Database Connection | API connects to PostgreSQL and performs CRUD operations
- [x] OT-P0-003 | Foliage Data API | Retrieve current foliage status for regions via REST endpoints
- [x] OT-P0-004 | Weather Integration | Fetch and store weather data for predictions via direct API/CLI flows (n8n workflows removed)
- [x] OT-P0-005 | Basic Prediction Engine | Generate foliage peak predictions using Ollama AI with fallback logic
- [x] OT-P0-006 | Interactive Map UI | Display regions with foliage status overlays using Leaflet.js
- [x] OT-P0-007 | Lifecycle Management | setup/develop/test/stop commands work properly

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | User Reports | Accept and display crowd-sourced foliage reports via GET/POST endpoints
- [x] OT-P1-002 | Time Slider | Navigate through past/present/future foliage states in UI
- [x] OT-P1-003 | Trip Planning | Save and manage multi-region trip plans with backend storage
- [x] OT-P1-004 | Photo Gallery | Display user-submitted photos by region and date with filtering

### 🟢 P2 – Future / expansion

- [x] OT-P2-001 | AI Predictions | Use Ollama llama3.2:latest for advanced pattern analysis
- [x] OT-P2-002 | Mobile Responsive | Optimize UI for mobile devices with responsive CSS
- [x] OT-P2-003 | Export Features | Download predictions and trip plans in CSV and JSON formats

## 🧱 Tech Direction Snapshot

**UI Stack**: JavaScript/Node.js frontend with Leaflet.js for interactive mapping, autumn-themed responsive design

**API Stack**: Go service handling data operations, predictions, and user reports

**Data Storage**:
- PostgreSQL for historical foliage data, weather records, user reports, and trip plans
- Redis for real-time data caching and performance optimization

**AI Integration**: Direct Ollama API calls for foliage peak predictions (llama3.2:latest model) with fallback to latitude-based typical peak weeks

**Integration Strategy**: Direct API/CLI flows for weather data collection; Ollama API calls per shared-workflows protocol

**Non-Goals**:
- Real-time satellite imagery processing (use pre-processed regional data)
- Mobile native apps (focus on responsive web UI)
- Social networking features beyond basic photo sharing

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- PostgreSQL (port 5433) – historical data and user content storage
- Redis – response caching and session management
- Ollama – AI-powered foliage peak predictions
- Automation handled via direct API calls and scheduled jobs (n8n workflows removed)

**Optional Resources**:
- Browserless – weather data scraping fallback

**Scenario Dependencies**: None (standalone scenario)

**Risks**:
- Ollama availability affects prediction quality (mitigated with fallback logic)
- Weather API rate limits may require caching strategy
- Foliage data accuracy depends on crowd-sourced report volume

**Launch Sequencing**:
1. Deploy P0 features for core tracking and prediction
2. Enable P1 user engagement features (reports, photos, trip planning)
3. Optimize P2 features for mobile and data export use cases

**Performance Expectations**:
- API response time: <500ms for data queries
- UI load time: <3s initial load
- Prediction generation: <5s per region
- Map rendering: <2s for overlay updates

## 🎨 UX & Branding

**Visual Palette**: Warm autumn theme with rich oranges, deep reds, golden yellows, and earthy browns creating a cozy seasonal atmosphere

**Typography Tone**: Clean, readable sans-serif fonts optimized for map overlays and data-dense trip planning views

**Motion Language**: Smooth transitions for time slider interactions and map pan/zoom; subtle fade effects for foliage status updates

**Accessibility Commitments**:
- WCAG 2.1 Level AA compliance for color contrast
- Keyboard navigation support for all interactive map features
- Screen reader compatibility for foliage status and prediction data
- Mobile-responsive design tested on 375x667 viewport

**Voice/Personality**: Informative and enthusiastic guide for autumn adventures, balancing scientific prediction data with the wonder of seasonal change

**Experience Promise**: Users feel confident planning their foliage trips with AI-backed predictions, real-time reports, and beautiful map visualization that captures the magic of fall
