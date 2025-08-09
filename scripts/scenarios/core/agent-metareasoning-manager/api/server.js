#!/usr/bin/env node

/**
 * Agent Metareasoning Manager API Server
 * Provides REST endpoints for CLI and external integration
 */

const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const rateLimit = require('express-rate-limit');
const { Pool } = require('pg');
const { createClient } = require('redis');
const axios = require('axios');
const { v4: uuidv4 } = require('uuid');

// Configuration
const config = {
  port: process.env.METAREASONING_API_PORT || 8093,
  host: process.env.METAREASONING_API_HOST || 'localhost',
  cors: {
    origin: process.env.CORS_ORIGINS ? process.env.CORS_ORIGINS.split(',') : ['http://localhost:3000', 'http://localhost:8001'],
    credentials: true
  },
  database: {
    host: process.env.POSTGRES_HOST || 'localhost',
    port: process.env.POSTGRES_PORT || 5432,
    database: process.env.POSTGRES_DB || 'postgres',
    user: process.env.POSTGRES_USER || 'postgres',
    password: process.env.POSTGRES_PASSWORD || '',
    max: 20,
    idleTimeoutMillis: 30000
  },
  redis: {
    url: `redis://localhost:${process.env.REDIS_PORT || 6379}`,
    keyPrefix: 'metareasoning:',
    retryDelayOnFailover: 100,
    socket: {
      connectTimeout: 5000,
      lazyConnect: true
    }
  },
  n8n: {
    baseUrl: process.env.N8N_BASE_URL || `http://localhost:${process.env.N8N_PORT || 5678}`,
    webhookBase: process.env.N8N_WEBHOOK_BASE || `http://localhost:${process.env.N8N_PORT || 5678}/webhook`
  },
  windmill: {
    baseUrl: process.env.WINDMILL_BASE_URL || `http://localhost:${process.env.WINDMILL_PORT || 8000}`,
    workspace: 'demo'
  },
  rateLimiting: {
    windowMs: 60 * 1000, // 1 minute
    max: 100, // requests per window
    message: 'Too many requests, please try again later'
  }
};

// Initialize Express app
const app = express();

// Database connection
const db = new Pool(config.database);

// Redis connection
const redis = createClient(config.redis);

// Middleware
app.use(helmet());
app.use(cors(config.cors));
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

// Rate limiting
const limiter = rateLimit({
  windowMs: config.rateLimiting.windowMs,
  max: config.rateLimiting.max,
  message: { error: config.rateLimiting.message },
  standardHeaders: true,
  legacyHeaders: false
});
app.use('/api/', limiter);

// Request logging middleware
app.use((req, res, next) => {
  const start = Date.now();
  const requestId = uuidv4();
  req.requestId = requestId;
  
  res.on('finish', () => {
    const duration = Date.now() - start;
    console.log(`[${new Date().toISOString()}] ${requestId} ${req.method} ${req.originalUrl} ${res.statusCode} ${duration}ms`);
  });
  
  next();
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error(`[${new Date().toISOString()}] Error in ${req.method} ${req.originalUrl}:`, err);
  
  const status = err.status || 500;
  const message = status === 500 ? 'Internal server error' : err.message;
  
  res.status(status).json({
    error: message,
    requestId: req.requestId,
    timestamp: new Date().toISOString()
  });
});

// Health check endpoint
app.get('/api/health', async (req, res) => {
  try {
    const healthChecks = {};
    
    // Check database connection
    try {
      await db.query('SELECT 1');
      healthChecks.database = 'connected';
    } catch (dbErr) {
      healthChecks.database = 'disconnected';
      healthChecks.database_error = dbErr.message;
    }
    
    // Check Redis connection
    try {
      if (!redis.isOpen) {
        await redis.connect();
      }
      await redis.ping();
      healthChecks.redis = 'connected';
    } catch (redisErr) {
      healthChecks.redis = 'disconnected';
      healthChecks.redis_error = redisErr.message;
    }
    
    // Check n8n connection
    try {
      const n8nResponse = await axios.get(`${config.n8n.baseUrl}/healthz`, { timeout: 5000 });
      healthChecks.n8n = n8nResponse.status === 200 ? 'connected' : 'disconnected';
    } catch (n8nErr) {
      healthChecks.n8n = 'disconnected';
      healthChecks.n8n_error = n8nErr.message;
    }
    
    // Check Windmill connection
    try {
      const windmillResponse = await axios.get(`${config.windmill.baseUrl}/api/version`, { timeout: 5000 });
      healthChecks.windmill = windmillResponse.status === 200 ? 'connected' : 'disconnected';
    } catch (windmillErr) {
      healthChecks.windmill = 'disconnected';
      healthChecks.windmill_error = windmillErr.message;
    }
    
    const allHealthy = Object.values(healthChecks).every(status => 
      typeof status === 'string' && status === 'connected'
    );
    
    res.status(allHealthy ? 200 : 503).json({
      status: allHealthy ? 'healthy' : 'degraded',
      timestamp: new Date().toISOString(),
      version: '1.0.0',
      services: healthChecks
    });
  } catch (error) {
    res.status(503).json({
      status: 'unhealthy',
      timestamp: new Date().toISOString(),
      error: error.message,
      requestId: req.requestId
    });
  }
});

