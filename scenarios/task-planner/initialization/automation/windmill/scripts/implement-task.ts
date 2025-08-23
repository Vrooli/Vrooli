/**
 * Implement a staged task using Claude Code
 * Integrates with n8n implementation-workflow via webhook
 */

import { PostgreSQLResource } from 'windmill-client'

type ImplementTaskInput = {
  task_id: string;
  override_staging?: boolean; // Allow implementing non-staged tasks
  implementation_notes?: string; // Additional context for implementation
}

type ImplementTaskResult = {
  success: boolean;
  task_id: string;
  status: string;
  title?: string;
  implementation_completed: boolean;
  files_modified: string[];
  tokens_used: number;
  estimated_cost: number;
  completed_at?: string;
  research_artifacts: number;
  error?: string;
  details?: string;
}

export async function main(
  postgresql: PostgreSQLResource,
  input: ImplementTaskInput
): Promise<ImplementTaskResult> {
  try {
    // Validate task exists and get details
    const taskQuery = await postgresql.sql`
      SELECT t.*, a.name as app_name, a.display_name as app_display_name, a.repository_url
      FROM tasks t
      JOIN apps a ON t.app_id = a.id
      WHERE t.id = ${input.task_id}
    `;

    if (taskQuery.length === 0) {
      return {
        success: false,
        task_id: input.task_id,
        status: 'not_found',
        implementation_completed: false,
        files_modified: [],
        tokens_used: 0,
        estimated_cost: 0,
        research_artifacts: 0,
        error: 'Task not found'
      };
    }

    const task = taskQuery[0];

    // Check if task is ready for implementation
    if (task.status !== 'staged' && !input.override_staging) {
      return {
        success: false,
        task_id: input.task_id,
        status: task.status,
        implementation_completed: false,
        files_modified: [],
        tokens_used: 0,
        estimated_cost: 0,
        research_artifacts: 0,
        error: `Task is not staged for implementation (current: ${task.status}). Use override_staging=true to force implementation.`
      };
    }

    // Check if task has required planning information
    if (!task.implementation_plan && !input.override_staging) {
      return {
        success: false,
        task_id: input.task_id,
        status: task.status,
        implementation_completed: false,
        files_modified: [],
        tokens_used: 0,
        estimated_cost: 0,
        research_artifacts: 0,
        error: 'Task lacks implementation plan. Please research the task first or use override_staging=true.'
      };
    }

    console.log(`🚀 Starting implementation for task: "${task.title}"`);
    console.log(`📋 App: ${task.app_display_name} (${task.app_name})`);
    console.log(`📊 Priority: ${task.priority}`);
    console.log(`⏱️ Estimated: ${task.estimated_hours || 'Unknown'} hours`);

    // Call n8n implementation workflow
    const n8nPayload = {
      task_id: input.task_id,
      implementation_notes: input.implementation_notes
    };

    const n8nResponse = await fetch('http://localhost:5679/webhook/implement-task', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(n8nPayload)
    });

    if (!n8nResponse.ok) {
      const errorText = await n8nResponse.text();
      throw new Error(`Implementation workflow failed: ${n8nResponse.status} ${errorText}`);
    }

    const result = await n8nResponse.json() as ImplementTaskResult;

    // Log implementation results
    if (result.success && result.implementation_completed) {
      console.log(`✅ Task implementation completed successfully`);
      console.log(`📁 Files Modified: ${result.files_modified.length}`);
      console.log(`🧠 Tokens Used: ${result.tokens_used}`);
      console.log(`💰 Estimated Cost: $${result.estimated_cost.toFixed(4)}`);
      
      // Update app completion statistics
      await updateAppImplementationStats(postgresql, task.app_id, result, task);
    } else if (result.success) {
      console.log(`⚠️ Task implementation partially successful`);
      console.log(`❌ Error: ${result.error}`);
    } else {
      console.log(`❌ Task implementation failed: ${result.error}`);
    }

    return result;

  } catch (error) {
    console.error('❌ Error in implement-task:', error);
    return {
      success: false,
      task_id: input.task_id,
      status: 'error',
      implementation_completed: false,
      files_modified: [],
      tokens_used: 0,
      estimated_cost: 0,
      research_artifacts: 0,
      error: error instanceof Error ? error.message : 'Unknown error occurred'
    };
  }
}

