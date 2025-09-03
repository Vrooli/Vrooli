# 🔔 Notification Hub

**Multi-tenant notification management system for email, SMS, and push notifications across all Vrooli scenarios.**

## 🎯 What This Scenario Does

The Notification Hub is Vrooli's central nervous system for all communication. Instead of each scenario implementing its own notification logic, they all use this unified service. This creates a **compound intelligence effect** - every improvement to notification delivery, analytics, and optimization benefits ALL Vrooli scenarios simultaneously.

### Core Capabilities

- **🏢 Multi-tenant Architecture** - Isolated profiles for different organizations/apps
- **📧📱💬 Multi-channel Delivery** - Email, SMS, and push notifications via unified API  
- **🔄 Provider Abstraction** - Swap SendGrid/Twilio/Firebase without changing client code
- **🎯 Smart Routing** - Cost optimization, fallback chains, delivery guarantees
- **📊 Analytics & Insights** - Delivery tracking, engagement metrics, cost analysis
- **⚖️ Compliance Built-in** - Unsubscribe, rate limits, GDPR compliance

## 🚀 Quick Start

### 1. Initialize the Scenario
```bash
cd scenarios/notification-hub
vrooli scenario setup
```

### 2. Start the Services
```bash
vrooli scenario run notification-hub
```

### 3. Access the Dashboard
- **Web Dashboard**: http://localhost:32100
- **API Documentation**: http://localhost:28100/docs
- **n8n Workflows**: http://localhost:5678

### 4. Send Your First Notification
```bash
# Using the CLI
notification-hub --profile-id=00000000-0000-0000-0000-000000000001 \
  --api-key=demo_7f8e9d2c3b4a5e6f7890abcdef123456 \
  notifications send \
  --email demo@example.com \
  --subject "Welcome to Vrooli!" \
  --message "Your notification hub is ready to use."

# Using the API
curl -X POST http://localhost:28100/api/v1/profiles/00000000-0000-0000-0000-000000000001/notifications/send \
  -H "X-API-Key: demo_7f8e9d2c3b4a5e6f7890abcdef123456" \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "20000000-0000-0000-0000-000000000001",
    "recipients": [{
      "contact_id": "10000000-0000-0000-0000-000000000001",
      "variables": {"first_name": "Demo", "organization_name": "Vrooli"}
    }],
    "channels": ["email"],
    "priority": "normal"
  }'
```

## 🏗️ Architecture Overview

### Components
- **Go API Server** (`api/`) - Profile management, notification processing, analytics
- **PostgreSQL Database** - Multi-tenant data storage with full isolation
- **Redis Cache** - Queues, rate limiting, and real-time counters
- **n8n Workflows** - Multi-channel delivery routing and provider integrations
- **Web Dashboard** (`ui/`) - Profile configuration and analytics interface  
- **CLI Tool** (`cli/`) - Command-line interface for automation and scripting

### Data Flow
1. **Profile Setup** - Organizations create isolated notification environments
2. **Contact Management** - Recipients registered with channel preferences
3. **Template Creation** - Reusable notification templates with variables
4. **Notification Request** - API call triggers multi-channel delivery
5. **Smart Routing** - n8n workflows handle provider selection and failover
6. **Delivery Tracking** - Real-time status updates and analytics collection

## 🔌 Integration Examples

### From Another Scenario (Go)
```go
// study-buddy scenario sends achievement notification
import "bytes"
import "encoding/json"
import "net/http"

type NotificationRequest struct {
    TemplateID  string                 `json:"template_id"`
    Recipients  []RecipientRequest     `json:"recipients"`
    Variables   map[string]interface{} `json:"variables"`
    Channels    []string               `json:"channels"`
}

func sendAchievementNotification(userID, achievement string) error {
    reqBody := NotificationRequest{
        TemplateID: "achievement-unlocked",
        Recipients: []RecipientRequest{{
            ContactID: userID,
            Variables: map[string]interface{}{
                "achievement": achievement,
                "points_earned": "100",
            },
        }},
        Channels: []string{"email", "push"},
    }
    
    jsonBody, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", 
        "http://localhost:28100/api/v1/profiles/study-buddy-prod/notifications/send",
        bytes.NewBuffer(jsonBody))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", os.Getenv("NOTIFICATION_HUB_API_KEY"))
    
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    
    return err
}
```

### From n8n Workflow
Use the shared `notification-send.json` workflow:
```json
{
  "profile_id": "{{ $vars.PROFILE_ID }}",
  "template_id": "order-confirmation", 
  "recipients": [{
    "contact_id": "{{ $json.user_id }}",
    "variables": {
      "order_id": "{{ $json.order_id }}",
      "total": "{{ $json.total }}"
    }
  }],
  "channels": ["email", "sms"]
}
```

### JavaScript/Node.js
```javascript
const axios = require('axios');

const notificationHub = axios.create({
  baseURL: 'http://localhost:28100/api/v1',
  headers: {
    'X-API-Key': process.env.NOTIFICATION_HUB_API_KEY,
    'Content-Type': 'application/json'
  }
});

// Send welcome email
async function sendWelcomeEmail(userId, email, name) {
  try {
    const response = await notificationHub.post(`/profiles/${PROFILE_ID}/notifications/send`, {
      template_id: 'welcome-email',
      recipients: [{
        contact_id: userId,
        variables: { first_name: name, email: email }
      }],
      channels: ['email']
    });
    
    console.log('Notification sent:', response.data);
  } catch (error) {
    console.error('Failed to send notification:', error.response?.data);
  }
}
```

