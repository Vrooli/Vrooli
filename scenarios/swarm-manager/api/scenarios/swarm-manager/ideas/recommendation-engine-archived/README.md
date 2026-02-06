# 🎯 Recommendation Engine

Universal personalization intelligence system that provides hybrid recommendations across all Vrooli scenarios using semantic embeddings and behavioral analytics.

## 🌟 What This Scenario Adds to Vrooli

The Recommendation Engine creates a **compound intelligence network** where every user interaction across all scenarios improves recommendations for everyone. This isn't just a feature - it's a **force multiplier** that makes every future scenario exponentially smarter.

### The Network Effect
- **Cross-Domain Learning**: Shopping behaviors inform content preferences and vice versa
- **Zero-Shot Recommendations**: Can recommend items in new scenarios based on learned preferences from other domains  
- **Permanent Intelligence**: Every interaction becomes permanent knowledge that improves the system forever

### Business Value
- **$25K-75K per deployment**: Adds enterprise-grade personalization to entire platforms
- **Immediate ROI**: Scenarios gain personalization capabilities without building custom recommendation logic
- **Market Differentiator**: First truly cross-domain recommendation system in automation platforms

## 🏗️ Architecture

### Hybrid Recommendation Approach
- **Semantic Similarity**: Qdrant vector embeddings for content-based recommendations
- **Collaborative Filtering**: PostgreSQL behavioral analytics for user-based recommendations  
- **Contextual Intelligence**: Time, device, session data for situational recommendations

### Core Components
- **Go API**: High-performance recommendation service (<100ms response time)
- **PostgreSQL**: User interactions, item metadata, recommendation analytics
- **Qdrant**: Vector database for semantic similarity search
- **Management UI**: Dashboard for monitoring connected scenarios
- **CLI Interface**: Command-line tools for integration and administration

## 🚀 Quick Start

### 1. Run the Scenario
```bash
# Start the recommendation engine
vrooli scenario run recommendation-engine

# Access the services
# Management Dashboard: http://localhost:3000
# API Documentation: http://localhost:8080/docs  
# API Health: http://localhost:8080/health
```

### 2. Test the CLI
```bash
# Check system status
recommendation-engine status --verbose

# Get help for any command
recommendation-engine help ingest
```

### 3. Ingest Sample Data
```bash
# Use the CLI to add items and user interactions
echo '[
  {
    "external_id": "product-1",
    "title": "Wireless Headphones", 
    "description": "High-quality bluetooth headphones",
    "category": "electronics",
    "metadata": {"price": 89.99, "brand": "TechSound"}
  }
]' > items.json

recommendation-engine ingest my-store --items-file items.json
```

### 4. Get Recommendations
```bash
# Get personalized recommendations
recommendation-engine recommend user-123 my-store --limit 5

# Find similar items
recommendation-engine similar product-1 my-store --threshold 0.8
```

## 📊 Integration Guide

### For Scenario Developers

#### 1. Ingest Your Data
```bash
# Items: products, articles, videos, etc.
curl -X POST http://localhost:8080/api/v1/recommendations/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_id": "your-scenario",
    "items": [
      {
        "external_id": "item-1",
        "title": "Your Item Title",
        "description": "Item description for embedding generation",
        "category": "your-category",
        "metadata": {"custom": "fields"}
      }
    ]
  }'
```

#### 2. Track User Interactions
```bash
# Views, likes, purchases, shares, etc.
curl -X POST http://localhost:8080/api/v1/recommendations/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "scenario_id": "your-scenario", 
    "interactions": [
      {
        "user_id": "user-123",
        "item_external_id": "item-1", 
        "interaction_type": "purchase",
        "interaction_value": 5.0,
        "context": {"device": "mobile", "session_id": "abc123"}
      }
    ]
  }'
```

#### 3. Get Recommendations
```bash
# Personalized recommendations for users
curl -X POST http://localhost:8080/api/v1/recommendations/get \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "scenario_id": "your-scenario",
    "limit": 10,
    "algorithm": "hybrid",
    "context": {"page": "homepage", "time_of_day": "evening"}
  }'
```

### Via CLI Integration
```bash
# Add to your scenario's startup script
recommendation-engine ingest your-scenario \
  --items-file data/items.json \
  --interactions-file data/interactions.json

# In your application code
recommendations=$(recommendation-engine recommend $USER_ID your-scenario --json)
```

## 🔧 API Reference

### Endpoints

#### `POST /api/v1/recommendations/ingest`
Ingest items and user interactions from scenarios.

**Request:**
```json
{
  "scenario_id": "string",
  "items": [
    {
      "external_id": "string",
      "title": "string", 
      "description": "string",
      "category": "string",
      "metadata": {}
    }
  ],
  "interactions": [
    {
      "user_id": "string",
      "item_external_id": "string",
      "interaction_type": "string",
      "interaction_value": 1.0,
      "context": {}
    }
  ]
}
```

#### `POST /api/v1/recommendations/get`
Get personalized recommendations for a user.

**Request:**
```json
{
  "user_id": "string",
  "scenario_id": "string", 
  "limit": 10,
  "algorithm": "hybrid",
  "exclude_items": ["item-1", "item-2"],
  "context": {}
}
```

**Response:**
```json
{
  "recommendations": [
    {
      "item_id": "string",
      "external_id": "string", 
      "title": "string",
      "description": "string",
      "confidence": 0.85,
      "reason": "Users who liked similar items",
      "category": "string"
    }
  ],
  "algorithm_used": "hybrid",
  "generated_at": "2025-01-01T12:00:00Z"
}
```

#### `POST /api/v1/recommendations/similar`
Find items similar to a given item.

