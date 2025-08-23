/**
 * API handlers for Task Planner Dashboard
 * Contains all the query functions referenced in the Windmill UI
 */

import { PostgreSQLResource } from 'windmill-client'

// Types
type App = {
  id: string;
  name: string;
  display_name: string;
  type: string;
  total_tasks: number;
  completed_tasks: number;
  created_at: string;
}

type Task = {
  id: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  tags: string[];
  estimated_hours: number | null;
  confidence_score: number | null;
  created_at: string;
  updated_at: string;
  app_name: string;
}

/**
 * Get all apps for the project selector
 */
export async function get_apps(postgresql: PostgreSQLResource): Promise<App[]> {
  const apps = await postgresql.sql`
    SELECT id, name, display_name, type, total_tasks, completed_tasks, created_at
    FROM apps
    ORDER BY display_name
  `;
  
  return apps.map(app => ({
    id: app.id,
    name: app.name,
    display_name: app.display_name,
    type: app.type,
    total_tasks: app.total_tasks || 0,
    completed_tasks: app.completed_tasks || 0,
    created_at: app.created_at
  }));
}

/**
 * Get backlog tasks for selected app
 */
export async function get_backlog_tasks(
  postgresql: PostgreSQLResource,
  app_id?: string
): Promise<Task[]> {
  const whereClause = app_id ? "WHERE t.app_id = $1 AND t.status = 'backlog'" : "WHERE t.status = 'backlog'";
  const params = app_id ? [app_id] : [];
  
  const tasks = await postgresql.sql`
    SELECT 
      t.id, t.title, t.description, t.status, t.priority, t.tags,
      t.estimated_hours, t.confidence_score, t.created_at, t.updated_at,
      a.display_name as app_name
    FROM tasks t
    JOIN apps a ON t.app_id = a.id
    ${app_id ? 
      postgresql.sql`WHERE t.app_id = ${app_id} AND t.status = 'backlog'` : 
      postgresql.sql`WHERE t.status = 'backlog'`
    }
    ORDER BY 
      CASE t.priority 
        WHEN 'critical' THEN 1
        WHEN 'high' THEN 2 
        WHEN 'medium' THEN 3
        WHEN 'low' THEN 4
      END,
      t.created_at DESC
  `;
  
  return tasks.map(formatTaskResponse);
}

/**
 * Get staged tasks ready for implementation
 */
export async function get_staged_tasks(
  postgresql: PostgreSQLResource,
  app_id?: string
): Promise<Task[]> {
  const tasks = await postgresql.sql`
    SELECT 
      t.id, t.title, t.description, t.status, t.priority, t.tags,
      t.estimated_hours, t.confidence_score, t.created_at, t.updated_at,
      a.display_name as app_name,
      CASE WHEN t.investigation_report IS NOT NULL THEN '✅' ELSE '❌' END as has_investigation,
      CASE WHEN t.implementation_plan IS NOT NULL THEN '✅' ELSE '❌' END as has_plan
    FROM tasks t
    JOIN apps a ON t.app_id = a.id
    ${app_id ? 
      postgresql.sql`WHERE t.app_id = ${app_id} AND t.status = 'staged'` : 
      postgresql.sql`WHERE t.status = 'staged'`
    }
    ORDER BY t.confidence_score DESC, t.priority, t.created_at DESC
  `;
  
  return tasks.map(task => ({
    ...formatTaskResponse(task),
    has_investigation: task.has_investigation,
    has_plan: task.has_plan
  }));
}

/**
 * Get tasks in progress
 */
export async function get_progress_tasks(
  postgresql: PostgreSQLResource,
  app_id?: string
): Promise<(Task & { hours_in_progress: number })[]> {
  const tasks = await postgresql.sql`
    SELECT 
      t.id, t.title, t.description, t.status, t.priority, t.tags,
      t.estimated_hours, t.confidence_score, t.created_at, t.updated_at,
      a.display_name as app_name, t.started_at,
      EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - t.started_at)) / 3600 as hours_in_progress
    FROM tasks t
    JOIN apps a ON t.app_id = a.id
    ${app_id ? 
      postgresql.sql`WHERE t.app_id = ${app_id} AND t.status = 'in_progress'` : 
      postgresql.sql`WHERE t.status = 'in_progress'`
    }
    ORDER BY t.started_at DESC
  `;
  
  return tasks.map(task => ({
    ...formatTaskResponse(task),
    hours_in_progress: parseFloat(task.hours_in_progress) || 0
  }));
}

/**
 * Get completed tasks
 */
