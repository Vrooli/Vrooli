import { Router } from 'express';
import { workflowsController } from '../controllers/workflows.controller.js';
import { authenticate } from '../middleware/auth.middleware.js';
import { validate, validateBody } from '../middleware/validation.middleware.js';
import {
  listWorkflowsQuerySchema,
  executeWorkflowSchema,
  getExecutionStatusSchema
} from '../schemas/workflows.schema.js';

const router = Router();

// List workflows
router.get(
  '/',
  authenticate,
  validateBody(listWorkflowsQuerySchema),
  workflowsController.list
);

// Get single workflow
router.get(
  '/:id',
  authenticate,
  workflowsController.getById
);

// Get recent executions for a workflow
router.get(
  '/:id/executions',
  authenticate,
  workflowsController.getRecentExecutions
);

// Execute workflow
router.post(
  '/:id/execute',
  authenticate,
  validate(executeWorkflowSchema),
  workflowsController.execute
);

// Get execution status
router.get(
  '/:workflowId/executions/:executionId',
  authenticate,
  validate(getExecutionStatusSchema),
  workflowsController.getExecutionStatus
);

export default router;