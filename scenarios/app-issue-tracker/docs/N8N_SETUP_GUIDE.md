# N8N Workflow Setup Guide

This guide explains how to manually set up N8N workflows for the App Issue Tracker.

## Prerequisites

- N8N running at http://localhost:5678
- App Issue Tracker API running at http://localhost:8090
- Qdrant running at http://localhost:6333
- PostgreSQL database with schema initialized

## Manual Workflow Import

Since automated import requires authentication setup, follow these steps to manually import workflows:

### 1. Access N8N UI

1. Open http://localhost:5678 in your browser
2. Complete the initial setup if this is a fresh install
3. Create an account or log in

### 2. Import Investigation Workflow

1. In N8N, click "Add Workflow" or "+"
2. Click on the "..." menu and select "Import from File"
3. Upload the workflow file: `initialization/automation/n8n/claude-investigation.json`
4. Review the imported workflow nodes:
   - **Webhook Trigger**: Receives investigation requests
   - **Get Issue Details**: Queries PostgreSQL for issue information  
   - **Prepare Investigation**: Formats data for Claude Code
   - **Claude Code Investigation**: Calls our claude-investigator.sh script
   - **Parse Response**: Extracts structured data from investigation results
   - **Update Issue**: Stores results back to PostgreSQL
   - **Update Vector DB**: Stores embeddings in Qdrant
   - **Webhook Response**: Returns success response

### 3. Configure Webhook URL

1. Click on the "Webhook Trigger" node
2. Set the webhook path to: `investigate-issue`
3. Note the full webhook URL: `http://localhost:5678/webhook/investigate-issue`

### 4. Configure Database Connections

For each PostgreSQL node:
1. Click on the node
2. Create/select PostgreSQL credentials:
   - Host: localhost
   - Database: issue_tracker
   - User: postgres
   - Password: [YOUR_POSTGRES_PASSWORD]
   - Port: 5432

### 5. Test the Workflow

1. Save and activate the workflow
2. Use the "Test" button or make a POST request to the webhook:

```bash
curl -X POST "http://localhost:5678/webhook/investigate-issue" \
  -H "Content-Type: application/json" \
  -d '{
    "issue_id": "test-123",
    "agent_id": "agent-001",
    "priority": "high",
    "title": "Test Issue",
    "description": "Testing the workflow",
    "type": "bug",
    "project_path": "/path/to/project",
    "prompt_template": "Investigate this test issue"
  }'
```

## Workflow Architecture

### Data Flow

```
API Request → Webhook → Database Query → Claude Investigation → Parse Results → Update Database → Response
```

### Claude Code Integration

The workflow calls our `claude-investigator.sh` script with these parameters:
- `issue_id`: Unique identifier for the issue
- `agent_id`: ID of the investigating agent
- `project_path`: Path to the codebase to investigate
- `prompt_template`: Investigation instructions for Claude

### Expected Output

The Claude investigation returns structured JSON:
```json
{
  "issue_id": "issue-123",
  "investigation_report": "Full investigation text...",
  "root_cause": "Identified root cause",
  "suggested_fix": "Recommended solution",
  "confidence_score": 8,
  "affected_files": ["file1.js", "file2.py"],
  "investigation_completed_at": "2024-01-15T10:30:00Z",
  "status": "completed"
}
```

## Troubleshooting

### Common Issues

1. **Webhook not accessible**: 
   - Ensure N8N is running and accessible
   - Check firewall settings
   - Verify webhook URL in API configuration

2. **Database connection fails**:
   - Verify PostgreSQL is running
   - Check connection credentials
   - Ensure database schema is initialized

3. **Claude Code script fails**:
   - Verify claude-code CLI is installed and accessible
   - Check script permissions: `chmod +x scripts/claude-investigator.sh`
   - Review script logs for error details

4. **Qdrant connection fails**:
   - Ensure Qdrant is running at localhost:6333
   - Verify collection exists: `issue_embeddings`
   - Check Qdrant logs for errors

### Testing Individual Components

Test each component separately:

```bash
# Test Claude investigator script
./scripts/claude-investigator.sh investigate "test-issue" "test-agent" "/path/to/code" "Test prompt"

# Test Qdrant connection
curl "http://localhost:6333/collections"

# Test API endpoint
curl "http://localhost:8090/health"

# Test PostgreSQL connection
# Set your PostgreSQL password as environment variable first:
# export POSTGRES_PASSWORD="your_password_here"
PGPASSWORD="${POSTGRES_PASSWORD}" psql -h localhost -p 5432 -U postgres -d issue_tracker -c "SELECT COUNT(*) FROM issues;"
```

## Monitoring

Use the monitoring script to check workflow status:

```bash
./scripts/monitor_workflows.sh
```

## Alternative: Direct Investigation

If N8N setup is complex, the API falls back to direct Claude Code investigation:

1. The API detects N8N unavailability
2. Calls `fallbackDirectInvestigation()` function
3. Still updates issue status and provides investigation results
4. Less sophisticated workflow but functional for basic use cases

## Next Steps

1. Import the workflow file manually through N8N UI
2. Configure all database connections
3. Test webhook endpoint
4. Monitor execution logs
5. Set up error handling and retry logic
6. Consider adding more workflows for fix generation and PR creation

For automated deployment, N8N authentication and API credentials need to be configured first.