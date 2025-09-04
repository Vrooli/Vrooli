# Code Tidiness Manager

## 🧹 Overview

The Code Tidiness Manager is an intelligent technical debt detection and cleanup automation system for Vrooli. It continuously monitors the codebase for inefficiencies, redundancies, and cleanup opportunities, providing both automated cleanup scripts and detailed analysis for complex architectural improvements.

## 🎯 Purpose

This scenario adds a **permanent maintenance intelligence capability** to Vrooli that:
- Automatically detects technical debt and cleanup opportunities
- Generates safe, confidence-scored cleanup scripts
- Learns from accepted/rejected suggestions to improve over time
- Provides metrics and insights about codebase health
- Enables other scenarios to maintain clean workspaces

## 💡 Why This Matters

As Vrooli grows through recursive self-improvement, code cleanliness becomes critical:
- **Clean workspace guarantee** - Agents can trust the codebase is maintained
- **Reduced context noise** - Less clutter means better AI performance
- **Resource optimization** - Identifies waste that can be reclaimed
- **Quality baseline** - Establishes standards for all future code generation

## 🚀 Quick Start

```bash
# Run initial setup
vrooli scenario setup code-tidiness-manager

# Start the service
vrooli scenario run code-tidiness-manager

# Run a quick scan
code-tidiness-manager scan --types backup_files,temp_files

# View suggestions
code-tidiness-manager suggestions --severity medium

# Execute cleanup (with dry run first)
code-tidiness-manager cleanup all --dry-run
code-tidiness-manager cleanup all --force
```

## 📊 Features

### Core Capabilities
- **Multi-pattern scanning** - Detects 30+ types of cleanup opportunities
- **Confidence scoring** - Each suggestion includes safety confidence
- **Dry-run mode** - Preview changes before execution
- **Learning system** - Improves suggestions based on feedback
- **Batch operations** - Handle multiple cleanups efficiently

### Detection Categories
- **File Cleanup** - Backup files, temp files, OS artifacts
- **Directory Cleanup** - Empty directories, orphaned folders
- **Code Quality** - Unused imports, dead code, TODOs
- **Architectural** - Duplicate scenarios, pattern violations
- **Security** - Hardcoded credentials, exposed secrets
- **Performance** - Large files, unoptimized assets

### Dashboard Features
- Real-time scan progress
- Cleanup history tracking
- Technical debt metrics
- Trend visualization
- One-click cleanup execution

## 🏗️ Architecture

### Components
- **Go API Server** - High-performance REST API
- **PostgreSQL** - Scan history and pattern storage
- **Redis** - Caching for performance
- **n8n Workflows** - Automated scanning and cleanup
- **Web Dashboard** - Clean, professional UI
- **CLI Tool** - Command-line interface

### Data Flow
1. Scanner identifies issues using pattern matching
2. Results stored in PostgreSQL with confidence scores
3. Dashboard presents categorized suggestions
4. User reviews and approves cleanups
5. System learns from accepted/rejected patterns
6. Metrics tracked for continuous improvement

## 🔌 Integration Points

### API Endpoints
- `POST /api/v1/tidiness/scan` - Trigger scan
- `GET /api/v1/tidiness/suggestions` - Retrieve suggestions
- `POST /api/v1/tidiness/cleanup` - Execute cleanups
- `GET /api/v1/tidiness/metrics` - Get debt metrics

### CLI Commands
```bash
code-tidiness-manager status        # Show system status
code-tidiness-manager scan          # Run scan
code-tidiness-manager suggestions   # List suggestions
code-tidiness-manager cleanup       # Execute cleanups
code-tidiness-manager register-rule # Add custom rule
```

### Events Published
- `tidiness.scan.completed` - Scan finished
- `tidiness.cleanup.executed` - Cleanup performed
- `tidiness.debt.threshold` - Debt level alert

## 📈 Metrics & Monitoring

### Key Metrics
- **Debt Score** - Overall technical debt quantification
- **Cleanup Success Rate** - Percentage of successful cleanups
- **Automation Rate** - Percentage of issues auto-cleanable
- **Time Saved** - Estimated developer hours saved

### Health Checks
- API endpoint availability
- Database connectivity
- Redis cache status
- Scan job queue depth

## 🎨 UI Style

The dashboard follows a **clean, professional design** inspired by GitHub Insights:
- Light theme with severity-based color coding
- Card-based layout for clear information hierarchy
- Satisfying cleanup progress animations
- Mobile-responsive for on-the-go monitoring

## 🔒 Security Considerations

- **Read-only by default** - Explicit permission required for cleanup
- **Dangerous operation prevention** - Blocks risky commands
- **Audit trail** - All actions logged with rollback capability
- **No credential scanning** - Avoids overlap with security tools

## 🚦 Deployment

### Local Development
```bash
cd scenarios/code-tidiness-manager
vrooli develop
```

### Production
- Supports Docker Compose deployment
- Kubernetes StatefulSet for persistence
- Serverless scanning via AWS Lambda
- Auto-scaling based on scan queue

## 💰 Value Proposition

### For Developers
- Saves 10-20 hours per month on maintenance
- Reduces debugging time from cleaner code
- Prevents accumulation of technical debt

### For Vrooli
- Enables recursive self-improvement through clean code
- Multiplies agent effectiveness
- Creates foundation for advanced refactoring scenarios

### Revenue Model
- **Free Tier**: 100 scans/month
- **Pro**: Unlimited scans - $99/month
- **Enterprise**: Custom rules & priority support - $499/month

## 🛣️ Roadmap

### Version 1.0 (Current)
- ✅ Core scanning engine
- ✅ Basic cleanup automation
- ✅ Web dashboard
- ✅ CLI interface

### Version 2.0 (Planned)
- [ ] Machine learning pattern recognition
- [ ] Cross-scenario duplicate detection
- [ ] Automated refactoring suggestions
- [ ] CI/CD pipeline integration

### Future Vision
- Fully autonomous code maintenance
- Predictive debt prevention
- Industry-specific cleanup patterns
- Multi-repository support

## 🤝 Contributing

This scenario follows Vrooli's contribution guidelines. Key areas for improvement:
- Adding new cleanup patterns
- Improving confidence scoring algorithms
- Enhancing the dashboard UI
- Optimizing scan performance

## 📚 Related Scenarios

### Enhances
- **All scenarios** benefit from clean code

### Enhanced By
- **git-manager** - Pre-commit cleanup integration
- **code-quality-enforcer** - Uses tidiness metrics
- **scenario-health-monitor** - Consumes debt data

### Enables
- **legacy-code-modernizer** - Uses cleanup patterns
- **resource-optimizer** - Leverages waste detection
- **performance-bottleneck-hunter** - Builds on inefficiency data

## 🐛 Troubleshooting

### Common Issues

**Scan taking too long**
- Reduce scan scope with `--paths` flag
- Use `--types` to limit detection patterns
- Enable caching with Redis

**False positives**
- Adjust confidence thresholds in settings
- Add exclusion patterns for your workflow
- Submit feedback to improve learning

**Database connection issues**
- Check PostgreSQL is running
- Verify connection string in config
- Ensure database migrations completed

## 📞 Support

For issues or questions:
- Check `/docs/code-tidiness-manager/` for detailed docs
- Run `code-tidiness-manager help` for CLI guidance
- View dashboard help section for UI assistance

---

*Code Tidiness Manager - Keeping Vrooli's intelligence clean and efficient, forever.*