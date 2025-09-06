# Feature Request Voting Scenario

## Overview
A democratic feature request voting and management system that enables any Vrooli scenario or app to collect, prioritize, and track user-requested features through a Trello-like kanban interface.

## Key Features
- **Multi-tenant Support**: Each scenario gets its own isolated feature request space
- **Flexible Authentication**: Public voting, authenticated-only, or custom rules via scenario-authenticator
- **Trello-like Kanban Board**: Visual management with drag-and-drop between status columns
- **Democratic Voting**: Upvote/downvote system to prioritize features
- **Rich Metadata**: Track priority, tags, comments, and status history
- **API & CLI Access**: Full programmatic access for integration with other scenarios

## Architecture
- **Backend**: Go API with PostgreSQL persistence
- **Frontend**: React with Tailwind CSS and react-beautiful-dnd
- **CLI**: Bash-based CLI for terminal operations
- **Authentication**: Integrates with scenario-authenticator (optional)

## Quick Start

### Setup
```bash
# From scenario directory
vrooli scenario setup feature-request-voting
```

### Run the Scenario
```bash
# Start all services
vrooli scenario run feature-request-voting

# Or start individually
cd scenarios/feature-request-voting
npm run dev  # Start UI
./api/feature-voting-api  # Start API
```

### CLI Usage
```bash
# Install CLI
cd cli && ./install.sh

# List feature requests
feature-voting list study-buddy

# Create a new request
feature-voting create "Dark mode" "Add dark mode support" --scenario study-buddy --priority high

# Vote on a request
feature-voting vote <request-id> --up

# Update request status
feature-voting update <request-id> --status in_development
```

## API Endpoints

### Feature Requests
- `GET /api/v1/scenarios/:id/feature-requests` - List requests for scenario
- `POST /api/v1/feature-requests` - Create new request
- `GET /api/v1/feature-requests/:id` - Get specific request
- `PUT /api/v1/feature-requests/:id` - Update request
- `DELETE /api/v1/feature-requests/:id` - Delete request
- `POST /api/v1/feature-requests/:id/vote` - Vote on request

### Scenarios
- `GET /api/v1/scenarios` - List all scenarios
- `GET /api/v1/scenarios/:id` - Get scenario details

## UI Features

### Kanban Board
- **Columns**: Proposed, Under Review, In Development, Shipped, Won't Fix
- **Drag & Drop**: Move cards between statuses
- **Quick Actions**: Vote directly from cards
- **Rich Cards**: Show votes, priority, tags, comments

### Filtering & Sorting
- Search by title or description
- Filter by priority level
- Sort by votes, date, or priority

### Dark Mode
Toggle between light and dark themes for comfortable viewing

## Integration with Other Scenarios

### As a Provider
Any scenario can integrate feature voting to collect user feedback:

```javascript
// Example: Add feature voting to your scenario
import api from 'feature-voting/api';

// Create a feature request space for your scenario
const scenario = await api.createScenario({
  name: 'my-awesome-app',
  display_name: 'My Awesome App',
  auth_config: { mode: 'authenticated' }
});

// Embed the voting UI
<iframe src={`http://localhost:3000?scenario=${scenario.id}`} />
```

### As a Consumer
Other scenarios can query feature requests for insights:

```bash
# Product manager agent querying high-priority requests
feature-voting list my-app --status proposed --sort priority --json | \
  jq '.requests[] | select(.priority == "high")'
```

## Database Schema

### Core Tables
- `scenarios` - Multi-tenant scenario spaces
- `feature_requests` - Feature request entries
- `votes` - User votes (supports anonymous)
- `comments` - Discussion threads
- `status_changes` - Audit trail
- `scenario_permissions` - Access control

### Views
- `feature_requests_with_user_votes` - Requests with user's vote status
- `feature_requests_summary` - Aggregated statistics by scenario and status

## Configuration

### Environment Variables
```bash
# API Configuration
API_PORT=8080
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=vrooli

# UI Configuration
UI_PORT=3000
VITE_API_URL=http://localhost:8080

# Authentication (optional)
SCENARIO_AUTHENTICATOR_URL=http://localhost:9000
```

### Service Configuration
Edit `.vrooli/service.json` to customize:
- Port ranges
- Resource dependencies
- Health check intervals
- Lifecycle hooks

## Testing

### Run Tests
```bash
# Test API
cd api && go test ./...

# Test UI
cd ui && npm test

# Test CLI
cd cli && bats feature-voting.bats

# Integration test
vrooli scenario test feature-request-voting
```

### Manual Testing
1. Create a feature request via UI
2. Vote on it using different sessions
3. Drag between status columns
4. Verify CLI reflects changes
5. Check API responses

## Deployment

### Local Development
Uses Docker Compose with hot-reload for rapid development

### Production
Can be deployed as:
- Kubernetes pods with Helm charts
- Docker containers with compose
- Standalone services on VMs

### Scaling Considerations
- PostgreSQL handles persistence (scale vertically)
- API is stateless (scale horizontally)
- UI serves static files (CDN-ready)
- Redis can be added for vote caching

## Security

### Authentication Modes
1. **Public**: Anyone can vote and propose
2. **Authenticated**: Requires sign-in via scenario-authenticator
3. **Custom**: Scenario-specific rules

### Data Protection
- User votes can be anonymous (session-based)
- SQL injection prevention via parameterized queries
- CORS configured for cross-origin requests
- Rate limiting recommended for production

## Roadmap

### Version 1.0 (Current)
✅ Multi-tenant support
✅ Kanban board UI
✅ Voting mechanism
✅ CLI tool
✅ REST API

### Version 2.0 (Planned)
- [ ] Real-time updates via WebSockets
- [ ] Comment threads with markdown
- [ ] Email notifications
- [ ] Bulk operations
- [ ] CSV/JSON export

### Version 3.0 (Future)
- [ ] AI-powered duplicate detection
- [ ] Roadmap visualization
- [ ] GitHub/Jira integration
- [ ] Custom workflows
- [ ] Analytics dashboard

## Troubleshooting

### Common Issues

**PostgreSQL Connection Failed**
```bash
# Check if postgres is running
resource-postgres status

# Verify connection settings
psql -h localhost -U postgres -d vrooli
```

**Port Already in Use**
```bash
# Find process using port
lsof -i :8080

# Kill process or change port in .vrooli/service.json
```

**Authentication Not Working**
- Ensure scenario-authenticator is running
- Check CORS settings in API
- Verify auth token is being sent in headers

## Contributing
This scenario follows Vrooli's contribution guidelines. Key principles:
- Keep changes within scenario directory
- Use shared resources via CLI commands
- Follow PRD requirements
- Add tests for new features

## License
Part of the Vrooli platform - see root LICENSE file