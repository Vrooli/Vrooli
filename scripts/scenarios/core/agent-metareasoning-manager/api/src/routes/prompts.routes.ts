import { Router } from 'express';
import { promptsController } from '../controllers/prompts.controller.js';
import { authenticate } from '../middleware/auth.middleware.js';
import { validate, validateBody, validateParams } from '../middleware/validation.middleware.js';
import {
  listPromptsQuerySchema,
  createPromptSchema,
  updatePromptSchema,
  getPromptSchema,
  deletePromptSchema,
  testPromptSchema
} from '../schemas/prompts.schema.js';
import { paginationSchema } from '../schemas/common.schema.js';

const router = Router();

// List prompts
router.get(
  '/',
  authenticate,
  validateBody(paginationSchema),
  promptsController.list
);

// Search prompts
router.get(
  '/search',
  authenticate,
  promptsController.search
);

// Get single prompt
router.get(
  '/:id',
  authenticate,
  validate(getPromptSchema),
  promptsController.getById
);

// Create prompt
router.post(
  '/',
  authenticate,
  validate(createPromptSchema),
  promptsController.create
);

// Update prompt
router.put(
  '/:id',
  authenticate,
  validate(updatePromptSchema),
  promptsController.update
);

// Delete prompt
router.delete(
  '/:id',
  authenticate,
  validate(deletePromptSchema),
  promptsController.delete
);

// Test prompt
router.post(
  '/:id/test',
  authenticate,
  validate(testPromptSchema),
  promptsController.test
);

export default router;