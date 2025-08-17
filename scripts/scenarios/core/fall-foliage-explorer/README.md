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
- Region-specific forecasts based on weather patterns
- Photo galleries from previous years
- Trip planning suggestions based on foliage timing
- Crowd-sourced reports from users in different regions

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