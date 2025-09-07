# Bookmark Intelligence Hub

Transform your social media bookmarks into organized, actionable intelligence across platforms.

## 🎯 Overview

The Bookmark Intelligence Hub automatically discovers, extracts, categorizes, and suggests actions for your bookmarks from Reddit, X (Twitter), TikTok, and other social media platforms. It creates a permanent bridge between social media consumption and productive action-taking within the Vrooli ecosystem.

## ✨ Key Features

- **Multi-Platform Integration**: Supports Reddit, X (Twitter), TikTok with modular architecture
- **AI-Powered Categorization**: Automatically categorizes bookmarks with learning capabilities
- **Action Suggestions**: Suggests relevant actions for other Vrooli scenarios
- **Multi-Tenant Support**: Separate profiles for different use cases
- **Real-Time Processing**: Configurable polling intervals for fresh content
- **Cross-Scenario Integration**: Feeds data to recipe-book, workout-plan-generator, and more

## 🚀 Quick Start

### Prerequisites

- PostgreSQL database (managed by Vrooli)
- Huginn automation platform (managed by Vrooli)
- Browserless headless browser service (managed by Vrooli)

### Installation

1. Run the scenario setup:
   ```bash
   vrooli scenario run bookmark-intelligence-hub
   ```

2. Open the dashboard:
   ```
   http://localhost:3200
   ```

3. Configure your social media platforms in the dashboard

### CLI Usage

```bash
# Check status
bookmark-intelligence-hub status

# List profiles
bookmark-intelligence-hub profile list

# Sync all bookmarks
bookmark-intelligence-hub sync

# Sync specific platform
bookmark-intelligence-hub sync reddit

# View pending actions
bookmark-intelligence-hub actions

# Approve an action
bookmark-intelligence-hub actions approve action-123
```

## 🏗️ Architecture

### Components

- **API Server** (Go): Core bookmark processing and categorization logic
- **Web Dashboard** (JavaScript): Profile management and action approval interface
- **Platform Integrations**: Modular connectors for Reddit, Twitter, TikTok
- **Database Layer**: PostgreSQL for bookmark storage and analytics
- **Automation Layer**: Huginn agents for social media monitoring
- **Fallback System**: Browserless for JavaScript-heavy content extraction

### Data Flow

1. **Collection**: Huginn agents monitor social media platforms for bookmarks
2. **Processing**: AI categorizes content and suggests actions
3. **Storage**: Bookmarks stored with metadata and categorization
4. **Action Generation**: System suggests actions for other scenarios
5. **User Approval**: Manual review and approval of suggested actions
6. **Execution**: Approved actions trigger integrations with other scenarios

## 🔧 Configuration

### Environment Variables

```bash
# API Configuration
API_PORT=15200                    # API server port
UI_PORT=3200                      # Dashboard port
DATABASE_URL=postgres://...       # PostgreSQL connection string

# Platform Credentials
REDDIT_CLIENT_ID=your_client_id
REDDIT_CLIENT_SECRET=your_secret
REDDIT_USERNAME=your_username
REDDIT_PASSWORD=your_password

TWITTER_SESSION_COOKIE=your_cookie
TIKTOK_SESSION_COOKIE=your_cookie

# Service URLs
HUGINN_URL=http://localhost:3000
BROWSERLESS_URL=http://localhost:3001
```

### Platform Setup

#### Reddit
1. Create Reddit app at https://reddit.com/prefs/apps
2. Set credentials in environment variables
3. Enable Reddit integration in dashboard

#### X (Twitter)
1. Export session cookie from browser
2. Set TWITTER_SESSION_COOKIE environment variable
3. Configure browserless fallback for reliability

#### TikTok
1. Export session cookie from browser
2. Set TIKTOK_SESSION_COOKIE environment variable
3. Enable TikTok integration in dashboard

## 📊 Categorization

### Built-in Categories

- **Programming**: Code, APIs, development resources
- **Recipes**: Cooking instructions, food content
- **Fitness**: Workouts, exercise routines
- **Travel**: Destinations, travel tips
- **News**: Current events, breaking news
- **Education**: Learning resources, tutorials
- **Entertainment**: Movies, games, entertainment

