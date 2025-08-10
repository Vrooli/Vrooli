import { Router } from 'express';
import promptsRoutes from './prompts.routes.js';
import workflowsRoutes from './workflows.routes.js';
import analysisRoutes from './analysis.routes.js';
import templatesRoutes from './templates.routes.js';
import { healthController } from '../controllers/health.controller.js';

const router = Router();

// Health and info endpoints (no auth required)
router.get('/health', healthController.health);
router.get('/info', healthController.info);
router.get('/stats', healthController.stats);

// API routes (auth required)
router.use('/prompts', promptsRoutes);
router.use('/workflows', workflowsRoutes);
router.use('/analyze', analysisRoutes);
router.use('/templates', templatesRoutes);

export default router;