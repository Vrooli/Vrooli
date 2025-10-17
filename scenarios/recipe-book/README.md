# Recipe Book Scenario

A warm, inviting household recipe management system with AI-powered search and generation capabilities. This scenario provides the culinary intelligence foundation for Vrooli household deployments.

## 🍳 Overview

Recipe Book transforms recipe management from simple storage to an intelligent culinary assistant that learns family preferences, suggests meals based on available ingredients, and enables seamless recipe sharing within households.

## ✨ Key Features

- **Semantic Recipe Search** - Find recipes by mood, ingredients, or dietary needs
- **AI Recipe Generation** - Create new recipes from prompts or available ingredients  
- **Multi-Tenant Support** - Private, shared, and public recipe visibility
- **Beautiful UI** - Cozy cookbook aesthetic with hand-drawn style
- **Full API/CLI** - Complete programmatic access for other scenarios
- **Recipe Modification** - Transform recipes to be vegan, keto, gluten-free, etc.
- **Nutrition Tracking** - Automatic nutritional information calculation
- **Family Sharing** - Share recipes with specific household members

## 🎨 Design Philosophy

The UI embraces a "grandma's recipe box" aesthetic with:
- Warm terracotta and sage green color palette
- Handwritten fonts for titles (Caveat)
- Recipe cards with worn edges
- Playful cooking animations
- Chef hat rating system

## 🔗 Integration Points

### Provides To
- **nutrition-tracker** - Actual recipes cooked with accurate nutrition
- **grocery-optimizer** - Ingredient lists for smart shopping
- **date-night-generator** - Food preference profiles
- **meal-prep-assistant** - Batch cooking optimization

### Consumes From
- **scenario-authenticator** - User authentication and profiles
- **contact-book** - Dietary restrictions and preferences
- **ollama** - AI recipe generation (via n8n workflow)

## 🚀 Quick Start

```bash
# Start the scenario
vrooli scenario run recipe-book

# Access the UI
open http://localhost:3250

# Use the CLI
recipe-book list
recipe-book search "chocolate cake"
recipe-book generate "healthy dinner for two"
```

## 📡 API Endpoints

- `GET /api/v1/recipes` - List recipes with filtering
- `POST /api/v1/recipes/search` - Semantic recipe search
- `POST /api/v1/recipes/generate` - AI recipe generation
- `POST /api/v1/recipes/{id}/modify` - Modify recipe for dietary needs
- `POST /api/v1/recipes/{id}/cook` - Mark as cooked and rate

## 🎯 Business Value

This scenario enables:
- **Family Bonding** - Share and preserve family recipes
- **Health Improvement** - Track nutrition and dietary goals
- **Time Savings** - 2-3 hours/week on meal planning
- **Cost Reduction** - 20% grocery savings through smart planning
- **Revenue Potential** - $15K-30K per deployment as subscription service

## 🧬 Future Evolution

Version 2.0 will add:
- Voice-controlled cooking mode
- Meal planning calendar
- Recipe video support
- Smart appliance integration
- Social recipe network

## 📝 Configuration

The scenario uses these environment variables:
- `API_PORT` - API server port (default: 3250)
- `POSTGRES_HOST/PORT/USER/PASSWORD/DB` - Database connection
- `RECIPE_BOOK_USER` - Default user ID for CLI

## 🧪 Testing

Run the test suite:
```bash
vrooli scenario test recipe-book
```

This validates:
- All P0 requirements from PRD
- API endpoint functionality
- CLI command execution
- Database schema initialization
- UI accessibility
- Performance targets (<200ms API, <500ms search)