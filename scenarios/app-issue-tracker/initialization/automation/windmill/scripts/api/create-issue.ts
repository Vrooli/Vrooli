// Create Issue API Endpoint
// Allows apps to report new issues to the tracker

import { pgClient, validateApiToken, generateEmbedding, publishEvent } from '@windmill/common';

export async function main(
  apiToken: string,
  title: string,
  description: string,
  type: 'bug' | 'feature' | 'improvement' | 'documentation' | 'performance' | 'security' = 'bug',
  priority: 'critical' | 'high' | 'medium' | 'low' = 'medium',
  errorMessage?: string,
  stackTrace?: string,
  affectedFiles?: string[],
  affectedComponents?: string[],
  environmentInfo?: Record<string, any>,
  reporterName?: string,
  reporterEmail?: string,
  reporterId?: string,
  externalId?: string,
  tags?: string[],
  metadata?: Record<string, any>
): Promise<{
  success: boolean;
  issueId?: string;
  message: string;
  similarIssues?: Array<{ id: string; title: string; similarity: number }>;
}> {
  const client = await pgClient();
  
  try {
    // Validate API token and get app info
    const appResult = await client.query(
      'SELECT id, name, display_name FROM apps WHERE api_token = $1 AND status = $2',
      [apiToken, 'active']
    );
    
    if (appResult.rows.length === 0) {
      return {
        success: false,
        message: 'Invalid API token or inactive app'
      };
    }
    
    const app = appResult.rows[0];
    
    // Check for duplicate by external ID if provided
    if (externalId) {
      const duplicateCheck = await client.query(
        'SELECT id FROM issues WHERE app_id = $1 AND external_id = $2',
        [app.id, externalId]
      );
      
      if (duplicateCheck.rows.length > 0) {
        return {
          success: false,
          message: 'Issue with this external ID already exists',
          issueId: duplicateCheck.rows[0].id
        };
      }
    }
    
    // Generate embedding for semantic search
    const fullText = `${title} ${description} ${errorMessage || ''} ${tags?.join(' ') || ''}`;
    const embedding = await generateEmbedding(fullText);
    
    // Search for similar issues
    const similarQuery = await client.query(`
      SELECT id, title, 
        1 - (embedding <=> $1::vector) as similarity
      FROM issues 
      WHERE app_id = $2 
        AND status NOT IN ('closed', 'wont_fix')
        AND embedding IS NOT NULL
      ORDER BY embedding <=> $1::vector
      LIMIT 5
    `, [embedding, app.id]);
    
    const similarIssues = similarQuery.rows
      .filter(row => row.similarity > 0.8)
      .map(row => ({
        id: row.id,
        title: row.title,
        similarity: row.similarity
      }));
    
    // Check if this might be a duplicate
    if (similarIssues.length > 0 && similarIssues[0].similarity > 0.95) {
      // Update occurrence count for the existing issue
      await client.query(`
        UPDATE issues 
        SET occurrence_count = occurrence_count + 1,
            last_seen_at = CURRENT_TIMESTAMP
        WHERE id = $1
      `, [similarIssues[0].id]);
      
      return {
        success: false,
        message: 'Very similar issue already exists - updated occurrence count',
        issueId: similarIssues[0].id,
        similarIssues
      };
    }
    
    // Insert the new issue
    const insertResult = await client.query(`
      INSERT INTO issues (
        app_id, external_id, title, description, type, priority,
        error_message, stack_trace, affected_files, affected_components,
        environment_info, reporter_name, reporter_email, reporter_id,
        tags, metadata, embedding, embedding_generated_at
      ) VALUES (
        $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, CURRENT_TIMESTAMP
      ) RETURNING id
    `, [
      app.id,
      externalId,
      title,
      description,
      type,
      priority,
      errorMessage,
      stackTrace,
      affectedFiles || [],
      affectedComponents || [],
      environmentInfo || {},
      reporterName,
      reporterEmail,
      reporterId,
      tags || [],
      metadata || {},
      embedding
    ]);
    
    const issueId = insertResult.rows[0].id;
    
    // Check if this issue matches any pattern groups
    await checkPatternGroups(client, issueId, title, description, errorMessage, stackTrace);
    
    // Publish event for real-time updates
    await publishEvent('issue_created', {
      issueId,
      appId: app.id,
      appName: app.display_name,
      title,
      type,
      priority
    });
    
    // Log API request
    await client.query(`
      INSERT INTO api_requests (app_id, endpoint, method, status_code, response_time_ms)
      VALUES ($1, $2, $3, $4, $5)
    `, [app.id, '/api/issues/create', 'POST', 201, Date.now() - startTime]);
    
    return {
      success: true,
      issueId,
      message: 'Issue created successfully',
      similarIssues: similarIssues.length > 0 ? similarIssues : undefined
    };
    
  } catch (error) {
    console.error('Error creating issue:', error);
    return {
      success: false,
      message: `Error creating issue: ${error.message}`
    };
  } finally {
    await client.end();
  }
}

async function checkPatternGroups(
  client: any,
  issueId: string,
  title: string,
  description: string,
  errorMessage?: string,
  stackTrace?: string
): Promise<void> {
  // Get all active pattern groups
  const patterns = await client.query(
    'SELECT id, common_keywords, common_stack_trace_patterns FROM pattern_groups'
  );
  
  const issueText = `${title} ${description} ${errorMessage || ''} ${stackTrace || ''}`.toLowerCase();
  
  for (const pattern of patterns.rows) {
    let matches = false;
    
    // Check keyword matches
    if (pattern.common_keywords && pattern.common_keywords.length > 0) {
      const keywordMatches = pattern.common_keywords.filter(
        (keyword: string) => issueText.includes(keyword.toLowerCase())
      );
      if (keywordMatches.length >= 2) {
        matches = true;
      }
    }
    
    // Check stack trace patterns
    if (!matches && stackTrace && pattern.common_stack_trace_patterns) {
      for (const stackPattern of pattern.common_stack_trace_patterns) {
        if (stackTrace.includes(stackPattern)) {
          matches = true;
          break;
        }
      }
    }
    
    if (matches) {
      // Update issue with pattern group
      await client.query(
        'UPDATE issues SET pattern_group_id = $1 WHERE id = $2',
        [pattern.id, issueId]
      );
      
      // Update pattern group statistics
      await client.query(`
        UPDATE pattern_groups 
        SET total_issues = total_issues + 1,
            active_issues = active_issues + 1,
            affected_app_ids = array_append(
              COALESCE(affected_app_ids, '{}'), 
              (SELECT app_id FROM issues WHERE id = $1)
            )
        WHERE id = $2
      `, [issueId, pattern.id]);
      
      break; // Only assign to first matching pattern
    }
  }
}

const startTime = Date.now();