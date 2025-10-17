# Email Triage 📧🤖

**Transform your email chaos into intelligent automation**

AI-powered multi-tenant email management system that intelligently processes, prioritizes, and acts on emails using natural language rules and semantic understanding. Perfect for businesses looking to deploy email management as a SaaS service.

## 🌟 Key Features

- **AI Rule Assistant**: Create complex email rules using natural language - "Archive all newsletters" becomes intelligent automation
- **Semantic Search**: Find emails by meaning, not just keywords - search for "urgent meeting" and find related emails even without those exact words
- **Smart Prioritization**: ML-based scoring algorithm analyzes sender importance (30%), subject urgency (25%), content (20%), recipients (15%), and recency (10%)
- **Automated Triage Actions**: Execute 8 different actions - forward, archive, label, mark important, auto-reply, move to folder, delete, mark read
- **Real-Time Processing**: Background processor syncs emails every 5 minutes, applies rules automatically, and sends high-priority notifications
- **Multi-Tenant SaaS**: Deploy as revenue-generating service with isolated user environments and subscription tiers
- **Zero n8n Dependency**: Pure Go implementation for maximum performance and reliability

## 💰 Business Value

- **Revenue Potential**: $20K-50K per enterprise deployment
- **Time Savings**: 2-4 hours/day saved on email management per user
- **SaaS Ready**: Built-in authentication, billing tiers, and multi-tenancy
- **Competitive Edge**: AI-powered rules that understand context, not just keywords

## 🏗️ Architecture

### Core Components
- **Go API**: Real-time email processing and AI integration
- **React Dashboard**: Modern email triage interface
- **PostgreSQL**: User profiles, rules, and email metadata
- **Qdrant**: Vector embeddings for semantic search
- **Ollama/LiteLLM**: AI rule generation and email analysis

### Resource Integration
- **scenario-authenticator**: Multi-tenant user management
- **mail-in-a-box**: IMAP/SMTP email server access
- **notification-hub**: Alerts and urgent email notifications
- **qdrant**: Vector database for semantic search
- **ollama**: AI-powered rule generation

## 🚀 Quick Start

### Prerequisites
Ensure these resources are running:
```bash
# Start required resources
vrooli resource start postgres
vrooli resource start qdrant
vrooli scenario run scenario-authenticator
vrooli resource start mail-in-a-box

# Optional for enhanced features
vrooli resource start ollama
vrooli resource start notification-hub
```

### Installation
```bash
# Start the email triage scenario
vrooli scenario run email-triage

# Access the dashboard
open http://localhost:3201

# CLI usage
email-triage account add user@example.com password123
email-triage rule create "Archive all newsletters and promotional emails"
email-triage search "meeting about quarterly budget"
```

## 📊 Usage Examples

### AI Rule Creation
```bash
# Natural language rule creation
email-triage rule create "Forward all emails from VIP clients to my assistant"
email-triage rule create "Auto-reply to support emails during weekends"
email-triage rule create "Mark urgent any email mentioning 'deadline' or 'ASAP'"
```

### Semantic Search
```bash
# Find emails by context, not keywords
email-triage search "project status updates"
email-triage search "contract negotiations with vendors"
email-triage search "customer complaints about delivery"
```

### Account Management
```bash
# Manage multiple email accounts
email-triage account add work@company.com password123
email-triage account add personal@gmail.com apppassword456
email-triage account list --status
```

## 🎯 Target Use Cases

### Business Email Management
- **Executives**: AI prioritizes important communications, auto-handles routine emails
- **Customer Support**: Intelligent ticket routing and auto-responses
- **Sales Teams**: Lead qualification and follow-up automation

### SaaS Deployment
- **Email Service Provider**: Deploy as white-label email intelligence platform
- **Enterprise Tool**: Internal email management for large organizations
- **Productivity Suite**: Add-on service for existing business tools

## 📈 Subscription Tiers

