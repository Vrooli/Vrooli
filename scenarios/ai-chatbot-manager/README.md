# 🚀 AI Chatbot Manager

**Build and manage AI-powered chatbots for 24/7 sales and support integration on websites**

AI Chatbot Manager is a complete SaaS platform that enables businesses to create, deploy, and manage intelligent chatbots powered by local AI models. It provides everything you need to add conversational AI to your website with zero external dependencies and complete data privacy.

## ✨ Features

### 🤖 **Intelligent Chatbots**
- **Local AI Power**: Ollama integration for privacy-first conversations
- **Custom Personalities**: Define unique chatbot behaviors and knowledge
- **Smart Conversations**: Context-aware responses with conversation memory
- **Lead Qualification**: Automatic lead capture and scoring

### 📊 **Management Dashboard**
- **Visual Dashboard**: React-based management interface
- **Real-time Analytics**: Conversation metrics and performance insights
- **Multi-tenant Support**: Manage multiple chatbots from one interface
- **A/B Testing**: Test different personalities and configurations

### 🔧 **Easy Integration**
- **Embeddable Widgets**: One-line integration for any website
- **Custom Styling**: Match your brand with configurable themes
- **WebSocket Support**: Real-time chat with instant responses  
- **Mobile Responsive**: Works seamlessly across all devices

### 🎯 **Business Intelligence**
- **Conversion Tracking**: Monitor leads and sales performance
- **Intent Analysis**: Understand what customers are asking about
- **Engagement Metrics**: Track conversation quality and length
- **Performance Optimization**: Data-driven insights for improvement

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   React UI      │    │    Go API        │    │   PostgreSQL    │
│   Dashboard     │◄──►│   REST/WebSocket │◄──►│   Database      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │     Ollama       │
                       │   AI Models      │
                       └──────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   Widget SDK     │
                       │   (JavaScript)   │
                       └──────────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Vrooli platform running
- Ollama service available
- PostgreSQL database
- Node.js (for UI development)
- Go 1.21+ (for API)

### Installation

1. **Start the scenario:**
   ```bash
   vrooli scenario run ai-chatbot-manager
   ```

2. **Access the dashboard:**
   ```
   http://localhost:3000
   ```

3. **Create your first chatbot:**
   - Click "Create New Chatbot"
   - Define personality and knowledge base
   - Copy the generated embed code
   - Add to your website

### CLI Quick Start

Install the CLI globally:
```bash
cd cli && ./install.sh
```

Basic CLI usage:
```bash
# Check status
ai-chatbot-manager status

# Create a new chatbot
ai-chatbot-manager create "Sales Bot" --personality "You are a sales expert"

# List all chatbots
ai-chatbot-manager list

# Test a chatbot interactively
ai-chatbot-manager chat <chatbot-id>

# View analytics
ai-chatbot-manager analytics <chatbot-id>
```

## 🔧 Configuration

### Environment Variables

```bash
# API Configuration
API_PORT=8090                    # API server port
DATABASE_URL=postgres://...      # PostgreSQL connection string
OLLAMA_URL=http://localhost:11434 # Ollama API endpoint

# UI Configuration  
UI_PORT=3000                     # React dashboard port
REACT_APP_API_URL=http://...     # API URL for frontend
```

### Chatbot Configuration

Each chatbot supports extensive configuration:

```json
{
  "name": "Sales Assistant",
  "personality": "You are an expert sales assistant...",
  "knowledge_base": "Our product features include...",
  "model_config": {
    "model": "llama3.2",
    "temperature": 0.7,
    "max_tokens": 1000
  },
  "widget_config": {
    "theme": "professional",
    "position": "bottom-right", 
    "primaryColor": "#3b82f6",
    "greeting": "Hi! How can I help you today?"
  }
}
```

## 📋 API Documentation

### Core Endpoints

#### Chatbot Management
```http
GET    /api/v1/chatbots           # List all chatbots
POST   /api/v1/chatbots           # Create new chatbot
GET    /api/v1/chatbots/{id}      # Get specific chatbot
PUT    /api/v1/chatbots/{id}      # Update chatbot
DELETE /api/v1/chatbots/{id}      # Delete chatbot
```

#### Chat Operations
```http
POST   /api/v1/chat/{id}          # Send message to chatbot
GET    /ws/{id}                   # WebSocket connection for real-time chat
```

#### Analytics
```http
GET    /api/v1/analytics/{id}     # Get chatbot analytics
GET    /api/v1/analytics/{id}/conversations # Get conversation history
```

### Example: Create Chatbot
```bash
curl -X POST http://localhost:8090/api/v1/chatbots \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Support Bot",
    "personality": "You are a helpful support assistant",
    "knowledge_base": "We offer 24/7 customer support...",
    "model_config": {
      "model": "llama3.2",
      "temperature": 0.6
    }
  }'
```

