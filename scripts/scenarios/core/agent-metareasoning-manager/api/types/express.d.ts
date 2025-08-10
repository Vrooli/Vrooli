/**
 * Express type extensions for Agent Metareasoning Manager API
 */

import { ApiToken } from './index.js';

declare global {
  namespace Express {
    interface Request {
      requestId?: string;
      apiToken?: ApiToken;
      startTime?: number;
    }
  }
}

export {};