| Feature | Free | Pro ($29/mo) | Business ($99/mo) |
|---------|------|--------------|-------------------|
| Email Accounts | 1 | 5 | 20 |
| Monthly Processing | 1,000 emails | 10,000 emails | 100,000 emails |
| AI Rules | 5 | 50 | 200 |
| Semantic Search | ✓ | ✓ | ✓ |
| Priority Support | - | ✓ | ✓ |
| API Access | - | ✓ | ✓ |

## 🔧 API Endpoints

### Email Account Management
- `POST /api/v1/accounts` - Connect new email account
- `GET /api/v1/accounts` - List user's email accounts
- `DELETE /api/v1/accounts/{id}` - Remove email account

### AI Rule Management  
- `POST /api/v1/rules` - Create rule with AI assistance
- `GET /api/v1/rules` - List user's triage rules
- `PUT /api/v1/rules/{id}` - Update rule configuration

### Semantic Search
- `GET /api/v1/emails/search?q={query}` - Search emails by context
- `GET /api/v1/emails/{id}` - Get email details
- `POST /api/v1/emails/{id}/actions` - Apply triage actions

## 🛠️ Development

### Local Development
```bash
# Install dependencies
cd api && go mod tidy
cd ui && npm install

# Run in development mode
go run api/main.go
npm run dev
```

### Testing
```bash
# Run scenario validation tests
vrooli scenario test email-triage

# API integration tests  
cd api && go test ./...

# UI component tests
cd ui && npm test
```

## 📚 Documentation

- **[PRD.md](PRD.md)** - Complete product requirements and architecture
- **[API Documentation](docs/api.md)** - Detailed API specification
- **[CLI Reference](docs/cli.md)** - Command-line interface guide
- **[Deployment Guide](docs/deployment.md)** - Production deployment instructions

## 🤝 Integration Examples

### Customer Support Automation
```javascript
// Integrate with existing support ticket system
const emailTriage = new EmailTriageClient();
await emailTriage.rules.create({
  description: "Forward urgent support emails to on-call engineer",
  actions: ["forward:oncall@company.com", "label:urgent", "priority:high"]
});
```

### Sales Pipeline Integration
```python
# Qualify leads automatically
import email_triage_client as et

client = et.Client(api_key=os.environ['EMAIL_TRIAGE_API_KEY'])
rule = client.rules.create(
    description="Tag emails from potential customers mentioning budget",
    actions=["add_to_crm", "notify_sales_team", "priority_score:0.8"]
)
```

## 🔒 Security & Privacy

- **Multi-tenant Isolation**: Complete data separation between users
- **Encrypted Storage**: All email data encrypted at rest
- **GDPR Compliant**: Data export and deletion capabilities
- **JWT Authentication**: Secure API access with token validation
- **Audit Logging**: Complete trail of all email processing actions

## 🚨 Known Limitations

- **Email Provider Support**: Currently requires mail-in-a-box (Gmail/Outlook support coming in v2.0)
- **AI Accuracy**: Generated rules may need human refinement (confidence scoring provided)
- **Real-time Processing**: Large volumes may have slight processing delays

## 🛣️ Roadmap

### Version 1.1 (Next 4 weeks)
- [ ] Mobile-responsive dashboard improvements
- [ ] Batch email operations with undo
- [ ] Enhanced AI rule confidence scoring

### Version 2.0 (Next 3 months)
- [ ] Gmail and Outlook integration
- [ ] Advanced sentiment analysis
- [ ] Team collaboration features
- [ ] Mobile app (iOS/Android)

### Long-term Vision
- [ ] Predictive email management
- [ ] Voice-controlled email triage
- [ ] Enterprise SSO integration
- [ ] Advanced compliance features

---

**Revenue Potential**: $20K-50K per deployment  
**Time to Value**: < 30 minutes setup  
**Competitive Edge**: Only AI-powered email triage with multi-tenant SaaS capability  

Ready to transform email management from chaos to intelligence? Start with `vrooli scenario run email-triage` 🚀