### Example: Send Message
```bash
curl -X POST http://localhost:8090/api/v1/chat/chatbot-id \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, I need help with pricing",
    "session_id": "user-session-123"
  }'
```

## 🎨 Widget Integration

### Basic Integration

Add this script tag to your website:
```html
<script src="https://your-domain.com/widget.js" 
        data-chatbot-id="your-chatbot-id">
</script>
```

### Advanced Integration

```html
<script src="https://your-domain.com/widget.js" 
        data-chatbot-id="your-chatbot-id"
        data-theme="dark"
        data-position="bottom-left"
        data-primary-color="#ff6b35">
</script>

<script>
// Programmatic control
ChatbotWidget.open();           // Open chat window
ChatbotWidget.close();          // Close chat window
ChatbotWidget.sendMessage("Hi"); // Send a message
</script>
```

### WordPress Integration

1. Go to your WordPress admin dashboard
2. Navigate to Appearance > Theme Editor
3. Add the widget script before `</body>` in footer.php
4. Or use a plugin like "Insert Headers and Footers"

### React Integration

```jsx
import { useEffect } from 'react';

function App() {
  useEffect(() => {
    // Load widget script
    const script = document.createElement('script');
    script.src = 'https://your-domain.com/widget.js';
    script.setAttribute('data-chatbot-id', 'your-chatbot-id');
    document.body.appendChild(script);
    
    return () => {
      document.body.removeChild(script);
    };
  }, []);
  
  return <div>Your app content</div>;
}
```

## 📊 Analytics & Insights

### Dashboard Metrics
- **Total Conversations**: Overall engagement volume
- **Lead Conversion Rate**: Percentage of chats that capture leads
- **Average Session Length**: User engagement duration
- **Popular Intents**: Most common user questions

### Advanced Analytics
- **Conversation Flow Analysis**: Understanding user journeys
- **Response Quality Scoring**: AI confidence levels
- **Peak Usage Patterns**: When users are most active
- **A/B Test Results**: Performance comparison of different configurations

### Custom Events Tracking
```javascript
// Track custom events in your widget integration
ChatbotWidget.on('leadCaptured', function(data) {
  // Send to your analytics platform
  gtag('event', 'chatbot_lead', {
    'chatbot_id': data.chatbotId,
    'conversation_id': data.conversationId
  });
});
```

## 🧪 Testing

### Run All Tests
```bash
# API tests
./test/test-api-endpoints.sh

# CLI tests (requires BATS)
cd cli && bats ai-chatbot-manager.bats

# UI tests
cd ui && npm test

# Integration tests
vrooli scenario test ai-chatbot-manager
```

### Manual Testing

1. **Test Chatbot Creation:**
   ```bash
   ai-chatbot-manager create "Test Bot" --personality "You are a test assistant"
   ```

2. **Test Chat Functionality:**
   ```bash
   ai-chatbot-manager chat <chatbot-id>
   ```

3. **Test Widget Integration:**
   - Create an HTML file with the widget code
   - Open in browser and test conversation flow

## 🔒 Security & Privacy

### Data Protection
- **Local Processing**: All AI inference happens locally via Ollama
- **No External APIs**: Zero dependency on external AI services
- **Encrypted Storage**: Conversations stored securely in PostgreSQL
- **GDPR Compliant**: Full control over user data

### Security Features
- **Input Sanitization**: All user inputs are sanitized
- **Rate Limiting**: Built-in protection against spam/abuse
- **Content Security Policy**: XSS protection for widgets
- **Audit Logging**: Complete conversation history tracking

### Privacy Controls
```javascript
// Widget privacy options
<script src="/widget.js" 
        data-chatbot-id="bot-id"
        data-store-conversations="false"  // Disable conversation storage
        data-collect-ip="false"           // Disable IP collection
        data-anonymous-mode="true">       // Enable anonymous mode
</script>
```

## 🚀 Deployment

### Production Deployment

1. **Build the application:**
   ```bash
   # Build API
   cd api && go build -o ai-chatbot-manager-api .
   
   # Build UI
   cd ui && npm run build
   ```

2. **Set production environment:**
   ```bash
   export NODE_ENV=production
   export DATABASE_URL=your-production-db-url
   export OLLAMA_URL=your-ollama-instance
   ```

3. **Deploy with process manager:**
   ```bash
   # Using PM2
   pm2 start ai-chatbot-manager-api --name chatbot-api
   pm2 start "serve -s ui/build" --name chatbot-ui
   ```

### Docker Deployment