### Custom Categories

Create custom categories with:
- Keyword matching rules
- Platform-specific patterns
- Integration actions for other scenarios
- Priority ordering

## 🔄 Cross-Scenario Integration

### Supported Integrations

- **recipe-book**: Automatically adds cooking content
- **workout-plan-generator**: Suggests fitness routines
- **research-assistant**: Archives content for research
- **travel-planner**: Adds destinations to travel lists
- **code-library**: Saves programming resources

### Adding New Integrations

1. Define action suggestion in platform integration
2. Configure target scenario and action data
3. Test integration with approval workflow
4. Monitor execution success rates

## 🧪 Testing

### Run Tests

```bash
# API endpoint tests
./test/test-bookmark-processing.sh

# Full scenario test
vrooli scenario test bookmark-intelligence-hub
```

### Test Data

The system includes demo profiles and sample bookmarks for testing:
- Demo profile with sample categorization rules
- Mock bookmarks from all supported platforms
- Sample action suggestions and approvals

## 🔒 Security & Privacy

- **Data Encryption**: Platform credentials encrypted at rest
- **Local Processing**: Content analysis performed locally
- **Profile Isolation**: Multi-tenant data separation
- **Audit Trail**: Complete logging of processing and actions
- **No External AI**: Uses local models when possible

## 📈 Analytics

### Metrics Tracked

- Processing accuracy and improvement over time
- Category distribution and trends
- Action approval/rejection rates
- Platform performance and reliability
- User satisfaction scores

### Dashboard Features

- Real-time processing statistics
- Category breakdown visualizations
- Platform health monitoring
- Action queue management
- Historical trend analysis

## 🛠️ Development

### Project Structure

```
bookmark-intelligence-hub/
├── api/                          # Go API server
├── cli/                          # Command-line interface
├── ui/                           # Web dashboard
├── integrations/platforms/       # Platform-specific integrations
├── initialization/               # Database and automation setup
├── test/                         # Test scripts
└── PRD.md                       # Product requirements document
```

### Adding New Platforms

1. Implement `PlatformIntegration` interface
2. Add platform-specific configuration
3. Create Huginn automation agents
4. Configure browserless fallback
5. Add categorization rules
6. Test integration workflow

### Database Schema

- `bookmark_profiles`: User profile configurations
- `bookmark_items`: Individual bookmarks and metadata
- `action_items`: Suggested actions and approval status
- `category_rules`: Categorization rules and learning data
- `platform_integrations`: Platform connection status
- `processing_stats`: Analytics and performance metrics

## 🚨 Troubleshooting

### Common Issues

**API not responding**
```bash
# Check API health
curl http://localhost:15200/health

# Check logs
pm::logs bookmark-intelligence-hub-api
```

**Platform sync failing**
- Verify credentials are set correctly
- Check rate limiting status
- Ensure Huginn agents are active
- Test browserless fallback

**Low categorization accuracy**
- Review and update category rules
- Add platform-specific keywords
- Train on user feedback
- Check content extraction quality

### Debug Mode

Enable verbose logging:
```bash
export DEBUG=true
vrooli scenario run bookmark-intelligence-hub
```

## 📚 Documentation

- [PRD.md](./PRD.md) - Product requirements and technical architecture
- [API Documentation](./api/) - REST API specification
- [CLI Reference](./cli/) - Command-line interface guide
- [Integration Guide](./integrations/) - Platform integration details

## 🤝 Contributing

This scenario is part of the Vrooli ecosystem. Improvements should:

1. Enhance cross-scenario compatibility
2. Improve categorization accuracy
3. Add new platform integrations
4. Optimize performance and reliability

## 📄 License

Part of the Vrooli platform ecosystem. See main project license.

---

**Generated with Vrooli AI** - This scenario represents a permanent capability that enhances the entire Vrooli ecosystem by transforming passive social media consumption into productive, actionable intelligence.