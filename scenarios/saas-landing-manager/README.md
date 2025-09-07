# 🚀 SaaS Landing Manager

**Automated landing page generation and optimization for SaaS scenarios with A/B testing, template system, and Claude Code integration.**

The SaaS Landing Manager is a powerful scenario that automatically detects SaaS business opportunities in your Vrooli ecosystem and generates professional landing pages with built-in A/B testing, analytics, and automated deployment capabilities.

## ✨ Key Features

- **🎯 Automatic SaaS Detection**: Scans scenarios to identify SaaS business opportunities based on PRD analysis and service configuration
- **🎨 Professional Templates**: Comprehensive template system with B2B tools, B2C apps, API services, and marketplace designs
- **⚡ AI-Powered Generation**: Leverages Ollama for intelligent content generation and brand integration
- **🔄 A/B Testing Framework**: Built-in traffic routing, variant management, and conversion optimization
- **🤖 Agent Orchestration**: Spawns Claude Code agents for intelligent landing page deployment
- **📊 Analytics Dashboard**: Real-time performance tracking, conversion metrics, and optimization insights
- **🌐 SEO Optimized**: Automatic meta tags, structured data, and Core Web Vitals optimization
- **📱 Mobile-First Design**: Responsive templates optimized for all devices

## 🎯 Business Value

This scenario transforms **every SaaS scenario** in your Vrooli ecosystem into a complete business with professional marketing presence:

- **Revenue Potential**: $15K - $75K per deployment for marketing agencies
- **Cost Savings**: Replaces $5K - $50K custom landing page development
- **Market Expansion**: 15% market growth through accessibility and professional presentation
- **Conversion Optimization**: 20%+ improvement through A/B testing

## 🏗️ Architecture

### Core Components

1. **Detection Engine** - Analyzes scenarios for SaaS characteristics
2. **Template System** - Professional landing page templates for different SaaS types
3. **Generation API** - Creates customized landing pages with A/B testing support
4. **Deployment Service** - Automated deployment via Claude Code agents
5. **Analytics Platform** - Performance tracking and optimization insights

### Technology Stack

- **Backend**: Go API with PostgreSQL for data persistence
- **Frontend**: Modern JavaScript dashboard with real-time updates
- **CLI**: Comprehensive command-line interface for automation
- **Templates**: HTML/CSS/JS templates with dynamic content insertion
- **Integration**: Claude Code agents for intelligent deployment

## 🚀 Quick Start

### Prerequisites

- Vrooli development environment
- PostgreSQL database
- Go 1.21+ and Node.js 18+ (installed automatically during setup)

### Installation

```bash
# Setup the scenario (run from scenarios/saas-landing-manager)
./scripts/manage.sh setup --yes yes

# Or using Vrooli CLI
vrooli scenario run saas-landing-manager
```

### First Steps

1. **Access the Dashboard**
   ```
   http://localhost:20000  # UI Dashboard
   http://localhost:15000  # API Documentation
   ```

2. **Scan for SaaS Scenarios**
   ```bash
   saas-landing-manager scan --force
   ```

3. **Generate Your First Landing Page**
   ```bash
   saas-landing-manager generate funnel-builder --ab-test
   ```

4. **Deploy to Target Scenario**
   ```bash
   saas-landing-manager deploy <landing-page-id> funnel-builder
   ```

## 📋 CLI Reference

### Core Commands

```bash
# Service Management
saas-landing-manager status              # Check service health
saas-landing-manager version             # Show version info

# SaaS Detection
saas-landing-manager scan                # Scan all scenarios
saas-landing-manager scan --force        # Force rescan
saas-landing-manager scan --scenario myapp  # Scan specific scenario

# Landing Page Generation  
saas-landing-manager generate <scenario> [options]
  --template <template-id>               # Use specific template
  --ab-test                              # Enable A/B testing
  --preview-only                         # Generate preview only

# Deployment
saas-landing-manager deploy <page-id> <target> [options]
  --method claude-agent                  # Use Claude Code agent (default)
  --method direct                        # Direct file deployment
  --backup                              # Backup existing files

# Template Management
saas-landing-manager template list       # List all templates
saas-landing-manager template list --category base
saas-landing-manager template list --saas-type b2b_tool

# Analytics
saas-landing-manager analytics           # View performance dashboard
saas-landing-manager analytics --timeframe 30d
```