// Basic info endpoint
app.get('/api/info', (req, res) => {
  res.json({
    name: 'Agent Metareasoning Manager API',
    version: '1.0.0',
    description: 'API for managing metareasoning tools and decision support workflows',
    endpoints: {
      health: '/api/health',
      info: '/api/info',
      prompts: '/api/prompts',
      workflows: '/api/workflows',
      analyze: '/api/analyze',
      templates: '/api/templates'
    },
    timestamp: new Date().toISOString()
  });
});

// Placeholder endpoints for future implementation

// Authentication middleware
const authenticateToken = async (req, res, next) => {
  const authHeader = req.headers['authorization'];
  const token = authHeader && authHeader.split(' ')[1]; // Bearer TOKEN
  
  if (!token) {
    return next(); // Allow requests without tokens for now
  }
  
  try {
    // Hash the token and check against database
    const tokenHash = require('crypto').createHash('sha256').update(token).digest('hex');
    const result = await db.query(
      'SELECT * FROM api_tokens WHERE token_hash = $1 AND is_active = true',
      [tokenHash]
    );
    
    if (result.rows.length === 0) {
      return res.status(401).json({ error: 'Invalid or inactive token' });
    }
    
    req.apiToken = result.rows[0];
    
    // Update last used timestamp
    await db.query(
      'UPDATE api_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1',
      [req.apiToken.id]
    );
    
    next();
  } catch (error) {
    console.error('Authentication error:', error);
    return res.status(500).json({ error: 'Authentication failed' });
  }
};

// Prompts management
app.get('/api/prompts', authenticateToken, async (req, res) => {
  try {
    const { category, pattern, limit = 50, offset = 0 } = req.query;
    
    let query = `
      SELECT id, name, description, category, pattern, variables, metadata, 
             tags, usage_count, average_rating, created_at, updated_at
      FROM prompts WHERE is_active = true
    `;
    
    const params = [];
    let paramCount = 0;
    
    if (category) {
      query += ` AND category = $${++paramCount}`;
      params.push(category);
    }
    
    if (pattern) {
      query += ` AND pattern = $${++paramCount}`;
      params.push(pattern);
    }
    
    query += ` ORDER BY usage_count DESC, average_rating DESC NULLS LAST`;
    query += ` LIMIT $${++paramCount} OFFSET $${++paramCount}`;
    params.push(limit, offset);
    
    const result = await db.query(query, params);
    const countResult = await db.query('SELECT COUNT(*) FROM prompts WHERE is_active = true');
    
    res.json({
      prompts: result.rows,
      count: result.rows.length,
      total: parseInt(countResult.rows[0].count),
      offset: parseInt(offset),
      limit: parseInt(limit)
    });
    
    // Log API usage
    await logApiUsage(req, res, req.apiToken?.id);
  } catch (error) {
    console.error('Error fetching prompts:', error);
    res.status(500).json({ error: 'Failed to fetch prompts' });
  }
});

app.get('/api/prompts/:id', authenticateToken, async (req, res) => {
  try {
    const { id } = req.params;
    
    const result = await db.query(
      'SELECT * FROM prompts WHERE id = $1 AND is_active = true',
      [id]
    );
    
    if (result.rows.length === 0) {
      return res.status(404).json({ error: 'Prompt not found' });
    }
    
    res.json({ prompt: result.rows[0] });
    await logApiUsage(req, res, req.apiToken?.id);
  } catch (error) {
    console.error('Error fetching prompt:', error);
    res.status(500).json({ error: 'Failed to fetch prompt' });
  }
});

