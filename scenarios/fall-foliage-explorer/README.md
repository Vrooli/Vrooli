# Fall Foliage Explorer

## Overview
An interactive, data-driven application for tracking and forecasting fall foliage peaks across North America. Combines real-time weather data, historical patterns, and predictive modeling to help travelers plan the perfect autumn trip.

## Purpose
- **Travel Planning**: Help users find peak foliage times for trip planning
- **Visual Discovery**: Interactive map with time slider for exploring foliage progression
- **Data Integration**: Demonstrates Vrooli's ability to combine multiple data sources
- **Predictive Analytics**: Shows AI-powered forecasting capabilities

## Features
- Interactive map with foliage intensity overlays
- Time slider showing past, present, and predicted foliage peaks
- Region-specific forecasts based on weather patterns and AI predictions
- User-submitted foliage reports with crowd-sourced data
- Photo gallery for sharing and browsing foliage photos by region and date
- Trip planning with multi-region itineraries stored in PostgreSQL
- Ollama-powered AI predictions for peak foliage timing
- Mobile-responsive design optimized for all devices

## UX Style
**Autumn Cozy**: Warm, earthy color palette with oranges, reds, yellows, and browns. The interface should evoke the feeling of a crisp autumn day - comfortable, inviting, and nostalgic. Think cabin-in-the-woods meets modern data visualization.

## Technical Architecture
- **Data Sources**: Weather APIs, historical foliage data, user reports
- **Visualization**: Leaflet.js for interactive maps with custom foliage overlays
- **Predictions**: Ollama for processing weather patterns and generating forecasts
- **Storage**: PostgreSQL for historical data, Redis for caching current conditions

## Dependencies
- Uses shared `ollama` workflow for AI predictions
- Leverages `rate-limiter` for API call management
- Weather data integration through n8n workflows

## Revenue Model
- Freemium with basic map access
- Premium features: detailed forecasts, trip planning, photo galleries
- Potential partnerships with travel agencies and tourism boards

## Usage

### Photo Gallery
The photo gallery allows users to share and browse foliage photos:

**Sharing Photos:**
1. Navigate to "Photo Gallery" tab
2. Select the region from dropdown
3. Choose the foliage status (Not Started, Progressing, Near Peak, Peak, Past Peak)
4. Enter photo URL
5. Add optional description
6. Click "Share Photo"

**Browsing Photos:**
- Filter by region using the region dropdown
- Filter by date using the date picker
- Photos display with region name, status badge, date, and description
- Photos are organized in a responsive grid layout

### API Endpoints

#### Photo/Report Management
```bash
# Get all reports with photos for a region
curl http://localhost:17175/api/reports?region_id=1

# Submit a new photo/report
curl -X POST http://localhost:17175/api/reports \
  -H "Content-Type: application/json" \
  -d '{
    "region_id": 1,
    "foliage_status": "peak",
    "photo_url": "https://example.com/photo.jpg",
    "description": "Beautiful colors!"
  }'
```

### Export Features

Export your foliage predictions and trip plans for offline use or sharing:

**Exporting Predictions:**
1. Navigate to "Regions" tab
2. Click "Export CSV" for spreadsheet format with all region data
3. Click "Export JSON" for structured data format suitable for programmatic use
4. Files include: region name, state, coordinates, elevation, current status, color intensity, and typical peak week

**Exporting Trip Plans:**
1. Navigate to "Trip Planner" tab
2. Create and save trip plans
3. Click "Export CSV" to download all saved trips as a spreadsheet
4. Click "Export JSON" for structured trip data with full region details
5. Files include: trip name, dates, selected regions with coordinates and details

### Mobile Access
The UI is fully responsive and optimized for mobile devices. All features including the map, photo gallery, trip planner, export buttons, and navigation work seamlessly on mobile viewports.