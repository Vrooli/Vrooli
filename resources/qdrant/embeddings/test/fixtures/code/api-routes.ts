/**
 * Email API Routes - Express.js endpoints for email management
 * Implements RESTful API for email operations with authentication
 */

import express from 'express';
import { body, param, query, validationResult } from 'express-validator';
import { EmailService } from '../services/EmailService';
import { AuthMiddleware } from '../middleware/auth';
import { RateLimitMiddleware } from '../middleware/rateLimit';

const router = express.Router();
const emailService = new EmailService();

// Apply authentication to all routes
router.use(AuthMiddleware.requireAuth);

/**
 * GET /api/emails
 * Fetch user's emails with pagination and filtering
 * 
 * Query Parameters:
 *   - page: number (default: 1)
 *   - limit: number (default: 20, max: 100)
 *   - category: string (optional filter)
 *   - search: string (optional text search)
 * 
 * Returns: Paginated list of email objects
 */
router.get('/emails', 
  RateLimitMiddleware.standard,
  [
    query('page').optional().isInt({ min: 1 }).toInt(),
    query('limit').optional().isInt({ min: 1, max: 100 }).toInt(),
    query('category').optional().isIn(['important', 'normal', 'spam', 'promotional']),
    query('search').optional().isLength({ min: 1, max: 200 }).trim()
  ],
  async (req: express.Request, res: express.Response) => {
    try {
      const errors = validationResult(req);
      if (!errors.isEmpty()) {
        return res.status(400).json({
          error: 'Invalid parameters',
          details: errors.array()
        });
      }

      const userId = req.user?.id;
      if (!userId) {
        return res.status(401).json({ error: 'User not authenticated' });
      }

      const {
        page = 1,
        limit = 20,
        category,
        search
      } = req.query;

      const result = await emailService.getEmails({
        userId,
        page: Number(page),
        limit: Number(limit),
        category: category as string,
        search: search as string
      });

      res.json({
        success: true,
        data: result.emails,
        pagination: {
          currentPage: page,
          totalPages: result.totalPages,
          totalCount: result.totalCount,
          hasNext: result.hasNext,
          hasPrev: result.hasPrev
        }
      });

    } catch (error) {
      console.error('Error fetching emails:', error);
      res.status(500).json({
        error: 'Internal server error',
        message: 'Failed to fetch emails'
      });
    }
  }
);

/**
 * POST /api/emails/:id/categorize
 * Update email category using AI or manual override
 * 
 * Path Parameters:
 *   - id: string (email ID)
 * 
 * Body:
 *   - category: string (required) - new category
 *   - useAI: boolean (optional) - whether to use AI suggestions
 * 
 * Returns: Updated email object
 */
router.post('/emails/:id/categorize',
  RateLimitMiddleware.heavy,
  [
    param('id').isUUID().withMessage('Invalid email ID format'),
    body('category').isIn(['important', 'normal', 'spam', 'promotional'])
      .withMessage('Invalid category'),
    body('useAI').optional().isBoolean()
  ],
  async (req: express.Request, res: express.Response) => {
    try {
      const errors = validationResult(req);
      if (!errors.isEmpty()) {
        return res.status(400).json({
          error: 'Validation failed',
          details: errors.array()
        });
      }

      const { id } = req.params;
      const { category, useAI = false } = req.body;
      const userId = req.user?.id;

      // Verify email belongs to user
      const email = await emailService.getEmailById(id);
      if (!email || email.userId !== userId) {
        return res.status(404).json({
          error: 'Email not found',
          message: 'Email does not exist or access denied'
        });
      }

      let finalCategory = category;
      
      // Use AI suggestion if requested
      if (useAI) {
        try {
          const aiSuggestion = await emailService.getAICategory(email);
          finalCategory = aiSuggestion.category;
          
          // Log AI usage for analytics
          await emailService.logAIUsage(userId, {
            action: 'categorize',
            confidence: aiSuggestion.confidence,
            originalCategory: email.category,
            suggestedCategory: finalCategory
          });
        } catch (aiError) {
          console.warn('AI categorization failed, using manual category:', aiError);
          // Continue with manual category
        }
      }

      // Update email category
      const updatedEmail = await emailService.updateEmailCategory(id, {
        category: finalCategory,
        updatedBy: useAI ? 'ai' : 'manual',
        updatedAt: new Date()
      });

      res.json({
        success: true,
        data: updatedEmail,
        aiUsed: useAI
      });

    } catch (error) {
      console.error('Error categorizing email:', error);
      res.status(500).json({
        error: 'Internal server error',
        message: 'Failed to categorize email'
      });
    }
  }
);

