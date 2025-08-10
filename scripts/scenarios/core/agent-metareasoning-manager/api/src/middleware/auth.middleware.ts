import { Request, Response, NextFunction } from 'express';
import crypto from 'crypto';
import { getDbPool } from '../config/database.js';
import { AuthenticationError, AuthorizationError } from '../utils/errors.js';
import { logger } from '../utils/logger.js';
import { config } from '../config/index.js';

// Extend Express Request to include auth data
declare global {
  namespace Express {
    interface Request {
      apiToken?: {
        id: string;
        name: string;
        permissions: Record<string, boolean>;
      };
    }
  }
}

/**
 * Hash an API key for secure storage
 */
export const hashApiKey = (apiKey: string): string => {
  return crypto
    .createHash('sha256')
    .update(apiKey + config.security.apiKeySalt)
    .digest('hex');
};

/**
 * Authenticate API requests using Bearer token
 */
export const authenticate = async (
  req: Request,
  res: Response,
  next: NextFunction
): Promise<void> => {
  try {
    const authHeader = req.headers.authorization;
    
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      throw new AuthenticationError('Missing or invalid authorization header');
    }

    const apiKey = authHeader.substring(7);
    const tokenHash = hashApiKey(apiKey);

    // Look up token in database
    const db = getDbPool();
    const result = await db.query(
      `SELECT id, name, permissions, expires_at 
       FROM api_tokens 
       WHERE token_hash = $1 AND (expires_at IS NULL OR expires_at > NOW())`,
      [tokenHash]
    );

    if (result.rows.length === 0) {
      throw new AuthenticationError('Invalid or expired API token');
    }

    const token = result.rows[0];
    
    // Attach token info to request
    req.apiToken = {
      id: token.id,
      name: token.name,
      permissions: token.permissions || {}
    };

    // Log successful authentication
    logger.debug({
      message: 'API request authenticated',
      tokenId: token.id,
      path: req.path,
      method: req.method
    });

    next();
  } catch (error) {
    next(error);
  }
};

/**
 * Check if the authenticated token has a specific permission
 */
export const requirePermission = (permission: string) => {
  return (req: Request, res: Response, next: NextFunction): void => {
    if (!req.apiToken) {
      next(new AuthenticationError('Authentication required'));
      return;
    }

    if (!req.apiToken.permissions[permission]) {
      next(new AuthorizationError(`Permission '${permission}' required`));
      return;
    }

    next();
  };
};

/**
 * Optional authentication - continues even if no token present
 */
export const optionalAuth = async (
  req: Request,
  res: Response,
  next: NextFunction
): Promise<void> => {
  const authHeader = req.headers.authorization;
  
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    // No token provided, continue without authentication
    next();
    return;
  }

  // If token is provided, validate it
  authenticate(req, res, next);
};