**Request:**
```json
{
  "item_external_id": "string",
  "scenario_id": "string",
  "limit": 10,
  "threshold": 0.7
}
```

### Health & Status
- `GET /health` - System health check
- `GET /docs` - API documentation

## 🖥️ Management Dashboard

Access the dashboard at `http://localhost:3000` to monitor:

- **System Status**: API, database, and vector database health
- **Connected Scenarios**: All scenarios using the recommendation engine
- **Metrics**: Total recommendations, users, items, interactions
- **Performance**: Algorithm effectiveness and response times

## 🧪 Testing

### Run All Tests
```bash
vrooli scenario test recommendation-engine
```

### Test Categories
```bash
# Structure and dependency validation
vrooli test --structure

# Resource health checks  
vrooli test --resources

# Integration tests
vrooli test --integration

# Performance validation
vrooli test --performance
```

### Manual Testing
```bash
# Test CLI commands
recommendation-engine status
recommendation-engine version --json

# Test API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/docs
```

## 🔄 Development Workflow

### Local Development
```bash
# Build and test the API
cd api
go mod download
go build -o recommendation-engine-api .
go test ./...

# Start development server
./recommendation-engine-api

# Install CLI globally
cd ../cli  
./install.sh

# Test CLI
recommendation-engine help
```

### Database Management
```bash
# Apply schema changes
psql $DATABASE_URL -f initialization/storage/postgres/schema.sql

# Load seed data
psql $DATABASE_URL -f initialization/storage/postgres/seed.sql

# Check database status
recommendation-engine status --verbose
```

## 📈 Performance & Scaling

### Performance Targets
- **API Response Time**: <100ms for 95% of recommendation requests
- **System Throughput**: 1000 requests/second sustained
- **Embedding Generation**: <5 seconds per 1000 items
- **Resource Usage**: <2GB memory, <50% CPU under normal load

### Scaling Considerations
- **Database**: PostgreSQL connection pooling and read replicas
- **Vector Search**: Qdrant clustering for large embedding collections
- **Caching**: Redis integration for frequently requested recommendations
- **Load Balancing**: Multiple API instances behind load balancer

## 🛠️ Configuration

### Environment Variables
```bash
# API Configuration
RECOMMENDATION_API_PORT=8080
RECOMMENDATION_UI_PORT=3000

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432  
POSTGRES_USER=postgres
POSTGRES_PASSWORD=
POSTGRES_DB=vrooli

# Vector Database
QDRANT_HOST=localhost
QDRANT_PORT=6334

# Optional Redis Cache
REDIS_HOST=localhost
REDIS_PORT=6379
```

### Scenario Configuration
Edit `.vrooli/service.json` to customize:
- Resource dependencies (postgres, qdrant, redis)
- Port assignments
- Initialization scripts
- Health check settings

## 🔍 Troubleshooting

### Common Issues

#### API Won't Start
```bash
# Check if ports are available
ss -tulpn | grep :8080

# Verify database connection
recommendation-engine status --verbose

# Check logs
tail -f api/logs/recommendation-engine.log
```

#### No Recommendations Returned
```bash
# Verify data has been ingested
recommendation-engine status --json | jq '.metrics'

# Check user interactions exist
psql $DATABASE_URL -c "SELECT COUNT(*) FROM user_interactions;"

# Test with seed data
psql $DATABASE_URL -f initialization/storage/postgres/seed.sql
```

#### Vector Database Issues
```bash
# Check Qdrant health
curl http://localhost:6334/health

# Verify collection exists
curl http://localhost:6334/collections/item-embeddings

# Rebuild embeddings
recommendation-engine rebuild-embeddings --force
```

### Debug Mode
```bash
# Enable debug logging
DEBUG=1 recommendation-engine status

# Verbose API logs  
RECOMMENDATION_LOG_LEVEL=debug ./api/recommendation-engine-api
```

## 🤝 Contributing

This scenario is designed to be **permanent intelligence** that improves with every interaction. When contributing:

1. **Maintain Backward Compatibility**: Existing integrations must continue working
2. **Preserve Performance**: Keep recommendation responses under 100ms
3. **Enhance Intelligence**: Focus on features that make recommendations smarter
4. **Document Changes**: Update API docs and integration examples

### Key Areas for Enhancement
- **Advanced Algorithms**: Implement deep learning models for better recommendations
- **Real-time Streaming**: WebSocket support for live recommendation updates  
- **A/B Testing**: Built-in experimentation framework for algorithm optimization
- **Cross-Platform**: Federated learning across multiple Vrooli installations

## 📚 Resources

- **PRD**: See `PRD.md` for complete product requirements
- **API Docs**: Available at `http://localhost:8080/docs` when running
- **Database Schema**: See `initialization/storage/postgres/schema.sql`
- **CLI Help**: Run `recommendation-engine help <command>` for detailed usage

## 🎯 What's Next

The Recommendation Engine enables entirely new categories of scenarios:

1. **Smart Content Curation**: News aggregators with perfect personalization
2. **Dynamic Commerce**: E-commerce stores that recommend based on lifestyle preferences
3. **Context-Aware Productivity**: Tools that adapt to your work patterns
4. **Personalized Automation**: Workflows that optimize based on user behavior
5. **Social Discovery**: Platforms that connect users with shared interests

Every new scenario that integrates with the Recommendation Engine makes ALL scenarios smarter. This is **compound intelligence** in action - the system literally cannot forget how to solve problems, only get better at solving them.

---

**Built with ❤️ for the Vrooli ecosystem** | [GitHub](https://github.com/vrooli/vrooli/scenarios/recommendation-engine) | [Documentation](./PRD.md)