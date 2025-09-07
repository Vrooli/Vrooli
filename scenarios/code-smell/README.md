# Code Smell Detector

A self-improving code quality guardian that continuously detects, tracks, and fixes code smell violations across the Vrooli codebase.

## Purpose

This scenario adds permanent code quality enforcement to Vrooli, preventing technical debt accumulation and maintaining consistent code standards. It combines AI-powered pattern detection with hot-reloadable rules to catch both general and Vrooli-specific code smells.

## Key Features

- **Hot-Reloadable Rules**: Add or modify detection rules without restarting
- **AI-Powered Detection**: Uses resource-claude-code for complex pattern analysis
- **Auto-Fix Capability**: Safely fixes trivial issues automatically
- **Review Queue**: Dangerous changes require manual approval via UI
- **Learning System**: Improves detection accuracy over time
- **Vrooli-Specific Rules**: Catches patterns unique to our codebase

## Vrooli-Specific Smells Detected

- Hard-coded ports (should use environment variables)
- Hard-coded home directories (should use ${HOME})
- Hard-coded Vrooli paths (should use ${VROOLI_ROOT})
- Direct n8n workflow creation (should use shared workflows)
- Missing PRD.md files in scenarios
- Direct resource API calls (should use CLI)
- And many more...

## Installation

```bash
# Install dependencies
npm install

# Install CLI
./cli/install.sh

# Start API server
go run api/main.go

# Start UI
python3 -m http.server 3090 --directory ui
```

## Usage

### CLI Commands

```bash
# Check status
code-smell status

# Analyze a directory
code-smell analyze ./scenarios --auto-fix

# List all rules
code-smell rules list

# View review queue
code-smell queue

# Apply a specific fix
code-smell fix <violation-id> --action approve
```

### API Endpoints

- `POST /api/v1/code-smell/analyze` - Analyze files for smells
- `GET /api/v1/code-smell/rules` - Get all configured rules
- `POST /api/v1/code-smell/fix` - Apply or reject a fix
- `GET /api/v1/code-smell/queue` - Get violations needing review
- `POST /api/v1/code-smell/learn` - Submit pattern for learning

### UI Access

Navigate to `http://localhost:3090` to access the dashboard with:
- Real-time statistics
- Review queue for dangerous fixes
- Rule management interface
- History timeline

## Adding Custom Rules

Create a new YAML file in `initialization/rules/` with your rules:

```yaml
rules:
  - id: my-custom-rule
    name: "My Custom Rule"
    category: needs-approval
    risk_level: moderate
    pattern:
      type: regex
      value: 'pattern_to_match'
    message: "Description of the issue"
```

Rules are hot-reloaded automatically within 1 second.

## Integration with Other Scenarios

This scenario provides code quality checks for:
- `git-manager` - Pre-commit hooks
- `scenario-improvement-loop` - Quality gates
- `ci-cd-healer` - Build failure prevention
- `technical-debt-manager` - Debt prioritization

## Architecture

- **Rules Engine**: Node.js with hot-reload via chokidar
- **API Server**: Go for high-performance analysis
- **UI**: React with real-time updates
- **Storage**: PostgreSQL for rules and history
- **AI Integration**: resource-claude-code for complex patterns

## Performance

- Analyzes 100+ files per minute
- Sub-500ms response for single file analysis
- 15-minute cache for repeated queries
- Incremental analysis via visited-tracker integration

## Future Enhancements

- Machine learning pattern detection
- Git hook integration
- Cross-repository analysis
- Team-specific rule sets
- Export reports in multiple formats

## Revenue Potential

$20K-$40K per enterprise deployment through:
- Reduced maintenance costs (30% time savings)
- Prevented production bugs
- Faster onboarding of new developers
- Compliance with code standards

## Style

Technical but approachable UI with:
- Dark theme optimized for developers
- GitHub-like diff viewer
- Keyboard shortcuts for power users
- Real-time statistics dashboard