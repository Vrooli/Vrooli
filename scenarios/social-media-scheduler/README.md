# Social Media Scheduler

> **Intelligent multi-platform content scheduling with AI-powered optimization**

An AI-driven social media scheduler that automatically optimizes content for each platform's unique constraints, audience, and culture while maintaining brand consistency and optimal posting times across Twitter, Instagram, LinkedIn, Facebook, and TikTok.

## 🎯 What This Scenario Does

The Social Media Scheduler transforms Vrooli into a comprehensive social media management platform by adding:

- **Multi-Platform Scheduling**: Schedule posts across Twitter, Instagram, LinkedIn, and Facebook simultaneously
- **AI Content Optimization**: Automatically adapt content for each platform's character limits, culture, and best practices
- **Visual Calendar Interface**: Intuitive drag-and-drop calendar for managing your content schedule
- **Real-Time Status Updates**: WebSocket-powered live updates on posting status and engagement metrics
- **Brand Consistency**: Integration with brand-manager for consistent visual identity across platforms
- **Campaign Management**: Organize posts into campaigns with inherited brand guidelines
- **Performance Analytics**: Track engagement, reach, and performance across all platforms

## 🧠 Intelligence Amplification

This scenario multiplies the value of every content-generating scenario in Vrooli:

- **Content Scenarios**: Any scenario that generates content (blog posts, announcements, campaigns) can now automatically distribute across all social platforms
- **Brand Scenarios**: Integrates with brand-manager to ensure consistent brand application
- **Campaign Scenarios**: Works with campaign-content-studio for comprehensive campaign management
- **Analytics Intelligence**: Platform-specific insights become reusable knowledge for optimizing future content

## 🚀 Quick Start

### Prerequisites

- Vrooli environment with PostgreSQL, Redis, MinIO, and Ollama resources enabled
- Go 1.21+ for API development
- Node.js 18+ for UI development

### Installation

```bash
# 1. Setup the scenario
vrooli scenario run social-media-scheduler

# 2. The setup automatically:
#    - Installs CLI globally
#    - Builds Go API server
#    - Installs UI dependencies
#    - Initializes database schema
#    - Seeds demo data

# 3. Access the interface
open http://localhost:38000  # Web UI
social-media-scheduler --help  # CLI
```

### Demo Account

The scenario comes with a pre-configured demo account:

- **Email**: `demo@vrooli.com`
- **Password**: `demo123`

## 💻 Usage

### Web Interface

The primary interface is a modern React application featuring:

1. **Dashboard**: Overview of scheduled posts, analytics, and quick actions
2. **Calendar View**: Visual calendar with drag-and-drop scheduling
3. **Post Composer**: Rich editor with platform-specific previews
4. **Analytics**: Performance tracking across all platforms
5. **Account Management**: Connect and manage social media accounts

### CLI Interface

Complete CLI for automation and scripting:

```bash
# Authentication
social-media-scheduler login user@example.com password
social-media-scheduler whoami

# Scheduling posts
social-media-scheduler schedule "Product Launch" \
  "We're excited to announce our new AI scheduler! 🚀 #ProductLaunch" \
  "twitter,linkedin" \
  "2024-07-15T14:00:00Z"

# Managing posts
social-media-scheduler list
social-media-scheduler status POST_ID

# Platform management
social-media-scheduler platforms
social-media-scheduler accounts
```

### API Integration

RESTful API for programmatic access:

```bash
# Schedule a post via API
curl -X POST http://localhost:18000/api/v1/posts/schedule \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "API Test Post",
    "content": "Posted via API! #automation",
    "platforms": ["twitter", "linkedin"],
    "scheduled_at": "2024-07-15T14:00:00Z",
    "auto_optimize": true
  }'
```

## 🏗️ Architecture

### Components

- **Go API Server** (`api/`): Core scheduling engine with platform adapters
- **React UI** (`ui/`): Modern web interface with real-time updates
- **CLI Tool** (`cli/`): Command-line interface for automation
- **Job Processor**: Redis-based background job queue for reliable posting
- **Platform Adapters**: Specialized handlers for each social media platform

