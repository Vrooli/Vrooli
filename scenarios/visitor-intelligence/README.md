# Visitor Intelligence

Real-time website visitor identification, behavioral tracking, and retention marketing automation for Vrooli scenarios.

## 🎯 Purpose

Visitor Intelligence transforms anonymous traffic into actionable visitor profiles across all Vrooli scenarios. This creates a permanent intelligence layer that remembers and learns from every interaction, enabling predictive behavior modeling and automated retention campaigns.

## 🚀 Quick Start

### Integration (Any Scenario)
```html
<!-- Add this single line to any scenario's HTML -->
<script src="http://localhost:${API_PORT}/tracker.js" data-scenario="your-scenario-name"></script>
```

### CLI Usage
```bash
# Check system status
visitor-intelligence status

# View visitor profile  
visitor-intelligence profile visitor-id-123

# Get scenario analytics
visitor-intelligence analytics my-scenario --timeframe 7d
```

### API Usage
```bash
# Track a visitor event
curl -X POST http://localhost:${API_PORT}/api/v1/visitor/track \
  -H "Content-Type: application/json" \
  -d '{
    "fingerprint": "abc123",
    "event_type": "pageview",
    "scenario": "my-app",
    "page_url": "/dashboard"
  }'
```

## 🔄 Capabilities Added to Vrooli

- **Cross-Scenario Intelligence**: Visitors tracked across ALL Vrooli scenarios
- **Behavioral Insights**: Real-time understanding of user patterns and intent  
- **Retention Automation**: Trigger marketing campaigns based on visitor actions
- **Privacy-First**: GDPR-compliant, first-party data collection only
- **API Integration**: Other scenarios can query visitor data and trigger events

## 📊 Features

### Visitor Identification (40-60% accuracy)
- JavaScript fingerprinting using browser attributes
- Session-based tracking with graceful degradation
- Cross-device correlation (future enhancement)

### Behavioral Tracking
- Page views, clicks, scroll depth, time on site
- Cart abandonment, form interactions, custom events
- Real-time session monitoring

### Analytics Dashboard  
- Live visitor activity monitoring
- Individual visitor profiles and timelines
- Scenario-specific performance metrics
- Audience segmentation and cohort analysis

### Privacy & Compliance
- GDPR/CCPA consent management
- Automatic data purging after retention period
- No third-party data sharing
- Respect for Do Not Track headers

## 🏗️ Architecture

### High-Performance Stack
- **Go API**: Sub-50ms tracking pixel responses
- **PostgreSQL**: Visitor profiles and event data
- **Redis**: Session caching and real-time processing
- **JavaScript Library**: Lightweight client-side tracking

### No n8n Dependencies
Built for speed with direct database operations instead of workflow overhead. Perfect for high-volume visitor tracking.

## 💰 Business Value

- **Revenue Potential**: $15K-50K per deployment (competitive with retention.com)
- **Cost Savings**: Eliminates need for external visitor identification services
- **Intelligence Multiplier**: Every scenario becomes smarter with visitor context
- **Compound Growth**: More scenarios = more data = better intelligence

## 🔮 Future Scenarios Enabled

1. **Retention Campaign Orchestrator** - Automated email/SMS based on behavior
2. **Personalization Engine** - Dynamic content adaptation per visitor
3. **A/B Testing Platform** - Real-time experimentation framework
4. **Intent Prediction System** - ML-powered visitor action forecasting
5. **Behavioral Analytics Hub** - Cross-scenario journey optimization

## 🛠️ Development

```bash
# Setup
vrooli scenario run visitor-intelligence

# Testing  
vrooli scenario test visitor-intelligence

# CLI Development
cd cli && ./install.sh

# API Development
cd api && go run .
```

## 📋 Requirements

- PostgreSQL (visitor data storage)
- Redis (session caching)
- Go 1.21+ (API development)
- Modern browser (JavaScript tracking)

---

**Status**: Development Ready  
**Business Value**: $15K-50K per deployment  
**Technical Impact**: Foundation for all Vrooli scenario intelligence