export async function get_completed_tasks(
  postgresql: PostgreSQLResource,
  app_id?: string
): Promise<(Task & { completion_date: string; has_implementation: boolean })[]> {
  const tasks = await postgresql.sql`
    SELECT 
      t.id, t.title, t.description, t.status, t.priority, t.tags,
      t.estimated_hours, t.confidence_score, t.created_at, t.updated_at,
      a.display_name as app_name, t.completed_at,
      CASE WHEN t.implementation_code IS NOT NULL THEN true ELSE false END as has_implementation
    FROM tasks t
    JOIN apps a ON t.app_id = a.id
    ${app_id ? 
      postgresql.sql`WHERE t.app_id = ${app_id} AND t.status = 'completed'` : 
      postgresql.sql`WHERE t.status = 'completed'`
    }
    ORDER BY t.completed_at DESC
    LIMIT 50
  `;
  
  return tasks.map(task => ({
    ...formatTaskResponse(task),
    completion_date: task.completed_at,
    has_implementation: task.has_implementation
  }));
}

/**
 * Get task statistics for analytics
 */
export async function get_task_stats(
  postgresql: PostgreSQLResource,
  app_id?: string
): Promise<{
  total: number;
  backlog: number;
  staged: number;
  in_progress: number;
  completed: number;
  failed: number;
  avg_hours: number;
  avg_confidence: number;
}> {
  const stats = await postgresql.sql`
    SELECT 
      COUNT(*) as total,
      COUNT(*) FILTER (WHERE status = 'backlog') as backlog,
      COUNT(*) FILTER (WHERE status = 'staged') as staged,
      COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress,
      COUNT(*) FILTER (WHERE status = 'completed') as completed,
      COUNT(*) FILTER (WHERE status = 'failed') as failed,
      AVG(estimated_hours) FILTER (WHERE estimated_hours IS NOT NULL) as avg_hours,
      AVG(confidence_score) FILTER (WHERE confidence_score IS NOT NULL) as avg_confidence
    FROM tasks t
    ${app_id ? postgresql.sql`WHERE t.app_id = ${app_id}` : postgresql.sql``}
  `;

  const result = stats[0];
  return {
    total: parseInt(result.total),
    backlog: parseInt(result.backlog),
    staged: parseInt(result.staged),
    in_progress: parseInt(result.in_progress),
    completed: parseInt(result.completed),
    failed: parseInt(result.failed),
    avg_hours: parseFloat(result.avg_hours) || 0,
    avg_confidence: parseFloat(result.avg_confidence) || 0
  };
}

/**
 * Get recent activity across the system
 */
export async function get_recent_activity(
  postgresql: PostgreSQLResource,
  app_id?: string,
  limit: number = 20
): Promise<Array<{
  id: string;
  task_title: string;
  app_name: string;
  from_status: string;
  to_status: string;
  triggered_by: string;
  reason: string;
  created_at: string;
}>> {
  const activity = await postgresql.sql`
    SELECT 
      tt.id, t.title as task_title, a.display_name as app_name,
      tt.from_status, tt.to_status, tt.triggered_by, tt.reason, tt.created_at
    FROM task_transitions tt
    JOIN tasks t ON tt.task_id = t.id
    JOIN apps a ON t.app_id = a.id
    ${app_id ? postgresql.sql`WHERE t.app_id = ${app_id}` : postgresql.sql``}
    ORDER BY tt.created_at DESC
    LIMIT ${limit}
  `;

  return activity.map(item => ({
    id: item.id,
    task_title: item.task_title,
    app_name: item.app_name,
    from_status: item.from_status || 'none',
    to_status: item.to_status,
    triggered_by: item.triggered_by,
    reason: item.reason || 'No reason provided',
    created_at: item.created_at
  }));
}

/**
 * Research selected task (called from UI action)
 */
export async function research_selected_task(
  postgresql: PostgreSQLResource,
  task_id: string
): Promise<{ success: boolean; message: string }> {
  try {
    // Import and call the research function
    const { main: researchTask } = await import('./research-task.ts');
    const result = await researchTask(postgresql, { task_id });
    
    return {
      success: result.success,
      message: result.success 
        ? `✅ Task researched successfully. Confidence: ${Math.round((result.confidence_score || 0) * 100)}%`
        : `❌ Research failed: ${result.error}`
    };
  } catch (error) {
    return {
      success: false,
      message: `❌ Error: ${error instanceof Error ? error.message : 'Unknown error'}`
    };
  }
}

/**
 * Implement selected task (called from UI action)
 */
export async function implement_selected_task(
  postgresql: PostgreSQLResource,
  task_id: string
): Promise<{ success: boolean; message: string }> {
  try {
    const { main: implementTask } = await import('./implement-task.ts');
    const result = await implementTask(postgresql, { task_id });
    
    return {
      success: result.success && result.implementation_completed,
      message: result.success && result.implementation_completed
        ? `✅ Task implemented successfully. Files modified: ${result.files_modified.length}`
        : `❌ Implementation failed: ${result.error}`
    };
  } catch (error) {
    return {
      success: false,
      message: `❌ Error: ${error instanceof Error ? error.message : 'Unknown error'}`
    };
  }
}