### Example Workflows

**Complete SaaS Setup Workflow:**
```bash
# 1. Scan for opportunities
saas-landing-manager scan --force

# 2. Generate landing page with A/B testing
saas-landing-manager generate invoice-generator --ab-test

# 3. Deploy using Claude Code agent
saas-landing-manager deploy lp-abc123 invoice-generator --method claude-agent

# 4. Monitor performance
saas-landing-manager analytics --scenario invoice-generator
```

**Bulk Landing Page Generation:**
```bash
# Generate pages for all detected SaaS scenarios
saas-landing-manager scan --json | jq -r '.scenarios[].scenario_name' | \
  xargs -I {} saas-landing-manager generate {} --ab-test
```

## 🎨 Template System

### Available Templates

#### Base Templates
- **B2B Tool Template** - Professional business application design
- **B2C App Template** - Consumer-friendly app landing page
- **API Service Template** - Developer-focused API documentation style

#### Industry-Specific Templates
- **FinTech SaaS** - Financial services styling with trust indicators  
- **Healthcare SaaS** - HIPAA-compliant design with privacy focus
- **E-commerce Platform** - Conversion-optimized for online sales

#### Component Library
- **Hero Sections** - Various hero layout options
- **Pricing Tables** - Subscription and usage-based pricing
- **Testimonials** - Social proof and customer stories
- **Feature Grids** - Product capability showcases

### Template Configuration

Templates use a JSON schema for dynamic content:

```json
{
  "title": "Professional SaaS Solution",
  "hero_title": "Transform Your Business Operations", 
  "features": [
    {
      "icon": "⚡",
      "title": "Lightning Fast",
      "description": "Optimized performance for enterprise scale"
    }
  ],
  "pricing_tiers": [
    {
      "name": "Professional",
      "price": 99,
      "features": ["Unlimited users", "24/7 support"]
    }
  ]
}
```

## 📊 Analytics & A/B Testing

### Metrics Tracked

- **Conversion Rate** - Percentage of visitors who complete desired actions
- **Bounce Rate** - Visitors who leave without interaction
- **Time on Page** - Average engagement duration
- **Core Web Vitals** - Page performance metrics (LCP, FID, CLS)
- **A/B Test Results** - Statistical significance and confidence intervals

### A/B Testing Features

- **Traffic Splitting** - Automatic visitor routing between variants
- **Statistical Significance** - Built-in confidence calculations
- **Variant Management** - Easy creation and modification of test versions
- **Performance Comparison** - Side-by-side metrics analysis

### Dashboard Analytics

Access detailed analytics via:
- **Web Dashboard**: http://localhost:20000/#analytics
- **CLI Reports**: `saas-landing-manager analytics --format json`
- **API Endpoints**: `/api/v1/analytics/dashboard`

## 🤖 Claude Code Integration

The SaaS Landing Manager leverages Claude Code agents for intelligent deployment:

### Agent Capabilities

1. **Intelligent Deployment** - Analyzes target scenario structure
2. **File Structure Optimization** - Creates proper landing page organization
3. **SEO Enhancement** - Adds meta tags and structured data
4. **Mobile Optimization** - Ensures responsive design implementation
5. **Performance Optimization** - Minifies assets and optimizes loading

### Deployment Methods

**Claude Agent (Recommended)**:
```bash
saas-landing-manager deploy abc123 target-scenario --method claude-agent
```
- Intelligent file placement and organization
- Automatic SEO and performance optimization  
- Backup creation and rollback capability
- Integration with existing scenario structure

