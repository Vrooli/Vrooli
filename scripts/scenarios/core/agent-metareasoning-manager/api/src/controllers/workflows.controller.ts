import { Request, Response } from 'express';
import { container } from 'tsyringe';
import { WorkflowsService } from '../services/workflows.service.js';
import { asyncHandler } from '../utils/asyncHandler.js';
import {
  ListWorkflowsQuery,
  ExecuteWorkflowRequest
} from '../schemas/workflows.schema.js';

class WorkflowsController {
  /**
   * List all workflows
   */
  list = asyncHandler(async (req: Request, res: Response) => {
    const workflowsService = container.resolve(WorkflowsService);
    
    const { type, limit = 20, offset = 0 } = req.query as any;
    const result = await workflowsService.getWorkflows(type, limit, offset);
    
    res.json({
      status: 'success',
      data: result,
      meta: {
        total: result.total,
        limit,
        offset
      }
    });
  });

  /**
   * Get a single workflow
   */
  getById = asyncHandler(async (req: Request, res: Response) => {
    const workflowsService = container.resolve(WorkflowsService);
    
    const workflow = await workflowsService.getWorkflowById(req.params.id);
    
    res.json({
      status: 'success',
      data: workflow
    });
  });

  /**
   * Execute a workflow
   */
  execute = asyncHandler(async (req: Request, res: Response) => {
    const workflowsService = container.resolve(WorkflowsService);
    
    const result = await workflowsService.executeWorkflow(req.params.id, req.body);
    
    res.json({
      status: 'success',
      data: result,
      message: 'Workflow execution started'
    });
  });

  /**
   * Get workflow execution status
   */
  getExecutionStatus = asyncHandler(async (req: Request, res: Response) => {
    const workflowsService = container.resolve(WorkflowsService);
    
    const { workflowId, executionId } = req.params;
    const execution = await workflowsService.getExecutionStatus(workflowId, executionId);
    
    res.json({
      status: 'success',
      data: execution
    });
  });

  /**
   * Get recent executions for a workflow
   */
  getRecentExecutions = asyncHandler(async (req: Request, res: Response) => {
    const workflowsService = container.resolve(WorkflowsService);
    
    const limit = parseInt(req.query.limit as string) || 10;
    const executions = await workflowsService.getRecentExecutions(req.params.id, limit);
    
    res.json({
      status: 'success',
      data: executions,
      meta: {
        count: executions.length
      }
    });
  });
}

export const workflowsController = new WorkflowsController();