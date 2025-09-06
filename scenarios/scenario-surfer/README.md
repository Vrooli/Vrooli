# Scenario Surfer 🌊

**StumbleUpon-style discovery interface for browsing and exploring running Vrooli scenarios**

Scenario Surfer provides a serendipitous browsing experience for discovering the growing ecosystem of Vrooli capabilities. Think of it as your window into the compound intelligence system - a way to surf through scenarios, discover unexpected combinations, and easily report issues when you find them.

## 🎯 What It Does

- **Discover Scenarios**: Automatically finds all running, healthy scenarios with web interfaces
- **Category Browsing**: Filter scenarios by purpose (work tools, games, development, etc.)
- **Random Exploration**: Hit the random button for serendipitous discovery
- **Seamless Navigation**: Minimal chrome that doesn't compete with scenario content
- **Issue Reporting**: One-click bug reports with automatic screenshot capture
- **Keyboard Shortcuts**: Space for random, Escape for back, R for report

## 🚀 Quick Start

```bash
# Start the service
vrooli scenario run scenario-surfer

# Open in browser
scenario-surfer open

# Or browse specific categories
scenario-surfer open work    # Business tools
scenario-surfer open fun     # Games and entertainment  
scenario-surfer open dev     # Development tools
```

## 🖥️ CLI Commands

```bash
# Check service status
scenario-surfer status

# List all discoverable scenarios
scenario-surfer discover

# Filter by category or health
scenario-surfer discover --category work
scenario-surfer discover --status healthy

# Open browser interface
scenario-surfer open [category]

# Get JSON output
scenario-surfer status --json
scenario-surfer discover --json
```

## 🎮 Browser Navigation

### Keyboard Shortcuts
- **Space**: Random scenario
- **Escape**: Go back in history
- **R**: Report issue with current scenario

### Interface Elements
- **Back Button**: Navigate through your browsing history
- **Category Filter**: Browse scenarios by purpose/type
- **Scenario Dropdown**: Jump directly to any available scenario
- **Random Button**: Serendipitous discovery
- **Report Button**: Easy issue reporting with screenshots

## 🔧 How It Works

1. **Discovery**: Calls `vrooli scenario status --json` to find running scenarios
2. **Filtering**: Only shows healthy scenarios with responding UI ports
3. **Categorization**: Reads scenario tags from `.vrooli/service.json` files
4. **Embedding**: Uses iframes to seamlessly display scenario UIs
5. **Issue Reporting**: Integrates with app-issue-tracker for bug reports

## 🏗️ Architecture

### API Server (Go)
- Scenario discovery via `vrooli scenario status --json`
- Health filtering and categorization
- Issue reporting proxy to app-issue-tracker
- Screenshot capture via browserless integration

### Web Interface
- Minimal chrome design that doesn't distract from scenarios
- Responsive navigation with keyboard shortcuts
- Iframe embedding with CORS handling
- Auto-hiding navigation on inactivity

### CLI Interface
- Status checking and scenario discovery
- Browser launching with category pre-filtering
- JSON output for programmatic access

## 📱 User Experience

### The StumbleUpon Philosophy
- **Serendipitous Discovery**: Find scenarios you didn't know existed
- **Minimal Friction**: One click to explore, one click to report issues  
- **Content First**: Chrome stays out of the way of scenario UIs
- **Quick Navigation**: Keyboard shortcuts for power users

### Category Modes
- **Work & Business**: Productivity tools, managers, generators
- **Fun & Games**: Entertainment, pickers, creative tools
- **Development**: Debugging, monitoring, code tools
- **AI & Intelligence**: Agents, reasoning, analysis tools

## 🔄 Integration

### Upstream Dependencies
- **vrooli CLI**: For scenario status discovery
- **Running Scenarios**: Content to browse and explore
- **app-issue-tracker**: Destination for bug reports

### Downstream Value
- **Quality Feedback**: Improves all scenarios through easy issue reporting
- **Usage Discovery**: Shows which capabilities are actually used
- **Ecosystem Growth**: Makes the compound intelligence visible and accessible

## 🎨 Design Principles

1. **Invisible Chrome**: UI elements fade away to focus on scenario content
2. **Serendipitous Discovery**: Random exploration over structured navigation
3. **Immediate Feedback**: One-click issue reporting with context
4. **Progressive Disclosure**: Start simple, reveal complexity as needed
5. **Keyboard First**: Power users can navigate entirely via keyboard

## 🔧 Development

### Local Development
```bash
# Install dependencies
cd ui && npm install

# Start API
cd api && go run main.go

# Start UI  
cd ui && npm start
```

### Adding New Categories
Categories are automatically derived from scenario tags in `.vrooli/service.json` files. Common tag patterns:
- Business: `business`, `product`, `manager`  
- Development: `dev`, `debug`, `api`, `test`
- Fun: `game`, `fun`, `entertainment`
- AI: `ai`, `intelligence`, `agent`

## 🌊 Why "Surfer"?

Just like web surfing, scenario surfing is about discovery, exploration, and following interesting paths through interconnected content. The ocean metaphor captures the vast, interconnected nature of the Vrooli ecosystem - scenarios are islands of capability in an ocean of possibilities.

---

*Scenario Surfer makes the invisible visible - transforming Vrooli's compound intelligence from abstract concept to tangible, explorable reality.*