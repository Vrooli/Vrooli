# Real Estate Platform Template

A comprehensive real estate platform with property listings, lead management, virtual tours, and market analytics.

## Features

- 🏠 **Property Management**: Complete listing system with MLS integration
- 👥 **Lead Management**: CRM with automated nurturing and scoring
- 📸 **Virtual Tours**: 360° tours and virtual staging capabilities
- 📊 **Market Analytics**: Price tracking and neighborhood insights
- 🤝 **Agent Tools**: Commission tracking and performance dashboards
- 📱 **Mobile Ready**: Responsive design for on-the-go access
- 🔍 **Smart Search**: AI-powered property recommendations
- 📧 **Automated Marketing**: Email campaigns and property alerts

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Property Portal                           │
│              (Web & Mobile Applications)                     │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Integration Layer                       │
│                    (MLS, Maps, Mortgage)                     │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Resource Layer                          │
├──────────────┬──────────────┬──────────────┬───────────────┤
│  PostgreSQL  │     MinIO    │    Qdrant    │   QuestDB     │
│  (Listings)  │   (Media)    │   (Search)   │  (Analytics)  │
├──────────────┼──────────────┼──────────────┼───────────────┤
│   Node-RED   │     n8n      │   Windmill   │    Vault      │
│   (Leads)    │    (CRM)     │  (Reports)   │  (Secrets)    │
└──────────────┴──────────────┴──────────────┴───────────────┘
```

## Quick Start

### 1. Set Environment Variables

Create `.env` file:
```bash
# Required
DB_USER=realestate_user
DB_PASSWORD=secure_password_here
MLS_API_KEY=your_mls_api_key
GOOGLE_MAPS_KEY=your_google_maps_key
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=notifications@youragency.com
SMTP_PASSWORD=app_specific_password

# Optional but Recommended
MLS_API_SECRET=your_mls_secret
MLS_ENDPOINT=https://api.mls.com/v2
GOOGLE_GEOCODING_KEY=your_geocoding_key
TWILIO_SID=your_twilio_sid
TWILIO_TOKEN=your_twilio_token
TWILIO_PHONE=+1234567890
MORTGAGE_CALC_KEY=mortgage_api_key
```

### 2. Deploy Template

```bash
# Copy template
cp scenario.json ~/.vrooli/scenarios.json

# Source environment variables
source .env

# Deploy all resources
../../index.sh --action inject-all
```

### 3. Verify Deployment

```bash
# Check status
../../index.sh --action status

# Test database
psql -h localhost -p 5432 -U realestate_user -d realestate

# Access services
# Node-RED: http://localhost:1880
# n8n: http://localhost:5678
# MinIO: http://localhost:9001
```

## Database Schema

### Core Tables

**properties**
- Property listings with full details
- Address and location data
- Pricing and status tracking
- Features and amenities

**agents**
- Agent profiles and credentials
- Performance metrics
- Commission tracking
- Territory assignments

**leads**
- Contact information
- Property preferences
- Activity tracking
- Lead scoring

**showings**
- Scheduled property viewings
- Agent assignments
- Feedback collection
- Follow-up tracking

**transactions**
- Offers and contracts
- Closing details
- Commission calculations
- Document tracking

**neighborhoods**
- Area demographics
- School information
- Amenity data
- Market statistics

### Sample Data

The template includes:
- 500 sample property listings
- 50 agent profiles
- 200 test leads
- 100 historical transactions
- Neighborhood data for major areas

## Workflows

### Node-RED Flows

**Lead Capture**
- Website form processing
- Lead validation
- Instant notifications
- CRM integration

**Property Alerts**
- Matching algorithm
- Email notifications
- SMS alerts
- Push notifications

**Showing Scheduler**
- Calendar integration
- Conflict detection
- Reminder system
- Route optimization

### n8n Workflows

**Lead Nurturing**
- Automated email sequences
- Property recommendations
- Market updates
- Follow-up scheduling

**Market Report Generator**
- Data aggregation
- Report formatting
- Distribution lists
- Archive management

**Commission Calculator**
- Transaction processing
- Split calculations
- Report generation
- Payment tracking

### Windmill Scripts

**MLS Sync**
- Daily listing updates
- Photo downloads
- Data validation
- Change detection

**Market Analysis**
- Price trend analysis
- Inventory tracking
- Days on market
- Comparative analysis

**Lead Scoring**
- Activity weighting
- Engagement metrics
- Predictive scoring
- Priority assignment

## Property Search

### Vector Search with Qdrant

Find similar properties based on:
- Features and amenities
- Location preferences
- Price range
- Property type
- Neighborhood characteristics

### Sample Search Queries

```javascript
// Find similar properties
POST /api/properties/similar
{
  "property_id": "prop_123",
  "limit": 10,
  "filters": {
    "price_range": [300000, 500000],
    "bedrooms": [3, 4],
    "city": "Austin"
  }
}