/**
 * Parse text input (called from UI)
 */
export async function parse_text_input(
  postgresql: PostgreSQLResource,
  app_id: string,
  raw_text: string
): Promise<{ success: boolean; tasks_created: number; message: string }> {
  try {
    const { main: parseText } = await import('./parse-unstructured.ts');
    const result = await parseText(postgresql, { app_id, raw_text, submitted_by: 'windmill-ui' });
    
    return {
      success: result.success,
      tasks_created: result.tasks_created,
      message: result.success 
        ? `✅ Created ${result.tasks_created} tasks from text`
        : `❌ Parsing failed: ${result.error}`
    };
  } catch (error) {
    return {
      success: false,
      tasks_created: 0,
      message: `❌ Error: ${error instanceof Error ? error.message : 'Unknown error'}`
    };
  }
}

/**
 * View task details (for modals/detail views)
 */
export async function view_task_details(
  postgresql: PostgreSQLResource,
  task_id: string
): Promise<{
  task: Task & {
    investigation_report?: string;
    implementation_plan?: string;
    technical_requirements?: string;
    implementation_code?: string;
    implementation_result?: string;
  };
  similar_tasks: Array<{ id: string; title: string; similarity_score: number }>;
  research_artifacts: Array<{ title: string; source_url: string; relevance_score: number }>;
}> {
  // Get task details
  const taskQuery = await postgresql.sql`
    SELECT 
      t.*, a.display_name as app_name
    FROM tasks t
    JOIN apps a ON t.app_id = a.id
    WHERE t.id = ${task_id}
  `;

  if (taskQuery.length === 0) {
    throw new Error('Task not found');
  }

  const task = {
    ...formatTaskResponse(taskQuery[0]),
    investigation_report: taskQuery[0].investigation_report,
    implementation_plan: taskQuery[0].implementation_plan,
    technical_requirements: taskQuery[0].technical_requirements,
    implementation_code: taskQuery[0].implementation_code,
    implementation_result: taskQuery[0].implementation_result
  };

  // Get similar tasks
  const similarTasks = await postgresql.sql`
    SELECT t2.id, t2.title, 
           1 - (t1.title_embedding <-> t2.title_embedding) as similarity_score
    FROM tasks t1, tasks t2
    WHERE t1.id = ${task_id} 
      AND t2.id != ${task_id}
      AND t1.app_id = t2.app_id
      AND t2.title_embedding IS NOT NULL
      AND t1.title_embedding <-> t2.title_embedding < 0.3
    ORDER BY similarity_score DESC
    LIMIT 5
  `;

  // Get research artifacts
  const artifacts = await postgresql.sql`
    SELECT title, source_url, relevance_score
    FROM research_artifacts
    WHERE task_id = ${task_id}
    ORDER BY relevance_score DESC
    LIMIT 10
  `;

  return {
    task,
    similar_tasks: similarTasks.map(t => ({
      id: t.id,
      title: t.title,
      similarity_score: parseFloat(t.similarity_score)
    })),
    research_artifacts: artifacts.map(a => ({
      title: a.title,
      source_url: a.source_url,
      relevance_score: parseFloat(a.relevance_score)
    }))
  };
}

/**
 * Cancel a task in progress
 */
export async function cancel_task(
  postgresql: PostgreSQLResource,
  task_id: string,
  reason: string = 'Cancelled by user'
): Promise<{ success: boolean; message: string }> {
  try {
    await postgresql.sql`
      UPDATE tasks 
      SET status = 'cancelled', implementation_result = ${reason}, completed_at = CURRENT_TIMESTAMP
      WHERE id = ${task_id} AND status = 'in_progress'
    `;

    await postgresql.sql`
      INSERT INTO task_transitions (task_id, from_status, to_status, triggered_by, trigger_type, reason)
      VALUES (${task_id}, 'in_progress', 'cancelled', 'windmill-ui', 'manual', ${reason})
    `;

    return {
      success: true,
      message: '✅ Task cancelled successfully'
    };
  } catch (error) {
    return {
      success: false,
      message: `❌ Failed to cancel task: ${error instanceof Error ? error.message : 'Unknown error'}`
    };
  }
}

/**
 * Helper function to format task responses consistently
 */
function formatTaskResponse(task: any): Task {
  return {
    id: task.id,
    title: task.title,
    description: task.description || '',
    status: task.status,
    priority: task.priority,
    tags: task.tags || [],
    estimated_hours: task.estimated_hours ? parseFloat(task.estimated_hours) : null,
    confidence_score: task.confidence_score ? parseFloat(task.confidence_score) : null,
    created_at: task.created_at,
    updated_at: task.updated_at,
    app_name: task.app_name
  };
}