/**
 * Get implementation readiness score for a task
 */
export async function getImplementationReadiness(
  postgresql: PostgreSQLResource,
  task_id: string
): Promise<{
  readiness_score: number; // 0-100
  readiness_factors: {
    has_investigation_report: boolean;
    has_implementation_plan: boolean;
    has_technical_requirements: boolean;
    has_research_artifacts: boolean;
    confidence_score_adequate: boolean;
  };
  missing_requirements: string[];
  recommendation: string;
}> {
  const task = await postgresql.sql`
    SELECT 
      t.*,
      COUNT(ra.id) as artifact_count
    FROM tasks t
    LEFT JOIN research_artifacts ra ON t.id = ra.task_id
    WHERE t.id = ${task_id}
    GROUP BY t.id
  `;

  if (task.length === 0) {
    return {
      readiness_score: 0,
      readiness_factors: {
        has_investigation_report: false,
        has_implementation_plan: false,
        has_technical_requirements: false,
        has_research_artifacts: false,
        confidence_score_adequate: false
      },
      missing_requirements: ['Task not found'],
      recommendation: 'Task does not exist'
    };
  }

  const taskData = task[0];
  const factors = {
    has_investigation_report: !!taskData.investigation_report,
    has_implementation_plan: !!taskData.implementation_plan,
    has_technical_requirements: !!taskData.technical_requirements,
    has_research_artifacts: taskData.artifact_count > 0,
    confidence_score_adequate: (taskData.confidence_score || 0) >= 0.6
  };

  const missingRequirements: string[] = [];
  let score = 0;

  // Calculate score based on factors
  if (factors.has_investigation_report) {
    score += 25;
  } else {
    missingRequirements.push('Investigation report');
  }

  if (factors.has_implementation_plan) {
    score += 30;
  } else {
    missingRequirements.push('Implementation plan');
  }

  if (factors.has_technical_requirements) {
    score += 20;
  } else {
    missingRequirements.push('Technical requirements');
  }

  if (factors.has_research_artifacts) {
    score += 15;
  } else {
    missingRequirements.push('Research artifacts');
  }

  if (factors.confidence_score_adequate) {
    score += 10;
  } else {
    missingRequirements.push('Adequate confidence score (>= 60%)');
  }

  // Generate recommendation
  let recommendation: string;
  if (score >= 85) {
    recommendation = '🟢 Task is ready for implementation';
  } else if (score >= 65) {
    recommendation = '🟡 Task is mostly ready, consider addressing missing requirements';
  } else if (score >= 45) {
    recommendation = '🟠 Task needs more preparation before implementation';
  } else {
    recommendation = '🔴 Task is not ready for implementation, research required';
  }

  return {
    readiness_score: score,
    readiness_factors: factors,
    missing_requirements: missingRequirements,
    recommendation
  };
}

/**
 * Get implementation history for a task
 */
export async function getImplementationHistory(
  postgresql: PostgreSQLResource,
  task_id: string
): Promise<Array<{
  id: string;
  action: string;
  agent_name: string;
  model_used: string;
  successful: boolean;
  tokens_used: number;
  cost_usd: number;
  execution_time_ms: number;
  error_message?: string;
  started_at: string;
  completed_at?: string;
}>> {
  const history = await postgresql.sql`
    SELECT 
      id, action, agent_name, model_used, successful,
      tokens_used, cost_usd, execution_time_ms, error_message,
      started_at, completed_at
    FROM agent_runs
    WHERE task_id = ${task_id} AND action = 'implement'
    ORDER BY started_at DESC
  `;

  return history.map(run => ({
    id: run.id,
    action: run.action,
    agent_name: run.agent_name,
    model_used: run.model_used,
    successful: run.successful,
    tokens_used: run.tokens_used || 0,
    cost_usd: parseFloat(run.cost_usd) || 0,
    execution_time_ms: run.execution_time_ms || 0,
    error_message: run.error_message,
    started_at: run.started_at,
    completed_at: run.completed_at
  }));
}

