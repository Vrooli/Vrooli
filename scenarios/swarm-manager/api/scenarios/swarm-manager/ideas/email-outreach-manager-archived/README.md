# Email Outreach Manager

> **AI-powered personalized email campaigns with contact intelligence**

Email Outreach Manager transforms manual email marketing into intelligent, relationship-aware campaigns. By combining AI template generation with contact-book integration, it creates personalized outreach that scales while maintaining authentic relationships.

## 🎯 What It Does

### Core Capabilities
- **AI Template Generation**: Create beautiful email templates from campaign briefs using Ollama
- **Contact Intelligence**: Leverage contact-book data for deep personalization (names, pronouns, preferences)
- **Personalization Tiers**: Three-level system (full, partial, template-only) with visual indicators
- **Drip Campaign Automation**: Time-based email sequences with intelligent follow-up
- **Campaign Analytics**: Track opens, clicks, and engagement with personalization insights
- **Professional Email Delivery**: Reliable sending via mail-in-a-box with proper SMTP handling

### Intelligence Amplification
Every campaign becomes relationship-intelligent:
- **Sales Teams** → Personalized lead nurturing based on contact relationship strength
- **Event Organizers** → Invitations tailored to attendee preferences and social connections  
- **Customer Success** → Proactive check-ins with relationship-aware messaging
- **Fundraising** → Investor outreach optimized by connection strength and interests

## 🚀 Quick Start

### Prerequisites
- PostgreSQL (campaign and analytics storage)
- Ollama (AI template generation)
- Mail-in-a-Box (email delivery)
- Contact-Book scenario (optional, for recipient intelligence)

### Installation

```bash
# Run the scenario
vrooli scenario run email-outreach-manager

# Or set up manually
cd scenarios/email-outreach-manager
vrooli setup
```

### Basic Usage

```bash
# Create a new campaign
email-outreach-manager create-campaign "Product Launch" "Announce our new AI feature" \
  --recipients-file contacts.csv \
  --tone professional \
  --template-docs product-specs.md

# Preview generated template
email-outreach-manager preview-template "Product announcement" \
  --docs product-specs.md \
  --tone professional

# Send campaign
email-outreach-manager send-campaign campaign-id-here \
  --schedule "2025-09-10T09:00:00Z"

# Check analytics
email-outreach-manager analytics campaign-id-here --detailed
```

### Campaign Creation Workflow

1. **Define Campaign**: Set purpose, tone, and target audience
2. **Template Generation**: AI creates personalized template from your documents/briefs
3. **Recipient Enrichment**: Contact-book integration adds personalization data
4. **Preview & Review**: See exactly how emails will appear to recipients
5. **Schedule & Send**: Execute campaign with analytics tracking

## 📊 Personalization Intelligence

### Three-Tier Personalization System

**🟢 Full Personalization** - Complete contact-book data available
- Recipient name, pronouns, preferences
- Relationship context and history
- Interest-based content customization
- Optimal send timing based on engagement patterns

**🟡 Partial Personalization** - Limited data available
- Basic information (name, email)
- Manual data supplementation
- Template personalization with available fields

**🔴 Template Only** - No personalization data
- Generic template with clear warning indicators
- Opportunity to enrich contact data for future campaigns

### Contact-Book Integration

```bash
# Campaign automatically enriches recipients from contact-book
email-outreach-manager create-campaign "Newsletter" "Monthly update" \
  --recipients-file subscribers.csv
  # -> Automatically looks up each recipient in contact-book
  # -> Adds names, pronouns, preferences, relationship context

# Fallback to manual data when contact-book unavailable
email-outreach-manager create-campaign "Event Invite" "Conference invitation" \
  --recipients-file attendees.csv \
  --fallback-data '{"default_pronouns": "they/them"}'
```

## 🏗️ Architecture

### Technology Stack
- **API**: Go with Gin framework
- **Database**: PostgreSQL for campaigns, templates, and analytics
- **AI**: Ollama integration for template generation
- **Email**: Mail-in-a-Box for reliable delivery
- **Queue**: Redis for campaign processing (optional)
- **UI**: Modern web dashboard for campaign management

### API Endpoints

```
POST   /api/v1/campaigns              # Create new campaign
GET    /api/v1/campaigns              # List campaigns
GET    /api/v1/campaigns/:id          # Get campaign details
POST   /api/v1/campaigns/:id/send     # Execute campaign
GET    /api/v1/campaigns/:id/analytics # Campaign performance

POST   /api/v1/templates/generate     # Generate AI template
GET    /api/v1/templates              # List templates
```

