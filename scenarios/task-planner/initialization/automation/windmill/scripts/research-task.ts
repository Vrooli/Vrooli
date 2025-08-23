/**
 * Research a task and generate detailed implementation plan
 * Integrates with n8n research-workflow via webhook
 */

import { PostgreSQLResource } from 'windmill-client'

type ResearchTaskInput = {
  task_id: string;
  force_refresh?: boolean; // Re-research even if already researched
}

type ResearchTaskResult = {
  success: boolean;
  task_id: string;
  status: string;
  investigation_report?: string;
  implementation_plan?: string;
  technical_requirements?: string;
  estimated_hours?: number;
  confidence_score?: number;
  artifacts_collected?: number;
  error?: string;
  details?: string;
}

export async function main(
  postgresql: PostgreSQLResource,
  input: ResearchTaskInput
): Promise<ResearchTaskResult> {
  try {
    // Validate task exists and is in correct status
    const taskQuery = await postgresql.sql`
      SELECT t.*, a.name as app_name, a.display_name as app_display_name
      FROM tasks t
      JOIN apps a ON t.app_id = a.id
      WHERE t.id = ${input.task_id}
    `;

    if (taskQuery.length === 0) {
      return {
        success: false,
        task_id: input.task_id,
        status: 'not_found',
        error: 'Task not found'
      };
    }

    const task = taskQuery[0];

    // Check if task is in backlog status (required for research)
    if (task.status !== 'backlog' && !input.force_refresh) {
      return {
        success: false,
        task_id: input.task_id,
        status: task.status,
        error: `Task is not in backlog status (current: ${task.status}). Use force_refresh=true to re-research.`
      };
    }

    // If forcing refresh, temporarily reset to backlog
    if (input.force_refresh && task.status !== 'backlog') {
      await postgresql.sql`
        UPDATE tasks 
        SET status = 'backlog', investigation_report = NULL, implementation_plan = NULL, technical_requirements = NULL
        WHERE id = ${input.task_id}
      `;
      
      // Record the status change
      await postgresql.sql`
        INSERT INTO task_transitions (task_id, from_status, to_status, triggered_by, trigger_type, reason)
        VALUES (${input.task_id}, ${task.status}, 'backlog', 'windmill-research', 'manual', 'Forced refresh for re-research')
      `;
    }

    console.log(`🔬 Starting research for task: "${task.title}"`);
    console.log(`📋 App: ${task.app_display_name} (${task.app_name})`);

    // Call n8n research workflow
    const n8nPayload = {
      task_id: input.task_id
    };

    const n8nResponse = await fetch('http://localhost:5679/webhook/research-task', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(n8nPayload)
    });

    if (!n8nResponse.ok) {
      const errorText = await n8nResponse.text();
      throw new Error(`Research workflow failed: ${n8nResponse.status} ${errorText}`);
    }

    const result = await n8nResponse.json() as ResearchTaskResult;

    // Log research results
    if (result.success) {
      console.log(`✅ Task research completed successfully`);
      console.log(`📊 Confidence Score: ${Math.round((result.confidence_score || 0) * 100)}%`);
      console.log(`⏱️ Estimated Hours: ${result.estimated_hours || 'Not estimated'}`);
      console.log(`📚 Research Artifacts: ${result.artifacts_collected || 0}`);
      
      // Update task statistics
      await updateTaskResearchStats(postgresql, task.app_id, result);
    } else {
      console.log(`❌ Task research failed: ${result.error}`);
    }

    return result;

  } catch (error) {
    console.error('❌ Error in research-task:', error);
    return {
      success: false,
      task_id: input.task_id,
      status: 'error',
      error: error instanceof Error ? error.message : 'Unknown error occurred'
    };
  }
}

/**
 * Get similar tasks to help with research context
 */