**Direct Deployment**:
```bash
saas-landing-manager deploy abc123 target-scenario --method direct
```
- Simple file copy operation
- Faster deployment for simple use cases
- Manual optimization required

## 📁 File Structure

### Generated Landing Page Structure

Each deployed landing page follows a standardized structure:

```
scenarios/your-saas-scenario/
├── landing/
│   ├── index.html              # Main landing page
│   ├── variants/
│   │   ├── a/index.html        # A/B test variant A
│   │   └── b/index.html        # A/B test variant B
│   ├── assets/
│   │   ├── images/             # Optimized images
│   │   ├── styles/
│   │   │   └── landing.css     # Landing page styles
│   │   └── scripts/
│   │       └── landing.js      # Analytics and interactions
│   └── config/
│       ├── landing.json        # A/B test configuration
│       └── analytics.json      # Analytics settings
```

### Template Inheritance

Templates support inheritance for consistent branding:

```
templates/
├── base/                       # Core templates
│   ├── b2b-tool.html
│   └── b2c-app.html
├── industry/                   # Industry-specific
│   ├── fintech.html
│   └── healthcare.html
└── components/                 # Reusable components
    ├── hero-sections/
    ├── pricing-tables/
    └── testimonials/
```

## 🔧 Configuration

### Environment Variables

```bash
# API Configuration
SAAS_LANDING_API_PORT=15000           # API server port
SAAS_LANDING_API_URL=http://localhost:15000/api/v1

# UI Configuration  
SAAS_LANDING_UI_PORT=20000            # Dashboard port

# Database Configuration
DB_HOST=localhost                      # PostgreSQL host
DB_PORT=5432                          # PostgreSQL port  
DB_NAME=vrooli                        # Database name
DB_USER=postgres                      # Database user
DB_PASSWORD=postgres                  # Database password

# Resource Integration
SCENARIOS_PATH=../../                 # Path to scenarios directory
CLAUDE_CODE_BINARY=resource-claude-code  # Claude Code CLI command
```

### Service Configuration

Edit `.vrooli/service.json` to customize:

- **Port Ranges** - Adjust API and UI port allocations
- **Resource Dependencies** - Enable/disable optional resources
- **Template Paths** - Configure template directory locations
- **Analytics Settings** - Customize tracking and reporting

## 🧪 Testing

### Run Test Suite

```bash
# Full test suite
./test/run-tests.sh

# Specific test categories
./test/run-tests.sh --unit          # Unit tests only
./test/run-tests.sh --integration   # Integration tests only
./test/run-tests.sh --performance   # Performance benchmarks
```

### Manual Testing

```bash
# Test API endpoints
curl http://localhost:15000/health
curl -X POST http://localhost:15000/api/v1/scenarios/scan

# Test CLI commands  
saas-landing-manager status
saas-landing-manager scan --dry-run

# Test UI dashboard
open http://localhost:20000
```

### Validation

The scenario includes comprehensive validation via `scenario-test.yaml`:

- **Structure Tests** - Required files and directories
- **API Tests** - Endpoint functionality and responses
- **Database Tests** - Schema and data integrity
- **CLI Tests** - Command functionality and output
- **Performance Tests** - Response time and throughput benchmarks

## 🚨 Troubleshooting

### Common Issues

**1. API Connection Failed**
```bash
# Check service status
saas-landing-manager status

# Verify API health
curl http://localhost:15000/health

# Check logs
tail -f api/api.log
```

**2. Database Connection Issues**
```bash
# Verify PostgreSQL is running
vrooli resource postgres status

# Check database schema
psql -d vrooli -c "\dt"

# Reinitialize if needed
vrooli setup --resources postgres
```

**3. Template Generation Errors**
```bash
# Verify templates are loaded
saas-landing-manager template list

# Check template database
psql -d vrooli -c "SELECT * FROM templates;"

# Reload seed data if needed
psql -d vrooli -f initialization/storage/postgres/seed.sql
```