// Search by preferences
POST /api/properties/search
{
  "query": "modern home with pool near schools",
  "filters": {
    "property_type": "single_family",
    "min_sqft": 2000
  }
}
```

## Market Analytics

### QuestDB Metrics

Real-time analytics:
- Property view tracking
- Price history
- Market trends
- Lead activity
- Agent performance

### Sample Queries

```sql
-- Average price by neighborhood
SELECT 
  zipcode,
  avg(list_price) as avg_price,
  count(*) as listings
FROM price_history
WHERE timestamp >= dateadd('m', -6, now())
GROUP BY zipcode
ORDER BY avg_price DESC;

-- Most viewed properties
SELECT 
  property_id,
  count(*) as views,
  avg(duration) as avg_duration
FROM property_views
WHERE timestamp >= dateadd('d', -30, now())
GROUP BY property_id
ORDER BY views DESC
LIMIT 20;
```

## Virtual Tours & Staging

### ComfyUI Integration

**Virtual Staging**
- Furniture placement
- Decor suggestions
- Style variations
- Before/after views

**Photo Enhancement**
- HDR processing
- Sky replacement
- Lawn enhancement
- Lighting correction

## Lead Management

### CRM Features

- **Lead Capture**: Multiple source tracking
- **Scoring**: Automated lead qualification
- **Assignment**: Round-robin or territory-based
- **Nurturing**: Drip campaigns and follow-ups
- **Analytics**: Conversion tracking and ROI

### Lead Workflow

```
Lead Source → Capture → Validation → Scoring → Assignment → Nurturing → Conversion
     ↓                                              ↓
  Analytics ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← Follow-up
```

## MLS Integration

### Sync Configuration

```javascript
// MLS sync settings
{
  "sync_interval": "2h",
  "resources": ["listings", "photos", "agents"],
  "filters": {
    "status": ["active", "pending"],
    "listing_date": "30d",
    "areas": ["downtown", "suburbs"]
  }
}
```

### Data Mapping

Map MLS fields to local schema:
- StandardFields → properties table
- CustomFields → metadata JSONB
- Photos → MinIO storage
- Agents → agents table

## Mobile Features

### Agent App Capabilities

- Property management
- Lead tracking
- Showing schedule
- Document signing
- Commission tracking
- Market reports

### Client Portal

- Saved searches
- Favorite properties
- Showing requests
- Document access
- Mortgage calculator

## Customization

### Adding Property Types

1. Update schema:
```sql
ALTER TABLE properties 
ADD COLUMN property_subtype VARCHAR(50);
```

2. Modify search vectors:
```javascript
// Add to Qdrant index
{
  "collection": "properties",
  "field": "property_subtype",
  "type": "keyword"
}
```

### Custom Workflows

Add specialized workflows:
- Rental management
- Investment analysis
- New construction
- Foreclosure tracking

### Branding

Customize appearance:
- Logo and colors
- Email templates
- Report layouts
- Portal themes

## Security

### Data Protection

- **PII Encryption**: Sensitive data encrypted at rest
- **Access Control**: Role-based permissions
- **Audit Logging**: Track all data access
- **Document Security**: Encrypted document storage
- **API Security**: Rate limiting and authentication

### Compliance

- RESPA compliance
- Fair Housing Act
- State real estate regulations
- GDPR/CCPA for client data

## Monitoring

### Performance Metrics

```bash
# Check system health
curl http://localhost:8000/health

# Database connections
psql -c "SELECT count(*) FROM pg_stat_activity"

# Storage usage
mc du local/property-images

# Lead response time
SELECT avg(response_time) FROM lead_activity
```

### Alerts

Set up monitoring for:
- New listing notifications
- Lead response SLA
- System performance
- Storage capacity
- API rate limits

## Troubleshooting

### Common Issues

**MLS Sync Failures**
```bash
# Check sync logs
docker logs windmill-worker

# Verify API credentials
curl -H "X-API-Key: $MLS_API_KEY" $MLS_ENDPOINT/test

# Manual sync trigger
./scripts/windmill/mls-sync.py --force
```

**Search Not Working**
```bash
# Check Qdrant status
curl http://localhost:6333/collections

# Rebuild search index
./tools/rebuild-search-index.sh
```

**Lead Notifications**
```bash
# Test email configuration
./test-smtp.sh

# Check Node-RED flows
curl http://localhost:1880/flows

# Verify webhook endpoints
curl -X POST http://localhost:1880/webhook/lead-capture
```

## Production Deployment

### Scaling Considerations

- Use managed PostgreSQL with read replicas
- Deploy MinIO in distributed mode
- Implement Redis caching layer
- Use CDN for property images
- Deploy workers across multiple nodes

### Backup Strategy

```bash
# Database backup
pg_dump -h localhost -U realestate_user realestate > backup.sql

# Media backup
mc mirror local/property-images /backup/images

# Configuration backup
n8n export:workflow --all > workflows.json
```

## Support

For assistance:
1. Check the [main documentation](../../README.md)
2. Review workflow logs
3. Enable debug mode for detailed output
4. Contact support with error logs