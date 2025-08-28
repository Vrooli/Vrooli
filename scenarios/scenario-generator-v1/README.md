# Scenario Generator V1

An autonomous scenario development pipeline that uses Claude Code to iteratively generate, refine, validate, and deploy complete Vrooli scenarios. This system provides a campaign-based dashboard for managing multiple scenario development projects.

## 🎯 What This Scenario Does

The Scenario Generator V1 transforms customer requests into fully functional Vrooli scenarios through an autonomous AI pipeline:

1. **Campaign Management**: Organize scenarios by business category (SaaS, AI Assistants, Automation, Analytics)
2. **Iterative Planning**: Claude Code analyzes requirements and creates detailed implementation plans with configurable refinement loops
3. **Autonomous Implementation**: Generates complete scenario files, workflows, and configurations
4. **Validation Pipeline**: Tests scenarios using scenario-to-app.sh with automatic bug fixing
5. **Pattern Learning**: Stores all interactions and issues in a database for continuous improvement

### Business Value
- **Revenue Generation**: Each scenario typically generates $10K-50K in value
- **Autonomous Development**: Reduces scenario creation time from weeks to hours
- **Pattern Learning**: Improves quality over time through accumulated experience
- **Scalability**: Can generate multiple scenarios concurrently

## 🏗️ Architecture

### Resources Used
- **windmill** (5681): Campaign dashboard UI with scenario management
- **n8n** (5678): Orchestrates the entire generation pipeline 
- **postgres** (5433): Stores campaigns, scenarios, generation logs, and improvement analytics
- **minio** (9000): Stores generated scenario files, plans, and artifacts
- **claude-code**: AI-driven scenario generation and refinement
- **redis** (6380): Queue management and session caching

### Pipeline Phases

#### 1. Planning Phase
- User submits scenario request through Windmill dashboard
- Claude Code analyzes requirements and creates implementation plan
- Configurable refinement iterations (1-5) improve plan quality
- Final plan stored in MinIO with complete architecture and specifications

#### 2. Implementation Phase  
- Claude Code generates all scenario files from refined plan
- Includes database schemas, workflows, configurations, and documentation
- Bug fixing iterations (1-5) improve code quality and completeness
- All generated files stored in MinIO with version control

#### 3. Validation Phase
- scenario-to-app.sh performs dry-run validation
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
- n8n (port 5678) 
- Windmill (port 5681)
- MinIO (port 9000)
- Redis (port 6380)
- Claude Code CLI installed and authenticated

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
   Open http://localhost:5681/apps/scenario-generator

### Creating Your First Scenario

1. Navigate to the dashboard
2. Select a campaign tab (SaaS Businesses, AI Assistants, etc.)
3. Click "Add New Scenario"
4. Fill in the scenario details:
   - **Description**: Detailed explanation of what you want built
   - **Complexity**: Simple ($5K-15K), Intermediate ($15K-35K), or Advanced ($35K-75K)
   - **Category**: Business type (marketplace, saas, productivity, etc.)
   - **Iterations**: Configure refinement loops for each phase
5. Submit and monitor progress in real-time

### Example Scenario Request

```
Create a SaaS business for managing freelance invoices and payments. The system should allow freelancers to create professional invoices, track payment status, send automated reminders, and generate financial reports. Include client management, project tracking, and integration with popular payment providers. The UI should be modern and mobile-responsive.
```

## 📊 Dashboard Features

### Campaign Organization
- **Tab-based Navigation**: Scenarios grouped by business category
- **Progress Tracking**: Real-time status updates and completion percentages  
- **Resource Management**: View estimated revenue and business metrics
- **Batch Operations**: Manage multiple related scenarios together

### Scenario Management
- **Status Monitoring**: Track scenarios through planning, implementation, and validation phases
- **Detailed Logs**: View all Claude Code interactions and system operations
- **File Explorer**: Browse generated scenario files and configurations
- **Error Analytics**: Understand common issues and improvement patterns

### Configuration Options
- **Iteration Counts**: Customize refinement loops for each phase
- **Complexity Levels**: Set appropriate difficulty and revenue targets
- **Resource Allocation**: Configure timeouts and concurrent limits
- **Feature Flags**: Enable/disable advanced features

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
N8N_HOST=localhost
N8N_PORT=5678
WINDMILL_HOST=localhost
WINDMILL_PORT=5681
MINIO_HOST=localhost
MINIO_PORT=9000
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
- n8n workflow validation
- Windmill app deployment
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
- **Workflow Failures**: Check n8n logs for execution errors
- **Database Connection**: Verify PostgreSQL credentials and connectivity
- **Validation Errors**: Review scenario-to-app.sh output for specific issues

**Useful Commands:**
```bash
# Check Claude Code status
resource-claude-code status

# View n8n workflows
curl http://localhost:5678/api/v1/workflows

# Check database connectivity
psql -h localhost -p 5433 -U postgres -d vrooli -c "SELECT COUNT(*) FROM scenarios;"

# Monitor Redis queues
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
4. **Optimize Workflows**: Improve n8n workflow efficiency and error handling
5. **Expand UI Features**: Add more dashboard capabilities and user experience improvements

The modular architecture makes it easy to enhance individual components without affecting the overall system.

---

**Ready to start generating scenarios?** Run the deployment script and access the dashboard to begin creating autonomous AI-powered business solutions\!
README_EOF < /dev/null