```dockerfile
# Dockerfile
FROM node:18 AS ui-build
WORKDIR /app/ui
COPY ui/package*.json ./
RUN npm install
COPY ui/ ./
RUN npm run build

FROM golang:1.21 AS api-build  
WORKDIR /app/api
COPY api/go.* ./
RUN go mod download
COPY api/ ./
RUN go build -o ai-chatbot-manager-api .

FROM debian:bullseye-slim
WORKDIR /app
COPY --from=api-build /app/api/ai-chatbot-manager-api ./
COPY --from=ui-build /app/ui/build ./ui/build
EXPOSE 8090 3000
CMD ["./ai-chatbot-manager-api"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-chatbot-manager
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ai-chatbot-manager
  template:
    metadata:
      labels:
        app: ai-chatbot-manager
    spec:
      containers:
      - name: api
        image: your-registry/ai-chatbot-manager:latest
        ports:
        - containerPort: 8090
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: chatbot-secrets
              key: database-url
```

## 🔧 Customization

### Custom AI Models

Add support for additional Ollama models:
```bash
# Pull additional models
ollama pull mistral
ollama pull codellama
ollama pull neural-chat

# Configure in chatbot settings
{
  "model_config": {
    "model": "neural-chat",
    "temperature": 0.8,
    "max_tokens": 1500
  }
}
```

### Theme Customization

Create custom widget themes:
```css
/* Custom theme CSS */
.chatbot-widget-theme-custom {
  --primary-color: #your-brand-color;
  --text-color: #your-text-color;
  --background-color: #your-bg-color;
  --border-radius: 12px;
}
```

### Plugin Architecture

Extend functionality with custom plugins:
```javascript
// Example plugin: Sentiment Analysis
ChatbotWidget.addPlugin({
  name: 'sentiment-analysis',
  onMessage: function(message, response) {
    const sentiment = analyzeSentiment(message);
    if (sentiment === 'negative') {
      // Escalate to human agent
      this.escalateToHuman();
    }
  }
});
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

1. **Clone and setup:**
   ```bash
   git clone <repository>
   cd ai-chatbot-manager
   ```

2. **Start development services:**
   ```bash
   # Start API
   cd api && go run main.go
   
   # Start UI (in another terminal)
   cd ui && npm start
   
   # Start Ollama (if not running)
   ollama serve
   ```

3. **Run tests:**
   ```bash
   ./test/test-api-endpoints.sh
   cd cli && bats ai-chatbot-manager.bats
   ```

## 📋 Roadmap

### Version 1.1
- [ ] Voice chat integration
- [ ] Multi-language support
- [ ] Advanced analytics dashboard
- [ ] CRM integrations (Salesforce, HubSpot)

### Version 1.2
- [ ] A/B testing framework
- [ ] Custom model fine-tuning
- [ ] White-label deployment options
- [ ] Advanced workflow automation

### Version 2.0
- [ ] Multi-channel support (SMS, Email, Social)
- [ ] Enterprise SSO integration
- [ ] Advanced ML-powered insights
- [ ] Marketplace for chatbot templates

## 🆘 Troubleshooting

### Common Issues

**Issue: API health check fails**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Check if PostgreSQL is accessible
psql $DATABASE_URL -c "SELECT 1;"

# Check API logs
tail -f api/api.log
```

**Issue: Widget not loading on website**
```javascript
// Check browser console for errors
console.log('Widget loaded:', window.ChatbotWidget);

// Verify script src URL is correct
// Check CORS settings if cross-origin
```

**Issue: Conversations not saving**
```bash
# Check database connection
ai-chatbot-manager status --verbose

# Verify database schema
psql $DATABASE_URL -c "\dt"
```

### Performance Optimization

1. **Database Indexing:**
   ```sql
   CREATE INDEX CONCURRENTLY idx_messages_conversation_timestamp 
   ON messages(conversation_id, timestamp);
   ```

2. **Ollama Model Optimization:**
   ```bash
   # Use quantized models for faster inference
   ollama pull llama3.2:8b-instruct-q4_K_M
   ```

3. **Widget Caching:**
   ```javascript
   // Enable widget caching
   <script src="/widget.js" 
           data-cache-enabled="true"
           data-cache-ttl="3600">
   </script>
   ```

## 📞 Support

- **Documentation**: [Full documentation](docs/)
- **Issues**: [GitHub Issues](issues/)
- **CLI Help**: `ai-chatbot-manager help`
- **API Status**: `GET /health`

## 📄 License

This project is part of the Vrooli ecosystem. See [LICENSE](LICENSE) for details.

---

**Built with ❤️ for the Vrooli platform**

Transform your website visitors into engaged customers with AI-powered conversations that never sleep. 🚀