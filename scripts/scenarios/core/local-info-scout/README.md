# Local Info Scout

## Overview
Local Info Scout is an intelligent location-based information discovery tool that helps users find nearby places, services, and points of interest through natural language queries and interactive maps.

## Purpose
This scenario enables local discovery by:
- Finding restaurants, stores, services based on specific criteria
- Providing real-time availability and hours information
- Offering personalized recommendations based on preferences
- Showing distance, ratings, and reviews from multiple sources

## Key Features
- **Natural Language Search**: "Find vegan restaurants within 2 miles"
- **Multi-Source Aggregation**: Combines data from maps, reviews, and local directories
- **Smart Filtering**: Filter by distance, ratings, price, hours, accessibility
- **Real-Time Updates**: Current hours, availability, and wait times
- **Discovery Mode**: Explore categories like "hidden gems" or "new openings"

## Dependencies
- **Shared Workflows**: 
  - `ollama.json` - For understanding natural language queries
  - `rate-limiter.json` - For managing API rate limits
- **Resources**: ollama, n8n, postgres, redis, browserless

## UX Design
Clean, map-focused interface with exploration-friendly design:
- Interactive map as the centerpiece
- Card-based results with photos and key info
- Smooth animations and transitions
- Mobile-first responsive design
- Light theme with accent colors for categories (green for parks, orange for food, etc.)

## Use Cases
- Finding specific services: "Where can I buy cat bowls nearby?"
- Restaurant discovery: "Vegan restaurants open now"
- Emergency needs: "24-hour pharmacy near me"
- Activity planning: "Kid-friendly activities this weekend"
- Local exploration: "Historic sites within walking distance"

## Integration Points
- Powers location features in other scenarios
- Can be called by `personal-relationship-manager` for gift shopping
- Integrates with `travel-map-filler` for tourist destinations
- Used by `morning-vision-walk` for finding walking routes