/**
 * POST /api/emails/batch/process
 * Process multiple emails in batch (categorization, spam detection)
 * 
 * Body:
 *   - emailIds: string[] (required) - array of email IDs
 *   - operations: string[] (required) - operations to perform
 * 
 * Returns: Batch processing results
 */
router.post('/emails/batch/process',
  RateLimitMiddleware.strict,
  [
    body('emailIds').isArray({ min: 1, max: 50 })
      .withMessage('Must provide 1-50 email IDs'),
    body('emailIds.*').isUUID()
      .withMessage('All email IDs must be valid UUIDs'),
    body('operations').isArray({ min: 1 })
      .withMessage('Must specify at least one operation'),
    body('operations.*').isIn(['categorize', 'spam_check', 'priority_score'])
      .withMessage('Invalid operation specified')
  ],
  async (req: express.Request, res: express.Response) => {
    try {
      const errors = validationResult(req);
      if (!errors.isEmpty()) {
        return res.status(400).json({
          error: 'Validation failed',
          details: errors.array()
        });
      }

      const { emailIds, operations } = req.body;
      const userId = req.user?.id;

      // Verify all emails belong to user
      const emails = await emailService.getEmailsByIds(emailIds, userId);
      const foundIds = emails.map(e => e.id);
      const missingIds = emailIds.filter(id => !foundIds.includes(id));

      if (missingIds.length > 0) {
        return res.status(404).json({
          error: 'Some emails not found',
          missingIds
        });
      }

      // Process batch operations
      const batchResults = await emailService.processBatch({
        emails,
        operations,
        userId
      });

      res.json({
        success: true,
        processed: batchResults.processed,
        failed: batchResults.failed,
        results: batchResults.results,
        processingTime: batchResults.processingTime
      });

    } catch (error) {
      console.error('Error processing email batch:', error);
      res.status(500).json({
        error: 'Internal server error',
        message: 'Failed to process email batch'
      });
    }
  }
);

/**
 * GET /api/emails/analytics
 * Get email analytics and insights for user
 * 
 * Query Parameters:
 *   - period: string (day|week|month|year) - analytics period
 *   - timezone: string (optional) - user timezone
 * 
 * Returns: Analytics data object
 */
router.get('/emails/analytics',
  RateLimitMiddleware.standard,
  [
    query('period').isIn(['day', 'week', 'month', 'year'])
      .withMessage('Invalid period'),
    query('timezone').optional().isLength({ min: 1, max: 50 })
  ],
  async (req: express.Request, res: express.Response) => {
    try {
      const errors = validationResult(req);
      if (!errors.isEmpty()) {
        return res.status(400).json({
          error: 'Validation failed',
          details: errors.array()
        });
      }

      const userId = req.user?.id;
      const { period = 'week', timezone = 'UTC' } = req.query;

      const analytics = await emailService.getAnalytics({
        userId,
        period: period as string,
        timezone: timezone as string
      });

      res.json({
        success: true,
        data: analytics,
        generatedAt: new Date().toISOString()
      });

    } catch (error) {
      console.error('Error fetching analytics:', error);
      res.status(500).json({
        error: 'Internal server error',
        message: 'Failed to fetch analytics'
      });
    }
  }
);

/**
 * WebSocket endpoint for real-time email updates
 * Handles live notifications when new emails arrive
 */
export const setupEmailWebSocket = (io: any) => {
  io.on('connection', (socket: any) => {
    console.log('Email WebSocket client connected:', socket.id);
    
    socket.on('subscribe_emails', async (data: { userId: string, token: string }) => {
      try {
        // Verify authentication
        const user = await AuthMiddleware.verifyWebSocketToken(data.token);
        if (!user || user.id !== data.userId) {
          socket.emit('auth_error', { message: 'Invalid authentication' });
          return;
        }
        
        // Join user-specific room
        socket.join(`user_emails_${data.userId}`);
        socket.emit('subscribed', { message: 'Successfully subscribed to email updates' });
        
      } catch (error) {
        console.error('WebSocket auth error:', error);
        socket.emit('auth_error', { message: 'Authentication failed' });
      }
    });
    
    socket.on('disconnect', () => {
      console.log('Email WebSocket client disconnected:', socket.id);
    });
  });
};

export default router;