import { Router } from 'express';
import { analysisController } from '../controllers/analysis.controller.js';
import { authenticate } from '../middleware/auth.middleware.js';
import { validate } from '../middleware/validation.middleware.js';
import {
  analyzeDecisionSchema,
  analyzeProsConsSchema,
  analyzeSwotSchema,
  analyzeRisksSchema
} from '../schemas/analysis.schema.js';

const router = Router();

// Decision analysis
router.post(
  '/decision',
  authenticate,
  validate(analyzeDecisionSchema),
  analysisController.analyzeDecision
);

// Pros and cons analysis
router.post(
  '/pros-cons',
  authenticate,
  validate(analyzeProsConsSchema),
  analysisController.analyzeProscons
);

// SWOT analysis
router.post(
  '/swot',
  authenticate,
  validate(analyzeSwotSchema),
  analysisController.analyzeSwot
);

// Risk analysis
router.post(
  '/risks',
  authenticate,
  validate(analyzeRisksSchema),
  analysisController.analyzeRisks
);

// Analysis chain
router.post(
  '/chain',
  authenticate,
  analysisController.executeChain
);

export default router;