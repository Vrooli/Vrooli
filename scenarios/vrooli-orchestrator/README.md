# 🎯 Vrooli Orchestrator

**Contextual Environment Orchestration & Startup Profile Management**

The Vrooli Orchestrator adds contextual intelligence adaptation to Vrooli - the ability to automatically configure the entire environment (resources, scenarios, and UI) for specific user contexts, use cases, and organizational needs. Instead of a one-size-fits-all startup, Vrooli becomes infinitely customizable through startup profiles.

## 🚀 Core Capability

This scenario provides **contextual environment orchestration** - intelligently managing what resources and scenarios run based on user context:

- **👨‍💻 Developer Mode**: Full dev tools, monitoring dashboards, debugging scenarios
- **🏢 Business Mode**: Productivity apps, communication tools, data analysis
- **🎨 Creative Mode**: Content creation, image generation, storytelling tools  
- **🎮 Gaming Mode**: Entertainment scenarios, games, interactive experiences
- **🏠 Household Mode**: Family management, schedules, shopping, automation
- **🎪 Demo Mode**: Curated showcase scenarios for demonstrations

## 🏗️ Architecture

### Components

1. **🗄️ PostgreSQL Database**: Stores profile configurations and activation history
2. **⚡ N8N Workflows**: Handle complex orchestration logic
   - `config-manager.json`: CRUD operations for profiles
   - `profile-activator.json`: Orchestrates resource/scenario startup
3. **🔧 Go API**: REST endpoints for profile management and status
4. **💻 CLI Tool**: `vrooli-orchestrator` command-line interface  
5. **🌐 Web Dashboard**: Visual profile management interface

### Data Flow
```
CLI/UI → Go API → N8N Workflows → Postgres Database
                ↓
       Vrooli Resource/Scenario Control
```

## 📋 Available Profiles

The orchestrator comes with pre-configured profiles:

| Profile | Description | Resources | Use Case |
|---------|-------------|-----------|----------|
| `developer-full` | Complete dev environment | postgres, n8n, ollama, searxng, qdrant, redis | Full-stack development |
| `developer-light` | Essential dev tools | postgres, n8n, ollama | Laptop/limited resources |
| `business-productivity` | Business tools | postgres, n8n, ollama | Office productivity |
| `creative-suite` | Content creation | postgres, n8n, ollama, minio | Creative work |
| `gaming-entertainment` | Games and fun | postgres, n8n | Entertainment |
| `household-management` | Family tools | postgres, n8n, ollama | Home management |
| `demo-showcase` | Curated demos | postgres, n8n, ollama, qdrant | Demonstrations |
| `research-analysis` | Research tools | postgres, n8n, ollama, qdrant, searxng | Research work |
| `minimal` | Basic testing | postgres | Testing/troubleshooting |

## 🛠️ Quick Start

### 1. Install the CLI
```bash
cd cli && ./install.sh
```

### 2. Start the Orchestrator
```bash
vrooli scenario run vrooli-orchestrator
```

### 3. Manage Profiles
```bash
# List available profiles
vrooli-orchestrator list-profiles

# Activate a profile
vrooli-orchestrator activate developer-light

# Check status
vrooli-orchestrator status

# Deactivate current profile  
vrooli-orchestrator deactivate
```

### 4. Access the Dashboard
Open http://localhost:3002 for the visual interface.

## 🔧 CLI Commands

### Profile Management
```bash
# List all profiles
vrooli-orchestrator list-profiles [--json]

# Get profile details
vrooli-orchestrator get-profile <name>

# Create new profile
vrooli-orchestrator create-profile <name> --resources postgres,n8n --scenarios system-monitor

# Update existing profile
vrooli-orchestrator update-profile <name> --description "Updated description"

# Delete profile
vrooli-orchestrator delete-profile <name>
```

### Profile Operations
```bash
# Activate profile (starts resources/scenarios)
vrooli-orchestrator activate <name> [--force]

# Deactivate current profile
vrooli-orchestrator deactivate

# Show current status
vrooli-orchestrator status [--json]
```

### System Commands
```bash
# Show CLI version
vrooli-orchestrator version

# Show help
vrooli-orchestrator help

# Show API endpoint
vrooli-orchestrator api
```