### Data Flow

```mermaid
graph TB
    A[User Creates Post] --> B[Content Optimization]
    B --> C[Platform Adaptation]
    C --> D[Queue Scheduling]
    D --> E[Background Processing]
    E --> F[Platform Posting]
    F --> G[Analytics Collection]
    G --> H[Real-time Updates]
```

### Resource Dependencies

- **PostgreSQL**: User accounts, posts, analytics, OAuth tokens
- **Redis**: Job queue, rate limiting, real-time updates
- **MinIO**: Media storage with platform-specific optimization
- **Ollama**: AI-powered content optimization
- **Browserless**: Fallback posting and screenshot verification

## 🔧 Configuration

### Environment Variables

```bash
# Ports
API_PORT=18000          # Go API server port
UI_PORT=38000           # React UI server port

# Database
DATABASE_URL=postgres://postgres:password@localhost:5432/vrooli_social_media_scheduler

# Redis
REDIS_URL=redis://localhost:6379/0

# Storage
MINIO_URL=http://localhost:9000

# AI Service
OLLAMA_URL=http://localhost:11434

# Security
JWT_SECRET=your-secret-key-change-in-production
```

### Platform Configuration

Each platform has specific requirements and optimization rules:

- **Twitter**: 280 characters, 4 media max, hashtag optimization
- **Instagram**: 2200 characters, requires media, hashtag strategy (5-10)
- **LinkedIn**: 3000 characters, professional tone, industry hashtags (1-3)
- **Facebook**: High character limit, community focus, engagement optimization

## 📊 Features in Detail

### AI Content Optimization

The system uses Ollama to intelligently adapt content for each platform:

- **Character Limits**: Automatically truncates or expands content appropriately
- **Tone Adaptation**: Professional for LinkedIn, conversational for Twitter, visual for Instagram
- **Hashtag Strategy**: Platform-specific hashtag recommendations and limits
- **Engagement Optimization**: Best practices for each platform's algorithm

### Real-Time Updates

WebSocket integration provides live updates for:

- Post status changes (scheduled → posting → posted/failed)
- Engagement metrics as they arrive
- Queue status and processing updates
- Account connection/disconnection events

### Campaign Integration

Seamless integration with campaign-content-studio:

- Import entire campaigns with multiple posts
- Inherit brand guidelines and content templates
- Bulk scheduling across platforms
- Campaign performance analytics

### Brand Consistency

Integration with brand-manager ensures:

- Consistent visual identity across platforms
- Brand color and font application to generated graphics
- Voice and messaging consistency
- Brand compliance checking

## 🔐 Security & Authentication

### Multi-Tenant Architecture

- JWT-based authentication with secure token management
- Per-user data isolation and access controls
- OAuth integration for social platform connections
- Encrypted storage of social media credentials

### Privacy & Compliance

- GDPR-compliant data export and deletion
- Audit trail for all user actions
- Secure token storage with automatic refresh
- Platform API terms compliance

## 📈 Analytics & Performance

### Metrics Tracked

- **Engagement**: Likes, shares, comments, reactions per platform
- **Reach**: Impressions and audience reach across platforms
- **Timing**: Optimal posting time analysis
- **Content**: Performance by content type and platform
- **Trends**: Historical performance and growth metrics

### Performance Optimization

- **Caching**: Redis-based caching for fast response times
- **Queue Processing**: Efficient background job processing
- **Database Optimization**: Indexed queries and connection pooling
- **CDN Integration**: MinIO for fast media delivery

## 🧪 Testing

### Integration Tests

```bash
# Run the full phased suite
./test/run-tests.sh

# Exercise a specific phase
./test/phases/test-integration.sh

# Or use the Vrooli CLI orchestrator
vrooli scenario test social-media-scheduler
```

### Test Coverage

- API endpoint functionality
- Authentication and authorization
- Post scheduling and processing
- Real-time updates and WebSocket connectivity
- Database operations and data integrity
- Resource connectivity and health checks

## 🔄 Integration with Other Scenarios

### Campaign Content Studio

