# Scenario Generator V1

An autonomous scenario development pipeline that uses Claude Code to iteratively generate, refine, validate, and deploy complete Vrooli scenarios. This system provides a campaign-based dashboard for managing multiple scenario development projects.

## 🎯 What This Scenario Does

The Scenario Generator V1 transforms customer requests into fully functional Vrooli scenarios through an autonomous AI pipeline:

1. **Backlog Management**: File-based queue system for managing scenario requests through YAML files
2. **Campaign Management**: Organize scenarios by business category (SaaS, AI Assistants, Automation, Analytics)
3. **Iterative Planning**: Claude Code analyzes requirements and creates detailed implementation plans with configurable refinement loops
4. **Autonomous Implementation**: Generates complete scenario files, workflows, and configurations
5. **Validation Pipeline**: Tests scenarios using direct execution with automatic bug fixing
6. **Pattern Learning**: Stores all interactions and issues in a database for continuous improvement

### Business Value
- **Revenue Generation**: Each scenario typically generates $10K-50K in value
- **Autonomous Development**: Reduces scenario creation time from weeks to hours
- **Pattern Learning**: Improves quality over time through accumulated experience
- **Scalability**: Can generate multiple scenarios concurrently

## 🏗️ Architecture

### Resources Used
- **postgres** (5433): Stores campaigns, scenarios, generation logs, and improvement analytics
- **claude-code**: AI-driven scenario generation and refinement via Vrooli's Claude Code integration
- **redis** (6380): Queue management and session caching (optional)

### Pipeline Phases

#### 1. Planning Phase
- User submits scenario request through React web interface
- Claude analyzes requirements via `vrooli resource claude` and creates implementation plan
- Configurable refinement iterations (1-5) improve plan quality
- Final plan includes complete architecture and specifications

#### 2. Implementation Phase  
- Claude generates all scenario files from refined plan via `vrooli resource claude`
- Includes database schemas, API code, configurations, and documentation
- Bug fixing iterations (1-5) improve code quality and completeness
- All generated files tracked and managed by the pipeline

#### 3. Validation Phase
- Direct scenario validation using `vrooli scenario test`
- Claude Code analyzes any validation errors and provides fixes
- Iterative debugging (1-10 attempts) until validation passes
- Successful scenarios are automatically deployed

#### 4. Analytics Phase
- All interactions, issues, and solutions stored in PostgreSQL
- Pattern analysis identifies common problems and optimal solutions
- Continuous improvement through learning from past generations

## 🚀 Quick Start

### Prerequisites
Ensure the following resources are available:
- PostgreSQL (port 5433)
- Redis (port 6380) [optional]
- Vrooli installation with Claude resource configured
- Go 1.21+ for API compilation

### Deployment

1. **Deploy the scenario:**
   ```bash
   cd scenarios/scenario-generator-v1
   ./deployment/startup.sh deploy
   ```

2. **Verify deployment:**
   ```bash
   ./test.sh
   ```

3. **Access the dashboard:**
   Open http://localhost:${UI_PORT} (default port varies by configuration)

### Creating Your First Scenario

#### Method 1: Using the Backlog System (Recommended)
1. **Add to Backlog** - Create a YAML file in `backlog/pending/`:
   ```bash
   cp backlog/templates/scenario-template.yaml backlog/pending/001-my-scenario.yaml
   # Edit the file with your scenario details
   ```
2. **View in UI** - Navigate to the Backlog tab to see your item
3. **Generate** - Click "Generate Now" to start processing
4. **Track Progress** - Item moves automatically: pending → in-progress → completed

#### Method 2: Direct Generation
1. Navigate to the dashboard
2. Go to "Generate New" tab
3. Fill in the scenario details:
   - **Description**: Detailed explanation of what you want built
   - **Complexity**: Simple ($5K-15K), Intermediate ($15K-35K), or Advanced ($35K-75K)
   - **Category**: Business type (marketplace, saas, productivity, etc.)
   - **Iterations**: Configure refinement loops for each phase
4. Submit and monitor progress in real-time

### Example Scenario Request

```
Create a SaaS business for managing freelance invoices and payments. The system should allow freelancers to create professional invoices, track payment status, send automated reminders, and generate financial reports. Include client management, project tracking, and integration with popular payment providers. The UI should be modern and mobile-responsive.
```

## 📁 Backlog System

The Scenario Generator includes a powerful folder-based backlog system for managing scenario requests:

### Features
- **File-Based Management**: Edit YAML files directly with any text editor
- **Automatic Monitoring**: File watcher detects changes in real-time
- **State Tracking**: Items move through folders: pending → in-progress → completed/failed
- **Priority System**: Filename prefixes (001-, 002-) determine processing order
- **Version Control**: Git tracks all backlog changes automatically

### Folder Structure
```
backlog/
├── pending/        # New requests waiting to be processed
├── in-progress/    # Currently being generated
├── completed/      # Successfully generated scenarios
├── failed/         # Failed attempts (can be retried)
├── templates/      # Template files for new items
└── README.md       # Detailed backlog documentation
```

### Quick Commands
```bash
# Add new scenario to backlog
cp backlog/templates/scenario-template.yaml backlog/pending/001-new-idea.yaml

# View backlog status
ls -la backlog/pending/*.yaml | wc -l  # Count pending items

# Bulk import scenarios
cp ~/my-scenarios/*.yaml backlog/pending/

# Check for high-priority items
ls backlog/pending/0[0-9][0-9]-*.yaml  # Lists items 001-099 (high priority)
```