// Workflow management
app.get('/api/workflows', authenticateToken, async (req, res) => {
  try {
    const { platform, pattern, limit = 50, offset = 0 } = req.query;
    
    let query = `
      SELECT id, name, description, platform, pattern, input_schema, output_schema,
             dependencies, tags, execution_count, average_duration_ms, created_at
      FROM workflows WHERE is_active = true
    `;
    
    const params = [];
    let paramCount = 0;
    
    if (platform) {
      query += ` AND platform = $${++paramCount}`;
      params.push(platform);
    }
    
    if (pattern) {
      query += ` AND pattern = $${++paramCount}`;
      params.push(pattern);
    }
    
    query += ` ORDER BY execution_count DESC`;
    query += ` LIMIT $${++paramCount} OFFSET $${++paramCount}`;
    params.push(limit, offset);
    
    const result = await db.query(query, params);
    const countResult = await db.query('SELECT COUNT(*) FROM workflows WHERE is_active = true');
    
    res.json({
      workflows: result.rows,
      count: result.rows.length,
      total: parseInt(countResult.rows[0].count),
      offset: parseInt(offset),
      limit: parseInt(limit)
    });
    
    await logApiUsage(req, res, req.apiToken?.id);
  } catch (error) {
    console.error('Error fetching workflows:', error);
    res.status(500).json({ error: 'Failed to fetch workflows' });
  }
});

// Analysis endpoints
app.post('/api/analyze/decision', authenticateToken, async (req, res) => {
  try {
    const { input, context, options = {} } = req.body;
    
    if (!input) {
      return res.status(400).json({ error: 'Input is required for decision analysis' });
    }
    
    // Create analysis execution record
    const executionResult = await db.query(`
      INSERT INTO analysis_executions (type, input_data, status, api_token_id)
      VALUES ($1, $2, $3, $4)
      RETURNING id
    `, ['decision', JSON.stringify({ input, context, options }), 'pending', req.apiToken?.id]);
    
    const executionId = executionResult.rows[0].id;
    
    // For now, return a mock analysis (TODO: integrate with actual n8n workflows)
    const mockAnalysis = {
      decision_analysis: {
        input: input,
        context: context || 'No specific context provided',
        factors_considered: ['feasibility', 'impact', 'resources', 'timeline', 'risks'],
        recommendation: 'Proceed with caution - conduct pilot test first',
        confidence: 75,
        reasoning: 'Based on the complexity and potential impact, a phased approach is recommended'
      },
      execution_id: executionId,
      status: 'completed',
      analysis_type: 'decision'
    };
    
    // Update execution record with results
    await db.query(`
      UPDATE analysis_executions 
      SET output_data = $1, status = $2, execution_time_ms = $3, completed_at = CURRENT_TIMESTAMP
      WHERE id = $4
    `, [JSON.stringify(mockAnalysis), 'completed', 1500, executionId]);
    
    res.json(mockAnalysis);
    await logApiUsage(req, res, req.apiToken?.id);
  } catch (error) {
    console.error('Error in decision analysis:', error);
    res.status(500).json({ error: 'Decision analysis failed' });
  }
});

app.post('/api/analyze/pros-cons', authenticateToken, async (req, res) => {
  try {
    const { input, context, options = {} } = req.body;
    
    if (!input) {
      return res.status(400).json({ error: 'Input is required for pros-cons analysis' });
    }
    
    const executionResult = await db.query(`
      INSERT INTO analysis_executions (type, input_data, status, api_token_id)
      VALUES ($1, $2, $3, $4)
      RETURNING id
    `, ['pros_cons', JSON.stringify({ input, context, options }), 'pending', req.apiToken?.id]);
    
    const executionId = executionResult.rows[0].id;
    
    const mockAnalysis = {
      pros_cons_analysis: {
        input: input,
        context: context || 'General context',
        pros: [
          { item: 'Potential for significant improvement', weight: 8, explanation: 'High impact on objectives' },
          { item: 'Leverages existing capabilities', weight: 7, explanation: 'Builds on current strengths' },
          { item: 'Market opportunity', weight: 6, explanation: 'Timing appears favorable' }
        ],
        cons: [
          { item: 'Implementation complexity', weight: 7, explanation: 'Requires significant technical changes' },
          { item: 'Resource requirements', weight: 6, explanation: 'Substantial investment needed' },
          { item: 'Execution risks', weight: 5, explanation: 'Several failure points identified' }
        ],
        net_score: 4, // Sum of weighted pros minus cons
        recommendation: 'Proceed with careful planning and risk mitigation',
        confidence: 80
      },
      execution_id: executionId,
      status: 'completed',
      analysis_type: 'pros_cons'
    };
    
    await db.query(`
      UPDATE analysis_executions 
      SET output_data = $1, status = $2, execution_time_ms = $3, completed_at = CURRENT_TIMESTAMP
      WHERE id = $4
    `, [JSON.stringify(mockAnalysis), 'completed', 2200, executionId]);
    
    res.json(mockAnalysis);
    await logApiUsage(req, res, req.apiToken?.id);
  } catch (error) {
    console.error('Error in pros-cons analysis:', error);
    res.status(500).json({ error: 'Pros-cons analysis failed' });
  }
});

