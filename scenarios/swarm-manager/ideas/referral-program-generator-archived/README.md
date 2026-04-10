# Referral Program Generator

**Transform any Vrooli scenario into a revenue-generating business with intelligent referral programs.**

The Referral Program Generator is a permanent business intelligence capability that automatically analyzes scenarios, extracts branding and pricing information, and generates complete referral/affiliate systems with optimized commission structures.

## 🌟 Features

- **🎯 Smart Analysis**: Automatically extract branding, pricing, and structural information from any scenario
- **⚡ Instant Generation**: Create complete referral programs with AI-optimized commission rates
- **🤖 Auto Implementation**: Deploy referral logic using resource-claude-code integration
- **📊 Performance Tracking**: Monitor referral performance and optimize conversion rates
- **🔄 Cross-Scenario Networks**: Enable scenarios to promote each other for compound growth
- **💰 Revenue Optimization**: Compound business intelligence that improves with every program

## 🚀 Quick Start

### 1. Setup
```bash
# Install and configure
./scripts/manage.sh setup --yes yes

# Start the scenario
vrooli scenario run referral-program-generator
```

### 2. Access the Interface
- **Dashboard**: http://localhost:3000
- **API**: http://localhost:8080/health
- **CLI**: `referral-program-generator --help`

### 3. Generate Your First Referral Program

**Via Web Interface:**
1. Navigate to http://localhost:3000
2. Click "Analyze Scenario" 
3. Enter your scenario path
4. Generate and implement the referral program

**Via CLI:**
```bash
# Analyze a scenario
referral-program-generator analyze ../my-scenario

# Generate referral program
referral-program-generator generate analysis-result.json

# Implement automatically
referral-program-generator implement --auto <program-id> ../my-scenario
```

**Via API:**
```bash
# Analyze scenario
curl -X POST http://localhost:8080/api/v1/referral/analyze \
  -H "Content-Type: application/json" \
  -d '{"mode": "local", "scenario_path": "../my-scenario"}'

# Generate program  
curl -X POST http://localhost:8080/api/v1/referral/generate \
  -H "Content-Type: application/json" \
  -d '{"analysis_data": {...}}'
```

## 🏗️ Architecture

### Core Components

- **Analysis Engine** (`scripts/analyze-scenario.sh`): Extracts branding, pricing, and structural information
- **Pattern Generator** (`lib/referral-pattern-template.sh`): Creates standardized referral implementations
- **Claude Code Integration** (`scripts/claude-code-integration.sh`): Automates deployment via resource-claude-code
- **Auth Integration** (`lib/auth-integration.go`): Integrates with scenario-authenticator for user management
- **REST API** (`api/main.go`): Provides programmatic access to all functionality
- **Web Dashboard** (`ui/`): React-based interface for managing referral programs
- **CLI Tool** (`cli/referral-program-generator`): Command-line interface for automation

### Resource Dependencies

**Required:**
- `postgres`: Referral tracking, commission calculations, analytics storage
- `scenario-authenticator`: User account management and referral profiles

**Optional:**
- `resource-claude-code`: Automated implementation in target scenarios
- `browserless`: Web scraping for deployed scenario analysis
- `qdrant`: Semantic search of referral patterns and optimization

## 🎨 How It Works

### 1. Intelligent Analysis
The system analyzes scenarios to extract:
- **Branding**: Colors, fonts, logos, brand name
- **Pricing**: Models (subscription/one-time/freemium), tiers, rates
- **Structure**: API/UI frameworks, database schema, existing referral logic

### 2. Optimized Generation
Based on analysis, it generates:
- **Commission Rates**: AI-optimized based on pricing model and industry standards
- **Branded Materials**: Landing pages, email templates matching scenario aesthetics
- **Tracking Systems**: Unique codes, analytics, fraud detection
- **Database Schema**: Complete referral tracking infrastructure

### 3. Automated Implementation
Using resource-claude-code, it automatically:
- **Deploys Code**: Adds standardized referral logic to target scenarios
- **Updates Database**: Migrates schema with referral tables
- **Integrates UI**: Adds referral dashboard components
- **Validates**: Tests end-to-end functionality

## 📊 Business Intelligence

This scenario creates **permanent business intelligence** that compounds over time:

- **Pattern Recognition**: Learns optimal commission rates across different verticals
- **Conversion Optimization**: Identifies best-performing UI patterns and copy
- **Cross-Scenario Synergy**: Enables scenarios to promote each other
- **Fraud Detection**: Builds sophisticated anti-fraud heuristics
- **Revenue Attribution**: Tracks complex multi-scenario customer journeys

## 🔧 Configuration

### Commission Rates (Defaults)
- **Subscription Models**: 30% (recurring revenue)
- **One-Time Purchases**: 10% 
- **Freemium Models**: 20%

