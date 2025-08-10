import { Request, Response, NextFunction } from 'express';
import { v4 as uuidv4 } from 'uuid';
import { logger } from '../utils/logger.js';
import { getDbPool } from '../config/database.js';

// Extend Express Request to include logging properties
declare global {
  namespace Express {
    interface Request {
      requestId: string;
      startTime: number;
    }
  }
}

/**
 * Add request ID and timing to all requests
 */
export const requestLogger = (req: Request, res: Response, next: NextFunction): void => {
  req.requestId = uuidv4();
  req.startTime = Date.now();

  // Log request
  logger.info({
    type: 'request',
    requestId: req.requestId,
    method: req.method,
    path: req.path,
    query: req.query,
    ip: req.ip,
    userAgent: req.get('user-agent')
  });

  // Log response when finished
  res.on('finish', () => {
    const duration = Date.now() - req.startTime;
    
    logger.info({
      type: 'response',
      requestId: req.requestId,
      method: req.method,
      path: req.path,
      statusCode: res.statusCode,
      duration,
      ...(req.apiToken && { tokenId: req.apiToken.id })
    });

    // Log to database for analytics (async, don't wait)
    if (req.apiToken) {
      logApiUsage(req, res, duration).catch(err => {
        logger.error('Failed to log API usage:', err);
      });
    }
  });

  next();
};

/**
 * Log API usage to database for analytics
 */
async function logApiUsage(
  req: Request, 
  res: Response, 
  duration: number
): Promise<void> {
  const db = getDbPool();
  
  await db.query(
    `INSERT INTO api_usage_stats 
     (endpoint, method, response_code, execution_time_ms, token_id, request_id)
     VALUES ($1, $2, $3, $4, $5, $6)`,
    [
      req.path,
      req.method,
      res.statusCode,
      duration,
      req.apiToken?.id || null,
      req.requestId
    ]
  );
}