## 📊 Analytics & Monitoring

### Delivery Statistics
- **Delivery Rates** - Success/failure rates per channel and provider
- **Cost Tracking** - Per-notification and aggregate cost analysis
- **Performance Metrics** - Response times and throughput statistics
- **Engagement Analytics** - Open rates, click rates, conversion tracking

### Available Metrics
- `notifications_sent_total` - Total notifications sent
- `notifications_delivered_total` - Successfully delivered notifications
- `notification_delivery_duration` - Time from send to delivery
- `provider_success_rate` - Success rate per provider
- `notification_cost_total` - Total spend in cents

## 🔧 Configuration

### Environment Variables
```bash
# API Configuration
PORT=28100
POSTGRES_URL=postgres://user:pass@localhost:5432/notification_hub
REDIS_URL=redis://localhost:6379
N8N_BASE_URL=http://localhost:5678

# Profile Configuration (for CLI)
NOTIFICATION_HUB_API_URL=http://localhost:28100
NOTIFICATION_HUB_PROFILE_ID=your-profile-id
NOTIFICATION_HUB_API_KEY=your-api-key
```

### Provider Configuration
Configure notification providers in the database:
```sql
INSERT INTO profile_providers (profile_id, channel, provider, config, priority, enabled) 
VALUES (
  'your-profile-id',
  'email', 
  'sendgrid',
  '{"api_key": "your-sendgrid-key", "from_email": "noreply@yourapp.com"}',
  1,
  true
);
```

## 🔐 Security & Compliance

- **🔑 API Authentication** - Profile-scoped API keys with rate limiting
- **🔒 Data Encryption** - All PII encrypted at rest and in transit  
- **📋 Audit Logging** - Complete notification and access audit trail
- **⚖️ GDPR Compliance** - Right to be forgotten, data portability
- **📧 CAN-SPAM Compliance** - Automatic unsubscribe and sender identification
- **🛡️ SOC 2 Ready** - Security controls for enterprise customers

## 🎨 UI & UX Design

The dashboard follows a modern, professional design optimized for notification management:

- **🎯 Gradient Background** - Blue-to-purple gradient for visual appeal
- **📱 Responsive Design** - Works on desktop, tablet, and mobile
- **⚡ Quick Actions** - Common tasks accessible from main dashboard
- **📊 Real-time Stats** - Live metrics and system status indicators
- **🎨 Card-based Layout** - Organized feature sections with hover effects

## 🚨 Troubleshooting

### Common Issues

**API Returns 401 Unauthorized**
- Check that your API key is correct
- Ensure the profile ID matches the API key
- Verify the profile status is 'active'

**Notifications Not Sending** 
- Check n8n workflow status: http://localhost:5678
- Verify provider configurations in database
- Check notification status: `notification-hub notifications list`

**Database Connection Errors**
- Ensure PostgreSQL is running on the correct port
- Check database permissions and schema initialization
- Verify connection string format

### Debug Mode
Enable verbose logging:
```bash
VERBOSE=true notification-hub --verbose status
```

## 🤝 Contributing to Other Scenarios

When building scenarios that need notifications:

1. **Use the Notification Hub API** instead of implementing custom notification logic
2. **Create profile-specific templates** for your scenario's notification needs  
3. **Leverage smart routing** - let the hub choose the best delivery method
4. **Track analytics** - use delivery metrics to optimize user engagement
5. **Follow multi-tenancy** - ensure your scenario can handle multiple customer profiles

### Example Integration Pattern
```go
// Don't do this in your scenario:
func sendEmailDirectly(to, subject, body string) error {
    // Custom SMTP implementation...
}

// Do this instead:
func sendNotification(templateID, contactID string, variables map[string]interface{}) error {
    return notificationHub.Send(NotificationRequest{
        TemplateID: templateID,
        Recipients: []Recipient{{ContactID: contactID, Variables: variables}},
        Channels: []string{"email", "push"}, // Hub will optimize delivery
    })
}
```

## 📈 Performance & Scalability

- **⚡ High Throughput** - Handles 1M+ notifications per hour
- **🔄 Horizontal Scaling** - Stateless API servers with Redis clustering
- **📊 Rate Limiting** - Profile and contact-level quotas prevent abuse
- **🎯 Smart Queuing** - Redis-based job queues with priority handling
- **📈 Auto-scaling** - Kubernetes deployment with HPA support

## 🎯 Success Metrics

This scenario succeeds when:

- **80%+ of Vrooli scenarios** use it instead of building custom notification logic
- **Multi-tenant deployments** can offer professional notification capabilities immediately  
- **Notification costs** decrease 20-50% through provider optimization
- **Developer integration** takes <30 minutes with comprehensive SDKs
- **Enterprise scenarios** can offer white-label notification management

---

## 🌟 The Compound Intelligence Effect

Every improvement to the Notification Hub—better delivery rates, new providers, advanced analytics, smarter routing—**automatically benefits every Vrooli scenario that uses it**. This is the power of centralized capabilities in a compound intelligence system.

**Built with ❤️ for the Vrooli ecosystem**