/**
 * Update app-level implementation statistics
 */
async function updateAppImplementationStats(
  postgresql: PostgreSQLResource,
  app_id: string,
  implementation_result: ImplementTaskResult,
  task: any
): Promise<void> {
  try {
    // Calculate implementation metrics
    const estimated_hours = task.estimated_hours || 0;
    const actual_duration = task.started_at && implementation_result.completed_at
      ? (new Date(implementation_result.completed_at).getTime() - new Date(task.started_at).getTime()) / (1000 * 60 * 60)
      : null;

    // Update app metadata with implementation stats
    await postgresql.sql`
      UPDATE apps 
      SET 
        metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
          'implementation_stats', jsonb_build_object(
            'total_implementations', COALESCE((metadata->'implementation_stats'->>'total_implementations')::int, 0) + 1,
            'successful_implementations', COALESCE((metadata->'implementation_stats'->>'successful_implementations')::int, 0) + ${implementation_result.implementation_completed ? 1 : 0},
            'total_tokens_used', COALESCE((metadata->'implementation_stats'->>'total_tokens_used')::int, 0) + ${implementation_result.tokens_used},
            'total_cost_usd', COALESCE((metadata->'implementation_stats'->>'total_cost_usd')::numeric, 0) + ${implementation_result.estimated_cost},
            'avg_files_per_implementation', COALESCE((metadata->'implementation_stats'->>'avg_files_per_implementation')::numeric, 0) * 0.9 + ${implementation_result.files_modified.length} * 0.1,
            'last_implementation', ${new Date().toISOString()}
          )
        ),
        updated_at = CURRENT_TIMESTAMP
      WHERE id = ${app_id}
    `;

    console.log('📊 Updated app implementation statistics');
  } catch (error) {
    console.warn('Failed to update implementation stats:', error);
  }
}

/**
 * Batch implement multiple staged tasks
 */
export async function batchImplementTasks(
  postgresql: PostgreSQLResource,
  task_ids: string[],
  max_concurrent: number = 2, // Lower concurrency for resource-intensive implementations
  implementation_notes?: string
): Promise<{
  total_requested: number;
  successful: number;
  failed: number;
  total_tokens: number;
  total_cost: number;
  results: ImplementTaskResult[];
}> {
  console.log(`🚀 Starting batch implementation for ${task_ids.length} tasks`);
  
  const results: ImplementTaskResult[] = [];
  let successful = 0;
  let failed = 0;
  let total_tokens = 0;
  let total_cost = 0;

  // Process tasks in smaller batches for implementations
  for (let i = 0; i < task_ids.length; i += max_concurrent) {
    const batch = task_ids.slice(i, i + max_concurrent);
    
    const batchPromises = batch.map(task_id =>
      main(postgresql, { task_id, implementation_notes }).then(result => {
        if (result.success && result.implementation_completed) {
          successful++;
        } else {
          failed++;
        }
        total_tokens += result.tokens_used;
        total_cost += result.estimated_cost;
        return result;
      })
    );

    const batchResults = await Promise.all(batchPromises);
    results.push(...batchResults);
    
    // Longer delay between batches for implementations
    if (i + max_concurrent < task_ids.length) {
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  }

  console.log(`✅ Batch implementation completed: ${successful} successful, ${failed} failed`);
  console.log(`🧠 Total tokens used: ${total_tokens}`);
  console.log(`💰 Total estimated cost: $${total_cost.toFixed(4)}`);

  return {
    total_requested: task_ids.length,
    successful,
    failed,
    total_tokens,
    total_cost,
    results
  };
}