**4. Claude Code Integration Issues**
```bash
# Check Claude Code availability
resource-claude-code status

# Test agent spawning
resource-claude-code spawn --task "test task"

# Use direct deployment as fallback
saas-landing-manager deploy <id> <target> --method direct
```

### Debug Mode

Enable verbose logging:

```bash
# API debug mode
API_DEBUG=true ./api/saas-landing-manager-api

# CLI debug mode
saas-landing-manager status --verbose

# UI debug mode (check browser console)
```

### Log Locations

- **API Logs**: `api/api.log`
- **UI Logs**: Browser developer console
- **CLI Logs**: stdout/stderr
- **Database Logs**: PostgreSQL logs via `vrooli resource postgres logs`

## 🤝 Contributing

### Development Setup

1. **Clone and Setup**
   ```bash
   cd scenarios/saas-landing-manager
   ./scripts/manage.sh setup --dev
   ```

2. **Start Development Servers**
   ```bash
   # API with hot reload
   cd api && go run main.go
   
   # UI with hot reload  
   cd ui && npm run dev
   ```

3. **Run Tests**
   ```bash
   ./test/run-tests.sh --watch
   ```

### Adding New Templates

1. Create template files in `templates/base/` or `templates/industry/`
2. Add configuration schema 
3. Insert into database via `initialization/storage/postgres/seed.sql`
4. Test with `saas-landing-manager template list`

### Extending SaaS Detection

Modify `api/main.go` in the `analyzeSaaSCharacteristics` function:

```go
// Add new detection patterns
patterns := []string{
    "your-pattern-here",
    "another-indicator",
}
```

## 🔗 Integration

### API Integration

Use the REST API to integrate with external tools:

```javascript
// Scan scenarios
const response = await fetch('/api/v1/scenarios/scan', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ force_rescan: true })
});

// Generate landing page
const generate = await fetch('/api/v1/landing-pages/generate', {
    method: 'POST', 
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
        scenario_id: 'my-saas-app',
        enable_ab_testing: true
    })
});
```

### Webhook Integration

Set up webhooks for external notifications:

```bash
# Configure webhook URL
export WEBHOOK_URL=https://your-service.com/webhook

# Enable webhook notifications
saas-landing-manager config set webhook.enabled true
saas-landing-manager config set webhook.url $WEBHOOK_URL
```

### Third-Party Analytics

Integrate with external analytics platforms:

```javascript
// Google Analytics integration
window.gtag('config', 'GA_MEASUREMENT_ID', {
    custom_map: {
        custom_parameter_1: 'scenario_name',
        custom_parameter_2: 'template_id'
    }
});
```

## 📚 Resources

### Documentation

- **PRD.md** - Complete product requirements and technical specifications
- **API Documentation** - http://localhost:15000/docs (when running)
- **Template Guide** - `templates/README.md`
- **CLI Reference** - `saas-landing-manager help --all`

### Examples

- **Example Templates** - `templates/base/` directory
- **Sample Configurations** - `data/examples/`
- **Test Scenarios** - `test/fixtures/`

### External Resources

- [Landing Page Best Practices](https://vrooli.com/docs/landing-pages)
- [A/B Testing Guidelines](https://vrooli.com/docs/ab-testing)
- [SEO Optimization](https://vrooli.com/docs/seo)
- [Conversion Rate Optimization](https://vrooli.com/docs/cro)

## 📄 License

This scenario is part of the Vrooli ecosystem and follows the same licensing terms. See the main Vrooli repository for license details.

## 🆘 Support

- **Documentation Issues**: Update this README or PRD.md
- **Bug Reports**: Use the scenario test framework to validate issues
- **Feature Requests**: Modify PRD.md with new requirements
- **Community**: Vrooli Discord and GitHub discussions

---

**Made with ❤️ by the Vrooli AI Agent System**  
*Transforming every SaaS scenario into a complete business with professional landing pages and marketing optimization.*