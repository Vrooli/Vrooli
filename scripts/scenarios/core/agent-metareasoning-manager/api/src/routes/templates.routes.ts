import { Router } from 'express';
import { templatesController } from '../controllers/templates.controller.js';
import { authenticate } from '../middleware/auth.middleware.js';
import { validate, validateBody } from '../middleware/validation.middleware.js';
import {
  listTemplatesQuerySchema,
  createTemplateSchema,
  applyTemplateSchema
} from '../schemas/templates.schema.js';

const router = Router();

// List templates
router.get(
  '/',
  authenticate,
  validateBody(listTemplatesQuerySchema),
  templatesController.list
);

// Get most used templates
router.get(
  '/most-used',
  authenticate,
  templatesController.getMostUsed
);

// Get single template
router.get(
  '/:id',
  authenticate,
  templatesController.getById
);

// Create template
router.post(
  '/',
  authenticate,
  validate(createTemplateSchema),
  templatesController.create
);

// Apply template
router.post(
  '/:id/apply',
  authenticate,
  validate(applyTemplateSchema),
  templatesController.apply
);

export default router;