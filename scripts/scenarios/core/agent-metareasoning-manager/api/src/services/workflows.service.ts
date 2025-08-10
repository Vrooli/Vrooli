import { injectable, inject } from 'tsyringe';
import { v4 as uuidv4 } from 'uuid';
import axios from 'axios';
import { WorkflowsRepository, WorkflowEntity } from '../repositories/workflows.repository.js';
import { config } from '../config/index.js';
import { ExternalServiceError, NotFoundError } from '../utils/errors.js';
import { logger } from '../utils/logger.js';
import { ExecuteWorkflowRequest, WorkflowExecution } from '../schemas/workflows.schema.js';

@injectable()
export class WorkflowsService {
  constructor(
    @inject(WorkflowsRepository) private workflowsRepo: WorkflowsRepository
  ) {}

  /**
   * Get all workflows with optional filtering
   */
  async getWorkflows(
    type?: 'n8n' | 'windmill',
    limit: number = 20,
    offset: number = 0
  ): Promise<{ workflows: WorkflowEntity[]; total: number }> {
    const filters = type ? { type, status: 'active' } : { status: 'active' };
    const result = await this.workflowsRepo.findPaginated(limit, offset, filters);
    
    return {
      workflows: result.data,
      total: result.total
    };
  }

  /**
   * Get a single workflow by ID
   */
  async getWorkflowById(id: string): Promise<WorkflowEntity> {
    return await this.workflowsRepo.findByIdOrThrow(id);
  }

  /**
   * Find a workflow by name
   */
  async findWorkflowByName(name: string): Promise<WorkflowEntity | null> {
    try {
      const workflows = await this.workflowsRepo.findByFilters({ name, status: 'active' });
      return workflows.length > 0 ? workflows[0] : null;
    } catch (error) {
      logger.error(`Error finding workflow by name: ${name}`, error);
      return null;
    }
  }

  /**
   * Execute a workflow
   */
  async executeWorkflow(
    id: string,
    data: ExecuteWorkflowRequest['body']
  ): Promise<WorkflowExecution> {
    const workflow = await this.getWorkflowById(id);
    const executionId = uuidv4();

    // Log execution start
    await this.workflowsRepo.logExecution(
      id,
      executionId,
      'running',
      data.input
    );

    try {
      let result: any;

      if (workflow.type === 'n8n') {
        result = await this.executeN8nWorkflow(workflow, data.input, data.timeout);
      } else if (workflow.type === 'windmill') {
        result = await this.executeWindmillWorkflow(workflow, data.input, data.timeout);
      } else {
        throw new Error(`Unknown workflow type: ${workflow.type}`);
      }

      // Update execution success
      await this.workflowsRepo.updateExecutionStatus(
        executionId,
        'completed',
        result
      );

      // Update workflow stats
      await this.workflowsRepo.updateExecutionStats(id);

      logger.info(`Workflow ${id} executed successfully`, { executionId });

      return {
        execution_id: executionId,
        workflow_id: id,
        status: 'completed',
        input: data.input,
        output: result,
        error: null,
        started_at: new Date(),
        completed_at: new Date(),
        duration_ms: 0 // Will be calculated from DB
      };
    } catch (error: any) {
      // Update execution failure
      await this.workflowsRepo.updateExecutionStatus(
        executionId,
        'failed',
        null,
        error.message
      );

      logger.error(`Workflow ${id} execution failed`, { executionId, error });

      throw new ExternalServiceError(workflow.type, error);
    }
  }

  /**
   * Execute n8n workflow
   */
  private async executeN8nWorkflow(
    workflow: WorkflowEntity,
    input: any,
    timeout: number
  ): Promise<any> {
    const webhookUrl = `${config.n8n.webhookBase}/${workflow.workflow_id}`;
    
    const response = await axios.post(webhookUrl, input, {
      timeout,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    return response.data;
  }

  /**
   * Execute Windmill workflow
   */
  private async executeWindmillWorkflow(
    workflow: WorkflowEntity,
    input: any,
    timeout: number
  ): Promise<any> {
    const apiUrl = `${config.windmill.baseUrl}/api/w/${config.windmill.workspace}/jobs/run/f/${workflow.workflow_id}`;
    
    // Start job
    const startResponse = await axios.post(apiUrl, input, {
      headers: {
        'Content-Type': 'application/json'
      }
    });

    const jobId = startResponse.data.id;

    // Poll for completion
    const pollInterval = 1000; // 1 second
    const maxPolls = Math.floor(timeout / pollInterval);
    
    for (let i = 0; i < maxPolls; i++) {
      await new Promise(resolve => setTimeout(resolve, pollInterval));
      
      const statusResponse = await axios.get(
        `${config.windmill.baseUrl}/api/w/${config.windmill.workspace}/jobs/${jobId}`
      );

      if (statusResponse.data.type === 'CompletedJob') {
        return statusResponse.data.result;
      } else if (statusResponse.data.type === 'FailedJob') {
        throw new Error(statusResponse.data.error || 'Workflow execution failed');
      }
    }

    throw new Error('Workflow execution timeout');
  }

  /**
   * Get workflow execution status
   */
  async getExecutionStatus(workflowId: string, executionId: string): Promise<any> {
    const execution = await this.workflowsRepo.getExecution(executionId);
    
    if (!execution) {
      throw new NotFoundError('Workflow execution', executionId);
    }

    if (execution.workflow_id !== workflowId) {
      throw new NotFoundError('Workflow execution', executionId);
    }

    return execution;
  }

  /**
   * Get recent executions for a workflow
   */
  async getRecentExecutions(workflowId: string, limit: number = 10): Promise<any[]> {
    return await this.workflowsRepo.getRecentExecutions(workflowId, limit);
  }
}