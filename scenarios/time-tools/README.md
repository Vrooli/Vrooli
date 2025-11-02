# Time Tools - Comprehensive Temporal Operations Platform

> **Enterprise-grade temporal operations and scheduling platform for timezone conversions, duration calculations, and intelligent scheduling**

## 🎯 Overview

Time Tools provides a comprehensive temporal operations platform that enables all Vrooli scenarios to perform timezone conversions, duration calculations, date arithmetic, scheduling optimization, and time-based analysis without implementing custom temporal logic.

## ✨ Features

### Core Capabilities
- **Timezone Conversion** - Convert between any global timezones with DST handling
- **Duration Calculations** - Calculate time between dates with business hours support
- **Scheduling Optimization** - Find optimal meeting times across multiple participants
- **Conflict Detection** - Identify and resolve scheduling conflicts automatically
- **Business Time** - Handle working hours, holidays, and business day calculations
- **Event Management** - Create, manage, and track calendar events

### Advanced Features
- Multi-timezone display for global teams
- Recurring event patterns with complex rules
- Time analytics and usage insights
- Holiday and special date management
- Calendar integration support

## 🚀 Quick Start

### Prerequisites
- Vrooli CLI installed
- PostgreSQL and Redis resources available
- Go 1.21+ (for development)

### Installation

```bash
# Start the scenario
vrooli scenario run time-tools

# Install CLI (if not auto-installed)
cd scenarios/time-tools/cli
./install.sh

# Verify installation
time-tools help
```

## 📚 Usage

### CLI Commands

```bash
# Convert timezone
time-tools convert '2024-01-15T10:00:00Z' UTC 'America/New_York'

# Calculate duration
time-tools duration '2024-01-15T09:00:00Z' '2024-01-15T17:00:00Z' --business-hours-only

# Find optimal meeting time
time-tools schedule --participants alice,bob,charlie --duration 60 --earliest 2024-01-15

# Detect scheduling conflicts
time-tools conflicts '2024-01-15T14:00:00Z' '2024-01-15T15:00:00Z' --organizer alice

# Show current time in timezone
time-tools now 'Asia/Tokyo'

# Format time in various styles
time-tools format '2024-01-15T10:00:00Z' human --timezone 'America/Los_Angeles'
```

### API Endpoints

```bash
# Get the dynamic API port
API_PORT=$(vrooli scenario port time-tools API_PORT)
API_BASE="http://localhost:${API_PORT}"

# Health check
curl $API_BASE/api/v1/health

# Timezone conversion
curl -X POST $API_BASE/api/v1/time/convert \
  -H "Content-Type: application/json" \
  -d '{
    "time": "2024-01-15T10:00:00Z",
    "from_timezone": "UTC",
    "to_timezone": "America/New_York"
  }'

# Duration calculation
curl -X POST $API_BASE/api/v1/time/duration \
  -H "Content-Type: application/json" \
  -d '{
    "start_time": "2024-01-15T09:00:00Z",
    "end_time": "2024-01-15T17:00:00Z",
    "business_hours_only": true
  }'

# Optimal scheduling
curl -X POST $API_BASE/api/v1/schedule/optimal \
  -H "Content-Type: application/json" \
  -d '{
    "participants": ["alice", "bob"],
    "duration_minutes": 60,
    "earliest_date": "2024-01-15",
    "latest_date": "2024-01-22"
  }'
```

## 🏗️ Architecture

### Components
- **API Server** - Go-based REST API with PostgreSQL integration
- **CLI Tool** - Lightweight wrapper around API endpoints
- **Web UI** - Modular dashboard for visual time management
- **Database** - PostgreSQL with temporal data models

### Data Models
- `scheduled_events` - Calendar events and appointments
- `timezone_definitions` - Timezone data with DST rules
- `recurrence_patterns` - Complex recurring event patterns
- `business_hours` - Working hours configuration
- `holidays` - Holiday and special date definitions
- `time_analytics` - Usage metrics and insights

## 🧪 Testing

```bash
# Run all tests
vrooli scenario test time-tools

# Test API endpoints
API_PORT=$(vrooli scenario port time-tools API_PORT)
curl http://localhost:${API_PORT}/api/v1/health

# Test CLI commands
time-tools status
```

## 📊 Business Value

### Target Markets
- Global enterprises requiring cross-timezone coordination
- Scheduling platforms and calendar applications
- Productivity tools and project management systems
- Remote teams and distributed organizations

### Revenue Potential
- **Initial**: $10,000 - Setup and customization
- **Recurring**: $2,000/month - SaaS subscriptions
- **Expansion**: $20,000 - Enterprise features
- **Total Estimate**: $40,000

### Key Differentiators
- Comprehensive temporal operations in one platform
- AI-powered scheduling optimization
- Business time intelligence
- Seamless timezone handling with DST support

## 🔧 Development

### Building from Source

```bash
# Build API server
cd api
go build -o time-tools-api

# Run with custom port
PORT=8080 ./time-tools-api
```

### Database Schema

The platform uses PostgreSQL with timezone-aware timestamps and advanced temporal functions. See `initialization/storage/postgres/schema.sql` for full schema.

## 📝 Configuration

Configuration is managed through `service.json` and environment variables:

- `API_PORT` - API server port (default: 7500-7599)
- `POSTGRES_HOST` - PostgreSQL host
- `POSTGRES_PORT` - PostgreSQL port
- `POSTGRES_DB` - Database name
- `REDIS_HOST` - Redis host for caching

## 🤝 Contributing

This scenario follows Vrooli contribution guidelines. Key areas for enhancement:

1. Additional timezone intelligence
2. Calendar platform integrations
3. Advanced scheduling algorithms
4. Mobile app support
5. Real-time collaboration features

## 📄 License

MIT License - See LICENSE file for details

## 🔗 Related Scenarios

- `calendar-integration-hub` - Multi-platform calendar sync
- `meeting-coordinator` - AI-powered meeting scheduling
- `deadline-management-system` - Project timeline tracking
- `workflow-scheduler` - Time-based automation

## 💡 Support

For issues or questions:
- Create an issue in the Vrooli repository
- Contact support@vrooli.com
- Join our Discord community

---

**Time Tools** - Making time management intelligent and effortless.
