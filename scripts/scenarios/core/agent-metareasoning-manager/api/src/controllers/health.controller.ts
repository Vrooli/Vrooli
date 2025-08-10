import { Request, Response } from 'express';
import { getDbPool, getRedisClient } from '../config/database.js';
import { config } from '../config/index.js';
import { asyncHandler } from '../utils/asyncHandler.js';
import axios from 'axios';

class HealthController {
  /**
   * Health check endpoint
   */
  health = asyncHandler(async (_req: Request, res: Response) => {
    const services: Record<string, 'healthy' | 'unhealthy'> = {};
    let overallStatus: 'healthy' | 'degraded' | 'unhealthy' = 'healthy';

    // Check database
    try {
      const db = getDbPool();
      await db.query('SELECT 1');
      services.database = 'healthy';
    } catch (error) {
      services.database = 'unhealthy';
      overallStatus = 'degraded';
    }

    // Check Redis
    try {
      const redis = await getRedisClient();
      await redis.ping();
      services.redis = 'healthy';
    } catch (error) {
      services.redis = 'unhealthy';
      overallStatus = 'degraded';
    }

    // Check n8n
    try {
      await axios.get(`${config.n8n.baseUrl}/healthz`, { timeout: 5000 });
      services.n8n = 'healthy';
    } catch (error) {
      services.n8n = 'unhealthy';
      // n8n is optional, so don't degrade overall status
    }

    // Check Windmill
    try {
      await axios.get(`${config.windmill.baseUrl}/api/version`, { timeout: 5000 });
      services.windmill = 'healthy';
    } catch (error) {
      services.windmill = 'unhealthy';
      // Windmill is optional, so don't degrade overall status
    }

    // If any critical service is unhealthy, overall is unhealthy
    if (services.database === 'unhealthy' || services.redis === 'unhealthy') {
      overallStatus = 'unhealthy';
    }

    const statusCode = overallStatus === 'healthy' ? 200 : 
                       overallStatus === 'degraded' ? 206 : 503;

    res.status(statusCode).json({
      status: overallStatus,
      timestamp: new Date().toISOString(),
      version: '1.0.0',
      uptime: process.uptime(),
      services
    });
  });

  /**
   * Basic info endpoint
   */
  info = (_req: Request, res: Response) => {
    res.json({
      name: 'Agent Metareasoning Manager API',
      version: '1.0.0',
      description: 'API for managing metareasoning tools and workflows',
      endpoints: {
        health: '/api/health',
        prompts: '/api/prompts',
        workflows: '/api/workflows',
        analysis: '/api/analyze/*',
        templates: '/api/templates'
      },
      documentation: '/api/docs'
    });
  };

  /**
   * Get API statistics
   */
  stats = asyncHandler(async (_req: Request, res: Response) => {
    const db = getDbPool();
    
    // Get various statistics
    const [promptCount, workflowCount, templateCount, recentUsage] = await Promise.all([
      db.query('SELECT COUNT(*) as count FROM prompts WHERE status = $1', ['active']),
      db.query('SELECT COUNT(*) as count FROM workflows WHERE status = $1', ['active']),
      db.query('SELECT COUNT(*) as count FROM templates'),
      db.query(`
        SELECT 
          DATE(created_at) as date,
          COUNT(*) as requests
        FROM api_usage_stats
        WHERE created_at > NOW() - INTERVAL '7 days'
        GROUP BY DATE(created_at)
        ORDER BY date DESC
      `)
    ]);

    res.json({
      status: 'success',
      data: {
        resources: {
          prompts: parseInt(promptCount.rows[0].count),
          workflows: parseInt(workflowCount.rows[0].count),
          templates: parseInt(templateCount.rows[0].count)
        },
        usage: {
          last_7_days: recentUsage.rows
        },
        system: {
          uptime_seconds: process.uptime(),
          memory_usage_mb: Math.round(process.memoryUsage().heapUsed / 1024 / 1024),
          node_version: process.version
        }
      }
    });
  });
}

export const healthController = new HealthController();