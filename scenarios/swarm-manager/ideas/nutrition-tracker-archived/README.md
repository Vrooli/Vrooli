# NutriTrack - Smart Nutrition Companion

## Overview
NutriTrack is a smart nutrition tracking application that helps users monitor their daily caloric intake, track macronutrients (protein, carbs, fat), and receive AI-powered meal suggestions tailored to their nutritional goals.

## Why It's Useful
- **Personalized Tracking**: Track calories and macros with intelligent daily/weekly goal setting
- **AI Meal Suggestions**: Get meal recommendations that fit within your remaining nutritional targets
- **Smart Analysis**: Uses AI to analyze food items and estimate nutritional content
- **Visual Progress**: Beautiful charts and progress indicators to keep you motivated

## Dependencies & Integration
This scenario demonstrates Vrooli's capability to:
- Use shared n8n workflows (ollama, embedding-generator, rate-limiter)
- Integrate multiple resources (PostgreSQL, Redis, Qdrant, Ollama)
- Provide both API and CLI interfaces for maximum flexibility
- Create consumer-friendly UIs with unique, vibrant design

## UX Design Philosophy
**Vibrant & Motivating**: The UI features a colorful, energetic design with:
- Gradient purple background with floating food emojis for a playful touch
- Bold, friendly typography using Poppins and Nunito fonts
- Color-coded macro tracking (green for protein, orange for carbs, pink for fats)
- Smooth animations and hover effects to make tracking feel fun, not like a chore
- Mobile-responsive design for on-the-go tracking

The design aims to make nutrition tracking feel less like work and more like a fun, engaging daily habit.

## Key Features
1. **Meal Logging**: Quick and easy meal entry with auto-complete suggestions
2. **Macro Tracking**: Visual breakdown of protein, carbohydrates, and fats
3. **AI Suggestions**: Get personalized meal ideas based on remaining targets
4. **Progress Charts**: Daily and weekly progress visualization
5. **Food Database**: Built-in database with common foods and their nutritional values

## Technical Architecture
- **API**: Go-based REST API for high performance
- **Storage**: PostgreSQL for user data, Redis for caching, Qdrant for semantic search
- **AI**: Ollama for meal suggestions and food analysis
- **Automation**: n8n workflows for complex nutritional calculations

## Future Enhancements
This scenario could integrate with:
- `study-buddy`: For learning about nutrition and healthy eating habits
- `personal-digital-twin`: To create personalized nutrition AI based on user preferences
- `morning-vision-walk`: For daily nutrition goal setting and meal planning
- `stream-of-consciousness-analyzer`: For voice-based meal logging

## CLI Usage
```bash
nutrition-tracker log "grilled chicken salad with olive oil"
nutrition-tracker suggest lunch --calories 500 --protein 30
nutrition-tracker progress --week
```

## API Endpoints
- `POST /api/meals` - Log a new meal
- `GET /api/meals` - Get meal history
- `POST /api/suggest` - Get AI meal suggestions
- `GET /api/progress` - Get nutritional progress data