app.post('/api/analyze/swot', authenticateToken, async (req, res) => {
  try {
    const { input, context, options = {} } = req.body;
    
    if (!input) {
      return res.status(400).json({ error: 'Input is required for SWOT analysis' });
    }
    
    const executionResult = await db.query(`
      INSERT INTO analysis_executions (type, input_data, status, api_token_id)
      VALUES ($1, $2, $3, $4)
      RETURNING id
    `, ['swot', JSON.stringify({ input, context, options }), 'pending', req.apiToken?.id]);
    
    const executionId = executionResult.rows[0].id;
    
    const mockAnalysis = {
      swot_analysis: {
        input: input,
        context: context || 'Strategic context',
        strengths: [
          'Strong technical capabilities',
          'Established market presence',
          'Experienced team',
          'Financial stability'
        ],
        weaknesses: [
          'Limited marketing reach',
          'Outdated infrastructure',
          'Skill gaps in emerging technologies'
        ],
        opportunities: [
          'Growing market demand',
          'New technology trends',
          'Potential partnerships',
          'Regulatory changes favorable'
        ],
        threats: [
          'Increased competition',
          'Economic uncertainty',
          'Technology disruption',
          'Regulatory risks'
        ],
        strategic_position: 'Diversified Growth',
        recommendations: [
          'Leverage strengths to capture opportunities',
          'Address infrastructure weaknesses',
          'Develop competitive moats against threats'
        ]
      },
      execution_id: executionId,
      status: 'completed',
      analysis_type: 'swot'
    };
    
    await db.query(`
      UPDATE analysis_executions 
      SET output_data = $1, status = $2, execution_time_ms = $3, completed_at = CURRENT_TIMESTAMP
      WHERE id = $4
    `, [JSON.stringify(mockAnalysis), 'completed', 3100, executionId]);
    
    res.json(mockAnalysis);
    await logApiUsage(req, res, req.apiToken?.id);
  } catch (error) {
    console.error('Error in SWOT analysis:', error);
    res.status(500).json({ error: 'SWOT analysis failed' });
  }
});

app.post('/api/analyze/risks', authenticateToken, async (req, res) => {
  try {
    const { action, constraints, options = {} } = req.body;
    
    if (!action) {
      return res.status(400).json({ error: 'Action is required for risk analysis' });
    }
    
    const executionResult = await db.query(`
      INSERT INTO analysis_executions (type, input_data, status, api_token_id)
      VALUES ($1, $2, $3, $4)
      RETURNING id
    `, ['risk_assessment', JSON.stringify({ action, constraints, options }), 'pending', req.apiToken?.id]);
    
    const executionId = executionResult.rows[0].id;
    
    const mockAnalysis = {
      risk_assessment: {
        action: action,
        constraints: constraints || 'No specific constraints provided',
        risks: [
          {
            risk: 'Technical implementation failure',
            category: 'Technical',
            probability: 'Medium',
            impact: 'High',
            risk_score: 6,
            mitigation: 'Extensive testing and phased rollout',
            early_warning: 'Performance degradation in testing'
          },
          {
            risk: 'Budget overrun',
            category: 'Financial', 
            probability: 'High',
            impact: 'Medium',
            risk_score: 6,
            mitigation: 'Regular budget reviews and scope controls',
            early_warning: '10% variance from planned spend'
          },
          {
            risk: 'Timeline delays',
            category: 'Operational',
            probability: 'Medium',
            impact: 'Medium', 
            risk_score: 4,
            mitigation: 'Buffer time and parallel workstreams',
            early_warning: 'Missing intermediate milestones'
          }
        ],
        overall_risk_posture: 'Medium-High',
        top_priorities: ['Technical failure', 'Budget control', 'Timeline management'],
        mitigation_summary: 'Focus on technical validation and financial controls'
      },
      execution_id: executionId,
      status: 'completed',
      analysis_type: 'risk_assessment'
    };
    
    await db.query(`
      UPDATE analysis_executions 
      SET output_data = $1, status = $2, execution_time_ms = $3, completed_at = CURRENT_TIMESTAMP
      WHERE id = $4
    `, [JSON.stringify(mockAnalysis), 'completed', 2800, executionId]);
    
    res.json(mockAnalysis);
    await logApiUsage(req, res, req.apiToken?.id);
  } catch (error) {
    console.error('Error in risk analysis:', error);
    res.status(500).json({ error: 'Risk analysis failed' });
  }
});

