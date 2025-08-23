/**
 * Parse unstructured text into actionable tasks
 * Integrates with n8n text-parser-workflow via webhook
 */

import { PostgreSQLResource } from 'windmill-client'

type ParseTextInput = {
  app_id: string;
  raw_text: string;
  input_type?: 'markdown' | 'plaintext' | 'structured_list';
  submitted_by?: string;
}

type ParseTextResult = {
  success: boolean;
  session_id?: string;
  tasks_created: number;
  tasks?: Array<{
    id: string;
    title: string;
    description: string;
    status: string;
    priority: string;
    tags: string[];
    estimated_hours: number | null;
    created_at: string;
  }>;
  error?: string;
  message?: string;
}

export async function main(
  postgresql: PostgreSQLResource,
  input: ParseTextInput
): Promise<ParseTextResult> {
  try {
    // Validate required inputs
    if (!input.app_id || !input.raw_text?.trim()) {
      return {
        success: false,
        tasks_created: 0,
        error: "Missing required fields: app_id and raw_text"
      };
    }

    // Get app details and verify API token
    const appQuery = `
      SELECT id, name, api_token, display_name 
      FROM apps 
      WHERE id = $1
    `;
    const appResult = await postgresql.sql`SELECT id, name, api_token, display_name FROM apps WHERE id = ${input.app_id}`;
    
    if (appResult.length === 0) {
      return {
        success: false,
        tasks_created: 0,
        error: "App not found"
      };
    }

    const app = appResult[0];

    // Call n8n text-parser workflow
    const n8nPayload = {
      app_id: input.app_id,
      api_token: app.api_token,
      raw_text: input.raw_text,
      input_type: input.input_type || 'markdown',
      submitted_by: input.submitted_by || 'windmill'
    };

    console.log('🤖 Calling n8n text parser workflow...');
    
    const n8nResponse = await fetch('http://localhost:5679/webhook/parse-text', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(n8nPayload)
    });

    if (!n8nResponse.ok) {
      const errorText = await n8nResponse.text();
      throw new Error(`n8n workflow failed: ${n8nResponse.status} ${errorText}`);
    }

    const result = await n8nResponse.json() as ParseTextResult;

    // Log parsing results
    if (result.success) {
      console.log(`✅ Successfully parsed ${result.tasks_created} tasks from text input`);
      console.log(`📝 Session ID: ${result.session_id}`);
      
      // Update app task count
      await postgresql.sql`
        UPDATE apps 
        SET total_tasks = total_tasks + ${result.tasks_created}, 
            updated_at = CURRENT_TIMESTAMP 
        WHERE id = ${input.app_id}
      `;
    } else {
      console.log(`❌ Text parsing failed: ${result.error}`);
    }

    return result;

  } catch (error) {
    console.error('❌ Error in parse-unstructured:', error);
    return {
      success: false,
      tasks_created: 0,
      error: error instanceof Error ? error.message : 'Unknown error occurred'
    };
  }
}

/**
 * Helper function to validate text content quality
 */
export function validateTextInput(text: string): { valid: boolean; issues: string[] } {
  const issues: string[] = [];
  
  if (text.length < 10) {
    issues.push('Text is too short (minimum 10 characters)');
  }
  
  if (text.length > 10000) {
    issues.push('Text is too long (maximum 10,000 characters)');
  }

  // Check for potential actionable content
  const actionablePatterns = [
    /\b(add|create|build|implement|fix|update|remove|delete|improve|optimize)\b/gi,
    /\b(todo|task|feature|bug|issue)\b/gi,
    /^[-*+]\s/gm, // Bullet points
    /^\d+\.\s/gm  // Numbered lists
  ];

  const hasActionableContent = actionablePatterns.some(pattern => pattern.test(text));
  if (!hasActionableContent) {
    issues.push('Text may not contain actionable tasks (no action words or list formatting found)');
  }

  return {
    valid: issues.length === 0,
    issues
  };
}

/**
 * Helper function to get parsing statistics for an app
 */
export async function getParsingStats(
  postgresql: PostgreSQLResource,
  app_id: string
): Promise<{
  total_sessions: number;
  successful_sessions: number;
  total_tasks_parsed: number;
  avg_tasks_per_session: number;
  recent_sessions: Array<{
    id: string;
    processed_at: string;
    tasks_extracted: number;
    raw_text_preview: string;
  }>;
}> {
  const stats = await postgresql.sql`
    SELECT 
      COUNT(*) as total_sessions,
      COUNT(*) FILTER (WHERE processed = true) as successful_sessions,
      COALESCE(SUM(tasks_extracted), 0) as total_tasks_parsed,
      COALESCE(AVG(tasks_extracted) FILTER (WHERE processed = true), 0) as avg_tasks_per_session
    FROM unstructured_sessions 
    WHERE app_id = ${app_id}
  `;

  const recentSessions = await postgresql.sql`
    SELECT id, processed_at, tasks_extracted, 
           LEFT(raw_text, 100) as raw_text_preview
    FROM unstructured_sessions 
    WHERE app_id = ${app_id} AND processed = true 
    ORDER BY processed_at DESC 
    LIMIT 5
  `;

  return {
    total_sessions: parseInt(stats[0].total_sessions),
    successful_sessions: parseInt(stats[0].successful_sessions), 
    total_tasks_parsed: parseInt(stats[0].total_tasks_parsed),
    avg_tasks_per_session: parseFloat(stats[0].avg_tasks_per_session),
    recent_sessions: recentSessions.map(session => ({
      id: session.id,
      processed_at: session.processed_at,
      tasks_extracted: session.tasks_extracted,
      raw_text_preview: session.raw_text_preview + (session.raw_text_preview.length === 100 ? '...' : '')
    }))
  };
}