### Branding Adaptation
- **SaaS Tools**: Professional, conversion-focused design
- **Consumer Apps**: Fun, engaging, social sharing emphasis
- **B2B Platforms**: Trust indicators, enterprise feel

### Fraud Protection
- Rate limiting (100 clicks/IP/hour)
- Conversion validation and duplicate detection
- IP tracking and behavioral analysis

## 🧪 Testing

```bash
# Run all tests
vrooli scenario test referral-program-generator

# Test specific components
./test/test-analysis.sh
./test/test-api-endpoints.sh
./test/test-cli-commands.sh

# Validate scenario structure
vrooli scenario validate referral-program-generator
```

## 📈 Usage Examples

### E-Commerce Scenario
```bash
# Analyze e-commerce app
referral-program-generator analyze ../my-store

# Generate with higher commission (competitive market)
referral-program-generator generate analysis.json --commission-rate 0.15

# Deploy with full automation
referral-program-generator implement --auto <program-id> ../my-store
```

### SaaS Application
```bash
# Analyze SaaS scenario
referral-program-generator analyze ../my-saas-app

# Generate with subscription optimization  
referral-program-generator generate analysis.json

# Preview changes before deployment
referral-program-generator implement --preview <program-id> ../my-saas-app
```

### Cross-Scenario Network
```bash
# Generate referral programs for multiple scenarios
for scenario in scenario-a scenario-b scenario-c; do
  referral-program-generator analyze ../$scenario
  referral-program-generator generate analysis-$scenario.json
  referral-program-generator implement --auto <program-id> ../$scenario
done
```

## 🔗 Integration

### With Other Scenarios
- **saas-billing-hub**: Payment processing and commission payouts
- **email-campaign-manager**: Automated referral marketing sequences  
- **analytics-dashboard**: Performance tracking and insights
- **seo-optimizer**: Landing page optimization for better conversion

### With External Services
- **Stripe/PayPal**: Commission payment processing
- **Email Services**: Automated referral communications
- **Analytics Platforms**: Advanced conversion tracking

## 🚨 Security & Compliance

- **Data Protection**: PII encryption for referral user data
- **Fraud Prevention**: Multi-layer validation and rate limiting
- **Audit Trails**: Complete transaction and commission logging
- **Compliance**: Affiliate marketing legal requirements (FTC guidelines)

## 💡 Advanced Features

### Multi-Tier Programs
```json
{
  "tiers": {
    "bronze": { "threshold": 10, "rate": 0.20 },
    "silver": { "threshold": 50, "rate": 0.25 },
    "gold": { "threshold": 100, "rate": 0.30 }
  }
}
```

### A/B Testing
```bash
# Generate multiple variants
referral-program-generator generate analysis.json --variant landing-page-a
referral-program-generator generate analysis.json --variant landing-page-b
```

### Custom Branding Override
```json
{
  "custom_branding": {
    "colors": {
      "primary": "#custom-color",
      "secondary": "#custom-secondary"
    },
    "fonts": ["Custom Font", "fallback"],
    "logo_path": "/custom/logo.png"
  }
}
```

## 🔄 Maintenance

### Database Cleanup
```sql
-- Remove old tracking events (keep 1 year)
DELETE FROM referral_events WHERE created_at < NOW() - INTERVAL '1 year';

-- Archive completed commissions
INSERT INTO commissions_archive SELECT * FROM commissions WHERE status = 'paid' AND paid_date < NOW() - INTERVAL '6 months';
```

### Performance Optimization
```bash
# Rebuild database indexes
psql -c "REINDEX DATABASE vrooli;"

# Update referral statistics
referral-program-generator update-stats --all
```

## 📚 API Documentation

### Analysis Endpoint
```
POST /api/v1/referral/analyze
{
  "mode": "local|deployed",
  "scenario_path": "string",
  "url": "string"
}
```

### Generation Endpoint
```
POST /api/v1/referral/generate
{
  "analysis_data": {...},
  "commission_rate": 0.20,
  "custom_branding": {...}
}
```

### Implementation Endpoint
```
POST /api/v1/referral/implement
{
  "program_id": "uuid",
  "scenario_path": "string", 
  "auto_mode": boolean
}
```

## 🤝 Contributing

This scenario follows Vrooli development patterns:
- **Code Style**: Follow existing scenario conventions
- **Testing**: All features must have comprehensive tests
- **Documentation**: Update README and API docs for changes
- **Security**: Never commit secrets or break existing functionality

## 📜 License

This scenario is part of the Vrooli project and follows the same licensing terms.

---

**🎯 Ready to monetize your scenarios?** Start with `vrooli scenario run referral-program-generator` and transform your ideas into revenue-generating businesses in minutes!

**💡 Remember**: Every referral program you create becomes permanent business intelligence that makes all future programs smarter and more profitable.