// Templates management
app.get('/api/templates', authenticateToken, async (req, res) => {
  try {
    const { pattern, is_public, limit = 50, offset = 0 } = req.query;
    
    let query = `
      SELECT id, name, description, pattern, structure, example_usage, 
             best_practices, limitations, tags, is_public, usage_count, created_at
      FROM templates
    `;
    
    const params = [];
    let paramCount = 0;
    let whereClause = [];
    
    if (pattern) {
      whereClause.push(`pattern = $${++paramCount}`);
      params.push(pattern);
    }
    
    if (is_public !== undefined) {
      whereClause.push(`is_public = $${++paramCount}`);
      params.push(is_public === 'true');
    }
    
    if (whereClause.length > 0) {
      query += ' WHERE ' + whereClause.join(' AND ');
    }
    
    query += ` ORDER BY usage_count DESC, created_at DESC`;
    query += ` LIMIT $${++paramCount} OFFSET $${++paramCount}`;
    params.push(limit, offset);
    
    const result = await db.query(query, params);
    const countQuery = whereClause.length > 0 
      ? 'SELECT COUNT(*) FROM templates WHERE ' + whereClause.join(' AND ')
      : 'SELECT COUNT(*) FROM templates';
    const countResult = await db.query(countQuery, params.slice(0, -2));
    
    res.json({
      templates: result.rows,
      count: result.rows.length,
      total: parseInt(countResult.rows[0].count),
      offset: parseInt(offset),
      limit: parseInt(limit)
    });
    
    await logApiUsage(req, res, req.apiToken?.id);
  } catch (error) {
    console.error('Error fetching templates:', error);
    res.status(500).json({ error: 'Failed to fetch templates' });
  }
});

// Helper function to log API usage
async function logApiUsage(req, res, tokenId) {
  try {
    await db.query(`
      INSERT INTO api_usage_stats (endpoint, method, response_code, execution_time_ms, api_token_id)
      VALUES ($1, $2, $3, $4, $5)
    `, [
      req.originalUrl,
      req.method,
      res.statusCode,
      Date.now() - req.startTime || 0,
      tokenId
    ]);
  } catch (error) {
    console.error('Error logging API usage:', error);
  }
}

// 404 handler
app.use('*', (req, res) => {
  res.status(404).json({
    error: 'Endpoint not found',
    path: req.originalUrl,
    method: req.method,
    timestamp: new Date().toISOString()
  });
});

// Graceful shutdown handler
process.on('SIGTERM', async () => {
  console.log('Received SIGTERM, shutting down gracefully...');
  
  // Close database connections
  await db.end();
  
  // Close Redis connection
  if (redis.isOpen) {
    await redis.quit();
  }
  
  process.exit(0);
});

process.on('SIGINT', async () => {
  console.log('Received SIGINT, shutting down gracefully...');
  
  // Close database connections
  await db.end();
  
  // Close Redis connection
  if (redis.isOpen) {
    await redis.quit();
  }
  
  process.exit(0);
});

// Start server
const server = app.listen(config.port, config.host, () => {
  console.log(`🚀 Agent Metareasoning Manager API Server started`);
  console.log(`   URL: http://${config.host}:${config.port}`);
  console.log(`   Health: http://${config.host}:${config.port}/api/health`);
  console.log(`   Info: http://${config.host}:${config.port}/api/info`);
  console.log(`   Environment: ${process.env.NODE_ENV || 'development'}`);
});

// Handle server errors
server.on('error', (err) => {
  console.error('Server error:', err);
  process.exit(1);
});

module.exports = app;