export async function getSimilarTasks(
  postgresql: PostgreSQLResource,
  task_id: string,
  limit: number = 5
): Promise<Array<{
  id: string;
  title: string;
  description: string;
  status: string;
  similarity_score: number;
  implementation_plan?: string;
  technical_requirements?: string;
}>> {
  try {
    // Get similar tasks using vector similarity
    const similarTasks = await postgresql.sql`
      SELECT 
        t2.id, t2.title, t2.description, t2.status,
        t2.implementation_plan, t2.technical_requirements,
        1 - (t1.title_embedding <-> t2.title_embedding) as similarity_score
      FROM tasks t1, tasks t2
      WHERE t1.id = ${task_id} 
        AND t2.id != ${task_id}
        AND t1.app_id = t2.app_id
        AND t2.title_embedding IS NOT NULL
        AND t1.title_embedding <-> t2.title_embedding < 0.3
      ORDER BY t1.title_embedding <-> t2.title_embedding
      LIMIT ${limit}
    `;

    return similarTasks.map(task => ({
      id: task.id,
      title: task.title,
      description: task.description,
      status: task.status,
      similarity_score: parseFloat(task.similarity_score),
      implementation_plan: task.implementation_plan,
      technical_requirements: task.technical_requirements
    }));

  } catch (error) {
    console.warn('Could not retrieve similar tasks:', error);
    return [];
  }
}

/**
 * Get research artifacts for a task
 */
export async function getResearchArtifacts(
  postgresql: PostgreSQLResource,
  task_id: string
): Promise<Array<{
  id: string;
  type: string;
  title: string;
  source_url: string;
  content: string;
  relevance_score: number;
  created_at: string;
}>> {
  const artifacts = await postgresql.sql`
    SELECT id, type, title, source_url, content, relevance_score, created_at
    FROM research_artifacts
    WHERE task_id = ${task_id}
    ORDER BY relevance_score DESC, created_at DESC
  `;

  return artifacts.map(artifact => ({
    id: artifact.id,
    type: artifact.type,
    title: artifact.title,
    source_url: artifact.source_url,
    content: artifact.content,
    relevance_score: parseFloat(artifact.relevance_score),
    created_at: artifact.created_at
  }));
}

/**
 * Update app-level research statistics
 */
async function updateTaskResearchStats(
  postgresql: PostgreSQLResource,
  app_id: string,
  research_result: ResearchTaskResult
): Promise<void> {
  try {
    // Calculate new average completion time and confidence
    const stats = await postgresql.sql`
      SELECT 
        COUNT(*) FILTER (WHERE status IN ('staged', 'completed')) as researched_tasks,
        AVG(estimated_hours) FILTER (WHERE estimated_hours IS NOT NULL) as avg_estimated_hours,
        AVG(confidence_score) FILTER (WHERE confidence_score IS NOT NULL) as avg_confidence
      FROM tasks
      WHERE app_id = ${app_id}
    `;

    if (stats.length > 0) {
      const { researched_tasks, avg_estimated_hours, avg_confidence } = stats[0];
      
      await postgresql.sql`
        UPDATE apps 
        SET 
          metadata = COALESCE(metadata, '{}'::jsonb) || jsonb_build_object(
            'research_stats', jsonb_build_object(
              'researched_tasks', ${researched_tasks},
              'avg_estimated_hours', ${avg_estimated_hours || 0},
              'avg_confidence', ${avg_confidence || 0},
              'last_research_update', ${new Date().toISOString()}
            )
          )
        WHERE id = ${app_id}
      `;
    }
  } catch (error) {
    console.warn('Failed to update research stats:', error);
  }
}

/**
 * Batch research multiple tasks
 */
export async function batchResearchTasks(
  postgresql: PostgreSQLResource,
  task_ids: string[],
  max_concurrent: number = 3
): Promise<{
  total_requested: number;
  successful: number;
  failed: number;
  results: ResearchTaskResult[];
}> {
  console.log(`🔬 Starting batch research for ${task_ids.length} tasks`);
  
  const results: ResearchTaskResult[] = [];
  let successful = 0;
  let failed = 0;

  // Process tasks in batches to avoid overwhelming the system
  for (let i = 0; i < task_ids.length; i += max_concurrent) {
    const batch = task_ids.slice(i, i + max_concurrent);
    
    const batchPromises = batch.map(task_id =>
      main(postgresql, { task_id }).then(result => {
        if (result.success) {
          successful++;
        } else {
          failed++;
        }
        return result;
      })
    );

    const batchResults = await Promise.all(batchPromises);
    results.push(...batchResults);
    
    // Small delay between batches
    if (i + max_concurrent < task_ids.length) {
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  }

  console.log(`✅ Batch research completed: ${successful} successful, ${failed} failed`);

  return {
    total_requested: task_ids.length,
    successful,
    failed,
    results
  };
}