// Trigger Investigation API Endpoint
// Initiates Claude-Code investigation for an issue

import { pgClient, httpRequest, publishEvent } from '@windmill/common';

export async function main(
  issueId: string,
  agentId?: string,
  priority: 'low' | 'normal' | 'high' = 'normal',
  dryRun: boolean = false
): Promise<{
  success: boolean;
  message: string;
  runId?: string;
  investigationId?: string;
}> {
  const client = await pgClient();
  
  try {
    // Get issue details
    const issueQuery = `
      SELECT 
        i.*,
        a.name as app_name,
        a.display_name as app_display_name,
        a.deployment_path as app_path,
        a.repository_url as app_repo,
        ag.id as current_agent_id,
        ag.display_name as current_agent_name
      FROM issues i
      JOIN apps a ON i.app_id = a.id
      LEFT JOIN agents ag ON i.assigned_agent_id = ag.id
      WHERE i.id = $1
    `;
    
    const issueResult = await client.query(issueQuery, [issueId]);
    
    if (issueResult.rows.length === 0) {
      return {
        success: false,
        message: 'Issue not found'
      };
    }
    
    const issue = issueResult.rows[0];
    
    // Check if investigation is already in progress
    const activeRunQuery = `
      SELECT id FROM agent_runs 
      WHERE issue_id = $1 AND status IN ('idle', 'working')
    `;
    const activeRunResult = await client.query(activeRunQuery, [issueId]);
    
    if (activeRunResult.rows.length > 0) {
      return {
        success: false,
        message: 'Investigation already in progress',
        runId: activeRunResult.rows[0].id
      };
    }
    
    // Select agent if not specified
    let selectedAgentId = agentId;
    if (!selectedAgentId) {
      // Auto-select best agent based on issue type and past performance
      const agentSelectionQuery = `
        SELECT 
          a.id,
          a.display_name,
          a.capabilities,
          CASE 
            WHEN a.total_runs > 0 
            THEN (a.successful_runs::DECIMAL / a.total_runs * 100)
            ELSE 50 
          END as success_rate
        FROM agents a
        WHERE 
          a.is_active = true
          AND 'investigate' = ANY(a.capabilities)
          AND (
            -- Prefer agents that have handled similar issues
            EXISTS (
              SELECT 1 FROM agent_runs ar
              JOIN issues i2 ON ar.issue_id = i2.id
              WHERE ar.agent_id = a.id
                AND i2.type = $1
                AND ar.successful = true
            )
            OR a.total_runs < 5  -- Give new agents a chance
          )
        ORDER BY 
          success_rate DESC,
          a.total_runs DESC
        LIMIT 1
      `;
      
      const agentResult = await client.query(agentSelectionQuery, [issue.type]);
      
      if (agentResult.rows.length === 0) {
        return {
          success: false,
          message: 'No suitable agent available for investigation'
        };
      }
      
      selectedAgentId = agentResult.rows[0].id;
    }
    
    // Get agent details
    const agentQuery = `
      SELECT * FROM agents WHERE id = $1 AND is_active = true
    `;
    const agentResult = await client.query(agentQuery, [selectedAgentId]);
    
    if (agentResult.rows.length === 0) {
      return {
        success: false,
        message: 'Agent not found or inactive'
      };
    }
    
    const agent = agentResult.rows[0];
    
    // Create agent run record
    const runResult = await client.query(`
      INSERT INTO agent_runs (
        issue_id,
        agent_id,
        status,
        input_context
      ) VALUES ($1, $2, $3, $4)
      RETURNING id
    `, [
      issueId,
      selectedAgentId,
      'idle',
      JSON.stringify({
        issue_title: issue.title,
        issue_description: issue.description,
        issue_type: issue.type,
        issue_priority: issue.priority,
        app_name: issue.app_name,
        error_message: issue.error_message,
        stack_trace: issue.stack_trace,
        affected_files: issue.affected_files,
        environment_info: issue.environment_info
      })
    ]);
    
    const runId = runResult.rows[0].id;
    
    // Update issue status and assignment
    if (!dryRun) {
      await client.query(`
        UPDATE issues 
        SET 
          status = 'investigating',
          assigned_agent_id = $1,
          assigned_at = CURRENT_TIMESTAMP
        WHERE id = $2
      `, [selectedAgentId, issueId]);
    }
    
    // Prepare prompt for Claude
    const userPrompt = agent.user_prompt_template
      .replace('{{app_name}}', issue.app_display_name)
      .replace('{{issue_title}}', issue.title)
      .replace('{{issue_description}}', issue.description)
      .replace('{{issue_type}}', issue.type)
      .replace('{{issue_priority}}', issue.priority)
      .replace('{{error_message}}', issue.error_message || 'N/A')
      .replace('{{stack_trace}}', issue.stack_trace || 'N/A')
      .replace('{{affected_files}}', (issue.affected_files || []).join(', '))
      .replace('{{app_path}}', issue.app_path || 'N/A')
      .replace('{{app_repo}}', issue.app_repo || 'N/A');
    
    // Trigger n8n workflow for Claude investigation
    const n8nWebhookUrl = process.env.N8N_WEBHOOK_URL || 'http://localhost:5678/webhook/investigate-issue';
    
    const investigationPayload = {
      runId,
      issueId,
      agentId: selectedAgentId,
      systemPrompt: agent.system_prompt,
      userPrompt,
      maxTokens: agent.max_tokens,
      temperature: agent.temperature,
      model: agent.model,
      priority,
      dryRun,
      context: {
        issue,
        agent,
        appPath: issue.app_path,
        repoUrl: issue.app_repo
      }
    };
    
    // Send to n8n for processing
    const n8nResponse = await httpRequest({
      url: n8nWebhookUrl,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(investigationPayload)
    });
    
    if (n8nResponse.status !== 200) {
      // Update run status to failed
      await client.query(`
        UPDATE agent_runs 
        SET status = 'failed', 
            error_message = $1,
            completed_at = CURRENT_TIMESTAMP
        WHERE id = $2
      `, ['Failed to trigger n8n workflow', runId]);
      
      return {
        success: false,
        message: 'Failed to trigger investigation workflow',
        runId
      };
    }
    
    const investigationId = n8nResponse.data.executionId;
    
    // Update run with investigation ID
    await client.query(`
      UPDATE agent_runs 
      SET 
        status = 'working',
        metadata = jsonb_set(
          COALESCE(metadata, '{}'::jsonb),
          '{investigation_id}',
          $1::jsonb
        )
      WHERE id = $2
    `, [JSON.stringify(investigationId), runId]);
    
    // Publish event for real-time updates
    await publishEvent('investigation_started', {
      issueId,
      runId,
      investigationId,
      agentId: selectedAgentId,
      agentName: agent.display_name
    });
    
    // Add comment to issue
    await client.query(`
      INSERT INTO issue_comments (
        issue_id,
        author_type,
        author_id,
        author_name,
        content,
        is_state_change,
        old_state,
        new_state
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `, [
      issueId,
      'system',
      'investigation-trigger',
      'System',
      `Investigation started using agent: ${agent.display_name}`,
      true,
      issue.status,
      'investigating'
    ]);
    
    return {
      success: true,
      message: `Investigation started with agent: ${agent.display_name}`,
      runId,
      investigationId
    };
    
  } catch (error) {
    console.error('Error triggering investigation:', error);
    return {
      success: false,
      message: `Error: ${error.message}`
    };
  } finally {
    await client.end();
  }
}