```bash
# Generate campaign → Schedule across platforms
curl -X POST /api/campaigns/generate-and-schedule \
  -d '{"campaign_id": "summer-launch", "platforms": ["twitter", "linkedin", "instagram"]}'
```

### Brand Manager

- Automatic brand asset application
- Consistent visual identity across posts
- Brand guideline enforcement
- Asset library integration

### Scenario Authenticator

- Multi-tenant SaaS capability
- User management and billing integration
- OAuth flow for social platform connections
- Team collaboration features

## 📚 API Reference

### Authentication

- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/register` - User registration
- `GET /api/v1/auth/me` - Current user info
- `GET /api/v1/auth/platforms` - Platform configurations

### Post Management

- `POST /api/v1/posts/schedule` - Schedule new post
- `GET /api/v1/posts/calendar` - Get calendar posts
- `GET /api/v1/posts/{id}` - Get specific post
- `PUT /api/v1/posts/{id}` - Update post
- `DELETE /api/v1/posts/{id}` - Delete post

### Analytics

- `GET /api/v1/analytics/overview` - Analytics overview
- `GET /api/v1/analytics/platforms` - Platform-specific analytics
- `GET /api/v1/analytics/optimal-times` - Optimal posting times

### Media Management

- `POST /api/v1/media/upload` - Upload media files
- `GET /api/v1/media` - List media files
- `POST /api/v1/media/{id}/optimize` - Optimize for platforms

## 🎨 UI Customization

### Theming

The React UI supports customization through:

- CSS custom properties for colors and spacing
- Modular component architecture
- Platform-specific branding colors
- Responsive design with mobile support

### Component Structure

```
ui/public/
├── index.html          # Main HTML template
├── app.js              # React application
└── assets/             # Static assets
```

## 🔍 Troubleshooting

### Common Issues

1. **API Server Not Starting**
   ```bash
   # Check if ports are available
   lsof -i :18000
   
   # Verify database connection
   psql -h localhost -p 5432 -U postgres -d vrooli_social_media_scheduler -c "SELECT 1;"
   ```

2. **Job Processing Issues**
   ```bash
   # Check Redis connectivity
   redis-cli ping
   
   # Monitor job queue
   redis-cli monitor
   ```

3. **OAuth Connection Failures**
   ```bash
   # Check platform API credentials
   # Verify callback URLs are correctly configured
   # Ensure SSL certificates are valid (for production)
   ```

### Debug Mode

Enable detailed logging:

```bash
export LOG_LEVEL=debug
export ENVIRONMENT=development
```

## 🛠️ Development

### Setting Up Development Environment

```bash
# 1. Clone and setup
git clone https://github.com/your-org/vrooli
cd vrooli/scenarios/social-media-scheduler

# 2. Install dependencies
cd api && go mod download
cd ../ui && npm install
cd ../cli && chmod +x social-media-scheduler

# 3. Setup database
psql -U postgres -c "CREATE DATABASE vrooli_social_media_scheduler;"
psql -U postgres -d vrooli_social_media_scheduler -f initialization/storage/schema.sql
psql -U postgres -d vrooli_social_media_scheduler -f initialization/storage/seed.sql

# 4. Start development servers
cd api && go run . --mode=both &
cd ui && node server.js &
```

### Adding New Platform

1. Implement `PlatformAdapter` interface in `api/platforms.go`
2. Add platform configuration to UI
3. Update database schema for platform-specific fields
4. Add platform-specific tests

### Contributing

1. Fork the repository
2. Create feature branch
3. Make changes with tests
4. Submit pull request

## 📝 License

This scenario is part of the Vrooli project. See the main project license for details.

## 🤝 Support

- **Documentation**: See `PRD.md` for detailed requirements
- **Issues**: Report bugs and feature requests through Vrooli's issue tracker
- **Community**: Join the Vrooli community for discussions and support

---

**Built with Vrooli's recursive intelligence philosophy**: Every feature becomes a capability that other scenarios can leverage, multiplying the system's overall intelligence and value.

🚀 **Ready to transform your social media management?** Get started with the demo account and experience the power of AI-driven social media scheduling!