## 🌐 Web Dashboard

The dashboard provides visual profile management:

- **📊 Status Overview**: Current profile, resource count, scenario count
- **🔍 Search & Filter**: Find profiles by name, description, or tags
- **▶️ One-Click Activation**: Activate profiles with visual feedback
- **✏️ Visual Editor**: Create and edit profiles through forms
- **🗑️ Safe Deletion**: Delete profiles with confirmation

Access at: `http://localhost:3002`

## 🛡️ API Endpoints

### Profile Management
- `GET /api/v1/profiles` - List all profiles
- `POST /api/v1/profiles` - Create new profile
- `GET /api/v1/profiles/{name}` - Get specific profile
- `PUT /api/v1/profiles/{name}` - Update profile
- `DELETE /api/v1/profiles/{name}` - Delete profile

### Operations  
- `POST /api/v1/profiles/{name}/activate` - Activate profile
- `POST /api/v1/profiles/current/deactivate` - Deactivate current
- `GET /api/v1/status` - System status

### System
- `GET /health` - Health check

## 📊 Profile Schema

```json
{
  "name": "my-profile",
  "display_name": "My Custom Profile", 
  "description": "Description of what this profile does",
  "resources": ["postgres", "n8n", "ollama"],
  "scenarios": ["system-monitor", "app-debugger"],
  "auto_browser": ["http://localhost:3000/dashboard"],
  "environment_vars": {},
  "idle_shutdown_minutes": 60,
  "dependencies": [],
  "metadata": {
    "target_audience": "developers",
    "resource_footprint": "medium",
    "use_case": "custom_workflow"
  }
}
```

## 🔗 Integration with Other Scenarios

The orchestrator enables intelligent scenario interactions:

```bash
# app-monitor can preload profiles before showing scenarios
vrooli-orchestrator activate gaming-entertainment

# morning-vision-walk can switch to productivity mode
vrooli-orchestrator activate business-productivity  

# deployment-manager can create customer-specific profiles
vrooli-orchestrator create-profile customer-demo --resources postgres --scenarios app-monitor
```

## 🧪 Testing

```bash
# Run CLI tests
cd cli && bats vrooli-orchestrator.bats

# Run scenario validation
vrooli scenario test vrooli-orchestrator

# Test API endpoints
curl http://localhost:15001/health
curl http://localhost:15001/api/v1/profiles
```

## 🚨 Troubleshooting

### Common Issues

**CLI not found**
```bash
# Add to PATH
echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**API connection failed** 
```bash
# Check if orchestrator is running
vrooli scenario run vrooli-orchestrator

# Verify API port
echo $API_PORT  # Should show the port
```

**Profile activation fails**
```bash
# Check resource status
vrooli status

# Try with force flag
vrooli-orchestrator activate <profile> --force

# Check logs in n8n dashboard
```

**Database connection issues**
```bash
# Check postgres resource
vrooli resource start postgres

# Verify database exists
psql -h localhost -p 5433 -U postgres -d postgres -c "SELECT COUNT(*) FROM profiles;"
```

## 🔮 Future Enhancements

- **🤖 AI-Driven Optimization**: Learn usage patterns and optimize profile recommendations
- **📅 Calendar Integration**: Auto-switch profiles based on calendar events
- **🔄 Profile Composition**: Profiles that extend other profiles
- **📈 Usage Analytics**: Track profile effectiveness and resource utilization
- **🌐 Remote Orchestration**: Manage profiles across multiple Vrooli instances

## 🎭 The Intelligence Multiplier

This isn't just profile management - it's **contextual intelligence adaptation**. Every profile activated makes Vrooli smarter for that context. The system learns:

- Which resource combinations work best for different use cases
- How to optimize startup times and resource allocation  
- What scenarios complement each other in different contexts
- User behavior patterns to suggest better profiles

**Result**: Vrooli becomes infinitely adaptable to any user, organization, or context while continuously improving its intelligence through every interaction.

---

**🔧 Built with**: Go, PostgreSQL, N8N, Express.js, Vanilla JavaScript  
**🎯 Purpose**: Make Vrooli contextually intelligent and infinitely adaptable  
**💡 Innovation**: First AI platform with built-in contextual environment orchestration