### Benefits
- ✅ Manual editing flexibility - modify YAML files directly
- ✅ Bulk operations - add multiple scenarios at once
- ✅ External integration - other tools can add items by creating files
- ✅ Transparent queue - see exactly what's pending at a glance
- ✅ Easy backup - simple file operations for archiving

## 📊 React UI Features

### Tab-Based Interface
- **Backlog**: Manage scenario queue with folder-based YAML files (pending, in-progress, completed, failed)
- **Browse Scenarios**: View all generated scenarios with search and filtering
- **Generate New**: Form interface for creating new scenarios from requirements
- **Templates**: Browse and use pre-built scenario templates

### Scenario Management
- **Status Monitoring**: Real-time status indicators (✅ Completed, 🔄 Generating, ❌ Failed, ⏰ Pending)
- **Detailed Views**: Modal overlays showing complete scenario information
- **Search & Filter**: Find scenarios by name, category, complexity, or status
- **Revenue Tracking**: View estimated business value for each scenario

### Modern Interface
- **Dark Theme**: AI-focused design with cyan and blue accents
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Real-time Updates**: Live progress tracking and status updates
- **Error Handling**: User-friendly error messages and retry options

## 🔧 Configuration

### Environment Variables

```bash
# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_DB=vrooli
POSTGRES_USER=postgres
POSTGRES_PASSWORD=

# Service Endpoints  
REDIS_HOST=localhost
REDIS_PORT=6380

# Feature Flags
CLAUDE_CODE_AVAILABLE=true
```

### Configuration Files

- `initialization/configuration/app-config.json`: UI settings and iteration defaults
- `initialization/configuration/resource-urls.json`: Service endpoint configuration
- `initialization/configuration/feature-flags.json`: Feature toggle configuration

## 🧪 Testing

### Integration Tests
```bash
# Run all tests
./test.sh

# Health check only
./deployment/startup.sh check
```

### Test Coverage
- Database schema and connectivity
- Pipeline orchestration validation
- React UI functionality
- Claude Code integration
- File structure and JSON validity
- SQL syntax validation

## 📈 Monitoring & Analytics

### Success Metrics
- **Generation Success Rate**: Target >85% of scenarios successfully deployed
- **Average Generation Time**: Target <30 minutes per scenario
- **First-Attempt Validation**: Target >70% pass validation without fixes
- **Revenue Generated**: Track cumulative business value created

### Pattern Analysis
The system automatically learns from:
- Common implementation patterns that work well
- Frequent validation errors and their solutions  
- Resource utilization optimization opportunities
- User request patterns and preferences

### Debugging

**Common Issues:**
- **Claude Code Authentication**: Run `claude auth login` to authenticate
- **Pipeline Failures**: Check API logs for execution errors
- **Database Connection**: Verify PostgreSQL credentials and connectivity
- **Validation Errors**: Review direct scenario test output for specific issues

**Useful Commands:**
```bash
# Check Claude Code status
resource-claude-code status

# View API health
curl http://localhost:${API_PORT}/health

# Check database connectivity
psql -h localhost -p 5433 -U postgres -d vrooli -c "SELECT COUNT(*) FROM scenarios;"

# Monitor Redis queues (if enabled)
redis-cli -p 6380 LLEN scenario-planning-queue
```

## 🔄 Continuous Improvement

The Scenario Generator V1 implements a learning system that:
- **Tracks Success Patterns**: Identifies what implementations work best
- **Learns from Failures**: Analyzes common issues and develops standard solutions
- **Optimizes Resources**: Improves generation speed and resource utilization
- **Predicts Success**: Develops scoring models for scenario viability

This creates a compound intelligence effect where each scenario generated makes the system better at generating future scenarios.

## 📚 Advanced Usage

### Batch Generation
Generate multiple related scenarios:
1. Create scenarios with similar requirements
2. Use pattern learning to accelerate subsequent generations
3. Monitor resource utilization across concurrent generations

### Custom Prompts
Modify prompt templates in the `prompts/` directory:
- `initial-planning-prompt.md`: Requirements analysis and architecture design
- `plan-refinement-prompt.md`: Plan improvement and optimization  
- `implementation-prompt.md`: Code generation and file creation
- `validation-prompt.md`: Error analysis and bug fixing

### Analytics Queries
Query the database for insights:
```sql
-- Success rate by complexity
SELECT complexity, 
       COUNT(*) as total,
       COUNT(*) FILTER (WHERE success = true) as successful,
       ROUND(COUNT(*) FILTER (WHERE success = true) * 100.0 / COUNT(*), 2) as success_rate
FROM scenarios 
GROUP BY complexity;

-- Common validation issues
SELECT issue_category, COUNT(*) as frequency
FROM validation_results 
WHERE success = false
GROUP BY issue_category 
ORDER BY frequency DESC;

-- Average generation time by category
SELECT category,
       ROUND(AVG(total_duration_minutes), 2) as avg_minutes
FROM scenarios 
WHERE completed_at IS NOT NULL
GROUP BY category;
```

## 🤝 Contributing

To extend the Scenario Generator V1:

1. **Add New Prompt Templates**: Create specialized prompts for specific scenario types
2. **Enhance Pattern Learning**: Improve the analytics and learning algorithms
3. **Extend Validation**: Add more comprehensive testing and quality checks
4. **Optimize Pipeline**: Improve generation pipeline efficiency and error handling
5. **Expand UI Features**: Add more dashboard capabilities and user experience improvements

The modular architecture makes it easy to enhance individual components without affecting the overall system.

---

**Ready to start generating scenarios?** Run the deployment script and access the dashboard to begin creating autonomous AI-powered business solutions\!
README_EOF < /dev/null