### Data Flow
1. **Campaign Creation** → Template generation via Ollama
2. **Recipient Processing** → Contact-book enrichment (if available)
3. **Personalization** → Multi-tier customization based on data quality
4. **Queue Management** → Redis-backed sending queue (if available)
5. **Email Delivery** → Mail-in-a-Box SMTP with tracking
6. **Analytics Collection** → Real-time engagement tracking

## 🎨 UI Experience

### Professional Email Marketing Interface
- **Campaign Builder**: Drag-and-drop template designer
- **Recipient Manager**: Import, enrich, and segment contact lists
- **Analytics Dashboard**: Real-time campaign performance with personalization insights
- **Template Library**: Reusable AI-generated templates with categorization
- **Send Management**: Schedule, queue, and monitor campaign delivery

### Style Profile
- Clean, professional email marketing interface inspired by ConvertKit/Mailchimp
- Focus on clarity and efficiency for campaign management
- Clear visual indicators for personalization quality and warnings
- Mobile-responsive dashboard for campaign monitoring

## 🔌 Integration Examples

### Sales Pipeline Integration
```bash
# Nurture leads based on relationship strength
email-outreach-manager create-campaign "Lead Follow-up" "Continue conversation" \
  --recipients-from-contact-book "tags:prospect" \
  --personalization-strategy relationship-aware
```

### Event Marketing Integration
```bash
# Conference invitations with attendee preferences
email-outreach-manager create-campaign "Tech Conference 2025" "Join us for cutting-edge AI talks" \
  --recipients-file conference-prospects.csv \
  --template-docs conference-agenda.pdf \
  --drip-sequence "invite,reminder-2weeks,reminder-1week,last-chance"
```

### Customer Success Integration
```bash
# Relationship maintenance outreach
email-outreach-manager create-campaign "Quarterly Check-in" "How can we better serve you?" \
  --recipients-from-contact-book "maintenance_priority:>0.7" \
  --tone friendly \
  --personalization-depth full
```

## 🧪 Testing & Validation

### Test Campaign Creation
```bash
# Test template generation
email-outreach-manager preview-template "Product launch" \
  --tone professional \
  --recipient-sample '{"name":"Alex Chen","pronouns":"they/them"}'

# Test contact integration
email-outreach-manager test-contact-integration --sample-email test@example.com

# Full campaign test
email-outreach-manager create-campaign "Test Campaign" "Test message" \
  --recipients-file test-contacts.csv \
  --test-mode
```

### Performance Validation
- Template generation completes within 30 seconds
- Contact enrichment averages under 2 seconds per recipient
- Campaign setup takes less than 5 minutes end-to-end
- Send rate maintains 100 emails/minute
- Analytics update in real-time

## 🚀 Future Enhancements

### Version 2.0 Planned Features
- **Dynamic Content**: Real-time personalization based on recipient behavior
- **Send Time Optimization**: AI-powered optimal delivery timing
- **Multi-channel Campaigns**: Email + social media coordination
- **Advanced Analytics**: Predictive engagement scoring

### Long-term Vision
- **Self-Optimizing Campaigns**: AI learns from engagement patterns
- **Relationship Evolution Tracking**: Campaign impact on relationship strength
- **Cross-Scenario Intelligence**: Integration with all communication scenarios

## 🤝 Contributing

### Development Setup
```bash
cd scenarios/email-outreach-manager
go mod download  # Install Go dependencies
./cli/install.sh  # Install CLI globally
npm install       # Install UI dependencies (in ui/ directory)
vrooli setup      # Initialize resources
```

### Key Files
- **PRD.md**: Comprehensive product requirements
- **api/main.go**: Go API server
- **cli/email-outreach-manager**: CLI implementation
- **ui/**: Modern web dashboard
- **initialization/storage/postgres/schema.sql**: Database schema

## 📚 Resources

- **[PRD.md](./PRD.md)**: Complete product requirements and technical specifications
- **CLI Reference**: `email-outreach-manager --help`
- **API Documentation**: Available at `/api/docs` when running
- **Contact-Book Integration**: See contact-book scenario documentation

---

**Email Outreach Manager transforms every communication into intelligent, relationship-aware outreach that strengthens connections while achieving business goals.**

🚀 **Ready to revolutionize your email marketing? Get started with `vrooli scenario run email-outreach-manager`**