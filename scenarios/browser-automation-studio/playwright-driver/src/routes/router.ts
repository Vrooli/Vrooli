/**
 * Lightweight Router
 *
 * Simple routing utility that replaces the large if/else chain in server.ts.
 * Keeps route definitions declarative and easier to maintain.
 *
 * Responsibilities:
 * - Route registration by path pattern and method
 * - Path parameter extraction (e.g., :id from /session/:id/run)
 * - Method-based dispatch
 * - 404/405 handling
 */

import type { IncomingMessage, ServerResponse } from 'http';
import { send404, send405 } from '../middleware';
import { logger } from '../utils';

/**
 * Route handler function signature.
 */
export type RouteHandler = (
  req: IncomingMessage,
  res: ServerResponse,
  params: Record<string, string>
) => Promise<void>;

/**
 * Route definition.
 */
interface Route {
  method: string;
  pattern: RegExp;
  paramNames: string[];
  handler: RouteHandler;
}

/**
 * Simple HTTP router with path parameter support.
 */
export class Router {
  private routes: Route[] = [];

  /**
   * Register a GET route.
   */
  get(path: string, handler: RouteHandler): void {
    this.addRoute('GET', path, handler);
  }

  /**
   * Register a POST route.
   */
  post(path: string, handler: RouteHandler): void {
    this.addRoute('POST', path, handler);
  }

  /**
   * Add a route with the given method and path.
   */
  private addRoute(method: string, path: string, handler: RouteHandler): void {
    const { pattern, paramNames } = this.compilePath(path);
    this.routes.push({ method, pattern, paramNames, handler });
  }

  /**
   * Compile a path pattern into a regex and extract param names.
   *
   * Examples:
   * - "/health" → /^\/health$/
   * - "/session/:id/run" → /^\/session\/([^/]+)\/run$/
   */
  private compilePath(path: string): { pattern: RegExp; paramNames: string[] } {
    const paramNames: string[] = [];

    // Escape special regex chars, then replace :param with capturing groups
    const regexStr = path
      .replace(/[.*+?^${}()|[\]\\]/g, '\\$&') // Escape regex special chars
      .replace(/:([a-zA-Z_][a-zA-Z0-9_]*)/g, (_match, paramName) => {
        paramNames.push(paramName);
        return '([^/]+)';
      });

    return {
      pattern: new RegExp(`^${regexStr}$`),
      paramNames,
    };
  }

  /**
   * Handle an incoming request by matching against registered routes.
   *
   * @returns true if a route was found and handled, false otherwise
   */
  async handle(req: IncomingMessage, res: ServerResponse): Promise<boolean> {
    const pathname = new URL(req.url || '/', `http://localhost`).pathname;
    const method = req.method || 'GET';

    // Find matching routes by path
    const matchingPaths: Array<{ route: Route; params: Record<string, string> }> = [];

    for (const route of this.routes) {
      const match = pathname.match(route.pattern);
      if (match) {
        const params: Record<string, string> = {};
        route.paramNames.forEach((name, index) => {
          params[name] = match[index + 1];
        });
        matchingPaths.push({ route, params });
      }
    }

    if (matchingPaths.length === 0) {
      // No path match at all - 404
      logger.debug('No route match', { method, pathname });
      send404(res, `Path not found: ${pathname}`);
      return false;
    }

    // Find exact method match
    const exactMatch = matchingPaths.find((m) => m.route.method === method);
    if (exactMatch) {
      logger.debug('Route matched', {
        method,
        pathname,
        params: exactMatch.params,
      });
      await exactMatch.route.handler(req, res, exactMatch.params);
      return true;
    }

    // Path exists but method not allowed - 405
    const allowedMethods = [...new Set(matchingPaths.map((m) => m.route.method))];
    logger.debug('Method not allowed', { method, pathname, allowedMethods });
    send405(res, allowedMethods);
    return false;
  }
}

/**
 * Create a new router instance.
 */
export function createRouter(): Router {
  return new Router();
}
