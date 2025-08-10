import { injectable } from 'tsyringe';
import { BaseRepository } from './base.repository.js';
import { Workflow } from '../schemas/workflows.schema.js';

export interface WorkflowEntity extends Workflow {
  id: string;
  created_at: Date;
  updated_at: Date;
}

@injectable()
export class WorkflowsRepository extends BaseRepository<WorkflowEntity> {
  protected tableName = 'workflows';
  protected columns = [
    'id',
    'name',
    'description',
    'type',
    'workflow_id',
    'input_schema',
    'output_schema',
    'tags',
    'status',
    'execution_count',
    'last_executed',
    'created_at',
    'updated_at'
  ];

  /**
   * Find workflows by filters
   */
  async findByFilters(filters: Record<string, any>): Promise<WorkflowEntity[]> {
    return await this.findAll(filters);
  }

  /**
   * Find workflows by type (n8n or windmill)
   */
  async findByType(type: 'n8n' | 'windmill'): Promise<WorkflowEntity[]> {
    const query = `
      SELECT ${this.columns.join(', ')}
      FROM ${this.tableName}
      WHERE type = $1 AND status = 'active'
      ORDER BY name ASC
    `;
    
    const result = await this.db.query(query, [type]);
    return result.rows;
  }

  /**
   * Update execution statistics
   */
  async updateExecutionStats(id: string): Promise<WorkflowEntity> {
    const query = `
      UPDATE ${this.tableName}
      SET 
        execution_count = execution_count + 1,
        last_executed = NOW(),
        updated_at = NOW()
      WHERE id = $1
      RETURNING ${this.columns.join(', ')}
    `;
    
    const result = await this.db.query(query, [id]);
    return result.rows[0];
  }

  /**
   * Log workflow execution
   */
  async logExecution(
    workflowId: string,
    executionId: string,
    status: string,
    input: any,
    output: any = null,
    error: string | null = null
  ): Promise<void> {
    const query = `
      INSERT INTO workflow_executions 
      (execution_id, workflow_id, status, input, output, error, started_at)
      VALUES ($1, $2, $3, $4, $5, $6, NOW())
    `;
    
    await this.db.query(query, [
      executionId,
      workflowId,
      status,
      JSON.stringify(input),
      output ? JSON.stringify(output) : null,
      error
    ]);
  }

  /**
   * Update execution status
   */
  async updateExecutionStatus(
    executionId: string,
    status: string,
    output?: any,
    error?: string
  ): Promise<void> {
    const query = `
      UPDATE workflow_executions
      SET 
        status = $2,
        output = $3,
        error = $4,
        completed_at = NOW(),
        duration_ms = EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000
      WHERE execution_id = $1
    `;
    
    await this.db.query(query, [
      executionId,
      status,
      output ? JSON.stringify(output) : null,
      error || null
    ]);
  }

  /**
   * Get execution by ID
   */
  async getExecution(executionId: string): Promise<any> {
    const query = `
      SELECT * FROM workflow_executions
      WHERE execution_id = $1
    `;
    
    const result = await this.db.query(query, [executionId]);
    return result.rows[0];
  }

  /**
   * Get recent executions for a workflow
   */
  async getRecentExecutions(workflowId: string, limit: number = 10): Promise<any[]> {
    const query = `
      SELECT * FROM workflow_executions
      WHERE workflow_id = $1
      ORDER BY started_at DESC
      LIMIT $2
    `;
    
    const result = await this.db.query(query, [workflowId, limit]);
    return result.rows;
  }
}