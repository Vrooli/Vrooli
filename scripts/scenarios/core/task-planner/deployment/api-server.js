#!/usr/bin/env node

/**
 * Task Planner API Server
 * Provides REST endpoints for CLI and external integration
 */

const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const rateLimit = require('express-rate-limit');
const { Pool } = require('pg');
const Redis = require('ioredis');
const axios = require('axios');
const { v4: uuidv4 } = require('uuid');

// Configuration
const config = {
  port: process.env.TASK_PLANNER_API_PORT || 8092,
  host: process.env.TASK_PLANNER_API_HOST || 'localhost',
  cors: {
    origin: process.env.CORS_ORIGINS ? process.env.CORS_ORIGINS.split(',') : ['http://localhost:3000', 'http://localhost:8001'],
    credentials: true
  },
  database: {
    host: process.env.POSTGRES_HOST || 'localhost',
    port: process.env.POSTGRES_PORT || 5434,
    database: process.env.POSTGRES_DB || 'task_planner',
    user: process.env.POSTGRES_USER || 'taskplanner_user',
    password: process.env.POSTGRES_PASSWORD,
    max: 20,
    idleTimeoutMillis: 30000
  },
  redis: {
    host: process.env.REDIS_HOST || 'localhost',
    port: process.env.REDIS_PORT || 6380,
    keyPrefix: 'task_planner:',
    retryDelayOnFailover: 100,
    maxRetriesPerRequest: 3
  },
  n8n: {
    baseUrl: process.env.N8N_BASE_URL || 'http://localhost:5679',
    webhookBase: process.env.N8N_WEBHOOK_BASE || 'http://localhost:5679/webhook'
  },
  rateLimiting: {
    windowMs: 60 * 1000, // 1 minute
    max: 60, // requests per window
    message: 'Too many requests, please try again later'
  }
};

// Initialize Express app
const app = express();

// Database connection
const db = new Pool(config.database);

// Redis connection
const redis = new Redis(config.redis);

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
    // Check database connection
    await db.query('SELECT 1');
    
    // Check Redis connection
    await redis.ping();
    
    res.json({
      status: 'healthy',
      timestamp: new Date().toISOString(),
      version: '1.0.0',
      services: {
        database: 'connected',
        redis: 'connected'
      }
    });
  } catch (error) {
    res.status(503).json({
      status: 'unhealthy',
      timestamp: new Date().toISOString(),
      error: error.message
    });
  }
});

// Apps endpoints
app.get('/api/apps', async (req, res) => {
  try {
    const result = await db.query(`
      SELECT id, name, display_name, type, total_tasks, completed_tasks, 
             created_at, updated_at
      FROM apps 
      ORDER BY display_name
    `);
    
    res.json({
      apps: result.rows,
      count: result.rows.length
    });
  } catch (error) {
    console.error('Error fetching apps:', error);
    res.status(500).json({ error: 'Failed to fetch apps' });
  }
});

app.post('/api/apps', async (req, res) => {
  try {
    const { name, display_name, type = 'external', repository_url, webhook_url } = req.body;
    
    if (!name || !display_name) {
      return res.status(400).json({ error: 'name and display_name are required' });
    }
    
    const apiToken = `${name.substring(0, 3)}_${require('crypto').randomBytes(16).toString('hex')}`;
    
    const result = await db.query(`
      INSERT INTO apps (id, name, display_name, type, repository_url, webhook_url, api_token)
      VALUES ($1, $2, $3, $4, $5, $6, $7)
      RETURNING id, name, display_name, api_token, created_at
    `, [uuidv4(), name, display_name, type, repository_url, webhook_url, apiToken]);
    
    res.status(201).json({
      app: result.rows[0]
    });
  } catch (error) {
    if (error.code === '23505') { // Unique violation
      return res.status(409).json({ error: 'App name already exists' });
    }
    console.error('Error creating app:', error);
    res.status(500).json({ error: 'Failed to create app' });
  }
});

app.get('/api/apps/:app_id', async (req, res) => {
  try {
    const { app_id } = req.params;
    
    const result = await db.query(`
      SELECT a.*, 
             COUNT(t.id) FILTER (WHERE t.status = 'backlog') as backlog_tasks,
             COUNT(t.id) FILTER (WHERE t.status = 'staged') as staged_tasks,
             COUNT(t.id) FILTER (WHERE t.status = 'in_progress') as in_progress_tasks,
             COUNT(t.id) FILTER (WHERE t.status = 'completed') as completed_tasks,
             COUNT(t.id) FILTER (WHERE t.status = 'failed') as failed_tasks
      FROM apps a
      LEFT JOIN tasks t ON a.id = t.app_id
      WHERE a.id = $1
      GROUP BY a.id
    `, [app_id]);
    
    if (result.rows.length === 0) {
      return res.status(404).json({ error: 'App not found' });
    }
    
    res.json({
      app: result.rows[0]
    });
  } catch (error) {
    console.error('Error fetching app:', error);
    res.status(500).json({ error: 'Failed to fetch app' });
  }
});

// Task parsing endpoint
app.post('/api/parse-text', async (req, res) => {
  try {
    const { app_id, raw_text, api_token, input_type = 'markdown', submitted_by = 'api' } = req.body;
    
    if (!app_id || !raw_text || !api_token) {
      return res.status(400).json({ error: 'app_id, raw_text, and api_token are required' });
    }
    
    // Call n8n text parser workflow
    const n8nResponse = await axios.post(`${config.n8n.webhookBase}/parse-text`, {
      app_id,
      raw_text,
      api_token,
      input_type,
      submitted_by
    }, {
      timeout: 120000 // 2 minutes
    });
    
    res.json(n8nResponse.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    console.error('Error parsing text:', error);
    res.status(500).json({ error: 'Failed to parse text' });
  }
});

// Tasks endpoints
app.get('/api/tasks', async (req, res) => {
  try {
    const {
      app_id,
      status,
      priority,
      tags,
      limit = 50,
      offset = 0,
      sort = 'created_at',
      order = 'desc'
    } = req.query;
    
    let whereClause = '1=1';
    const params = [];
    let paramIndex = 1;
    
    if (app_id) {
      whereClause += ` AND t.app_id = $${paramIndex}`;
      params.push(app_id);
      paramIndex++;
    }
    
    if (status) {
      whereClause += ` AND t.status = $${paramIndex}`;
      params.push(status);
      paramIndex++;
    }
    
    if (priority) {
      whereClause += ` AND t.priority = $${paramIndex}`;
      params.push(priority);
      paramIndex++;
    }
    
    if (tags) {
      whereClause += ` AND t.tags && $${paramIndex}`;
      params.push(tags.split(','));
      paramIndex++;
    }
    
    const validSorts = ['created_at', 'updated_at', 'priority', 'title'];
    const sortColumn = validSorts.includes(sort) ? sort : 'created_at';
    const sortOrder = order.toLowerCase() === 'asc' ? 'ASC' : 'DESC';
    
    params.push(parseInt(limit), parseInt(offset));
    
    const result = await db.query(`
      SELECT t.id, t.title, t.description, t.status, t.priority, t.tags,
             t.estimated_hours, t.confidence_score, t.created_at, t.updated_at,
             a.display_name as app_name
      FROM tasks t
      JOIN apps a ON t.app_id = a.id
      WHERE ${whereClause}
      ORDER BY ${sortColumn} ${sortOrder}
      LIMIT $${paramIndex} OFFSET $${paramIndex + 1}
    `, params);
    
    res.json({
      tasks: result.rows,
      count: result.rows.length,
      pagination: {
        limit: parseInt(limit),
        offset: parseInt(offset)
      }
    });
  } catch (error) {
    console.error('Error fetching tasks:', error);
    res.status(500).json({ error: 'Failed to fetch tasks' });
  }
});

app.get('/api/tasks/:task_id', async (req, res) => {
  try {
    const { task_id } = req.params;
    
    const result = await db.query(`
      SELECT t.*, a.display_name as app_name,
             COUNT(ra.id) as research_artifacts_count
      FROM tasks t
      JOIN apps a ON t.app_id = a.id
      LEFT JOIN research_artifacts ra ON t.id = ra.task_id
      WHERE t.id = $1
      GROUP BY t.id, a.display_name
    `, [task_id]);
    
    if (result.rows.length === 0) {
      return res.status(404).json({ error: 'Task not found' });
    }
    
    res.json({
      task: result.rows[0]
    });
  } catch (error) {
    console.error('Error fetching task:', error);
    res.status(500).json({ error: 'Failed to fetch task' });
  }
});

// Task research endpoint
app.post('/api/tasks/:task_id/research', async (req, res) => {
  try {
    const { task_id } = req.params;
    const { force_refresh = false } = req.body;
    
    // Call n8n research workflow
    const n8nResponse = await axios.post(`${config.n8n.webhookBase}/research-task`, {
      task_id,
      force_refresh
    }, {
      timeout: 300000 // 5 minutes
    });
    
    res.json(n8nResponse.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    console.error('Error researching task:', error);
    res.status(500).json({ error: 'Failed to research task' });
  }
});

// Task implementation endpoint
app.post('/api/tasks/:task_id/implement', async (req, res) => {
  try {
    const { task_id } = req.params;
    const { override_staging = false, implementation_notes } = req.body;
    
    // Call n8n implementation workflow
    const n8nResponse = await axios.post(`${config.n8n.webhookBase}/implement-task`, {
      task_id,
      override_staging,
      implementation_notes
    }, {
      timeout: 600000 // 10 minutes
    });
    
    res.json(n8nResponse.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    console.error('Error implementing task:', error);
    res.status(500).json({ error: 'Failed to implement task' });
  }
});

// Task status update endpoint
app.put('/api/tasks/:task_id/status', async (req, res) => {
  try {
    const { task_id } = req.params;
    const { to_status, reason = 'Manual status update', notes = '' } = req.body;
    
    if (!to_status) {
      return res.status(400).json({ error: 'to_status is required' });
    }
    
    const validStatuses = ['unstructured', 'backlog', 'staged', 'in_progress', 'completed', 'cancelled', 'failed'];
    if (!validStatuses.includes(to_status)) {
      return res.status(400).json({ error: 'Invalid status' });
    }
    
    // Call n8n status update workflow
    const n8nResponse = await axios.post(`${config.n8n.webhookBase}/status-update`, {
      task_id,
      to_status,
      reason,
      notes,
      triggered_by: 'api'
    }, {
      timeout: 30000 // 30 seconds
    });
    
    res.json(n8nResponse.data);
  } catch (error) {
    if (error.response) {
      return res.status(error.response.status).json(error.response.data);
    }
    console.error('Error updating task status:', error);
    res.status(500).json({ error: 'Failed to update task status' });
  }
});

// Similar tasks endpoint
app.get('/api/tasks/:task_id/similar', async (req, res) => {
  try {
    const { task_id } = req.params;
    const { limit = 5, threshold = 0.3 } = req.query;
    
    const result = await db.query(`
      SELECT t2.id, t2.title, t2.description, t2.status, t2.priority,
             1 - (t1.title_embedding <-> t2.title_embedding) as similarity_score
      FROM tasks t1, tasks t2
      WHERE t1.id = $1 
        AND t2.id != $1
        AND t1.app_id = t2.app_id
        AND t2.title_embedding IS NOT NULL
        AND t1.title_embedding <-> t2.title_embedding < $3
      ORDER BY t1.title_embedding <-> t2.title_embedding
      LIMIT $2
    `, [task_id, parseInt(limit), parseFloat(threshold)]);
    
    res.json({
      similar_tasks: result.rows,
      count: result.rows.length
    });
  } catch (error) {
    console.error('Error finding similar tasks:', error);
    res.status(500).json({ error: 'Failed to find similar tasks' });
  }
});

// Research artifacts endpoint
app.get('/api/tasks/:task_id/artifacts', async (req, res) => {
  try {
    const { task_id } = req.params;
    
    const result = await db.query(`
      SELECT id, type, title, source_url, content, relevance_score, quality_score, created_at
      FROM research_artifacts
      WHERE task_id = $1
      ORDER BY relevance_score DESC, created_at DESC
    `, [task_id]);
    
    res.json({
      artifacts: result.rows,
      count: result.rows.length
    });
  } catch (error) {
    console.error('Error fetching research artifacts:', error);
    res.status(500).json({ error: 'Failed to fetch research artifacts' });
  }
});

// Analytics endpoints
app.get('/api/analytics/stats', async (req, res) => {
  try {
    const { app_id, time_range = '30d' } = req.query;
    
    let timeFilter = '';
    let params = [];
    let paramIndex = 1;
    
    if (app_id) {
      timeFilter += ` AND t.app_id = $${paramIndex}`;
      params.push(app_id);
      paramIndex++;
    }
    
    // Add time range filter
    const timeRanges = {
      '7d': 7,
      '30d': 30,
      '90d': 90,
      '1y': 365
    };
    const days = timeRanges[time_range] || 30;
    timeFilter += ` AND t.created_at >= CURRENT_TIMESTAMP - INTERVAL '${days} days'`;
    
    const result = await db.query(`
      SELECT 
        COUNT(*) as total_tasks,
        COUNT(*) FILTER (WHERE status = 'backlog') as backlog_tasks,
        COUNT(*) FILTER (WHERE status = 'staged') as staged_tasks,
        COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress_tasks,
        COUNT(*) FILTER (WHERE status = 'completed') as completed_tasks,
        COUNT(*) FILTER (WHERE status = 'failed') as failed_tasks,
        AVG(estimated_hours) FILTER (WHERE estimated_hours IS NOT NULL) as avg_estimated_hours,
        AVG(confidence_score) FILTER (WHERE confidence_score IS NOT NULL) as avg_confidence_score,
        COUNT(DISTINCT app_id) as active_apps
      FROM tasks t
      WHERE 1=1 ${timeFilter}
    `, params);
    
    res.json({
      stats: result.rows[0],
      time_range,
      generated_at: new Date().toISOString()
    });
  } catch (error) {
    console.error('Error fetching analytics:', error);
    res.status(500).json({ error: 'Failed to fetch analytics' });
  }
});

app.get('/api/analytics/activity', async (req, res) => {
  try {
    const { limit = 20, app_id } = req.query;
    
    let whereClause = '1=1';
    const params = [];
    let paramIndex = 1;
    
    if (app_id) {
      whereClause += ` AND t.app_id = $${paramIndex}`;
      params.push(app_id);
      paramIndex++;
    }
    
    params.push(parseInt(limit));
    
    const result = await db.query(`
      SELECT 
        tt.id, t.title as task_title, a.display_name as app_name,
        tt.from_status, tt.to_status, tt.triggered_by, tt.reason, tt.created_at
      FROM task_transitions tt
      JOIN tasks t ON tt.task_id = t.id
      JOIN apps a ON t.app_id = a.id
      WHERE ${whereClause}
      ORDER BY tt.created_at DESC
      LIMIT $${paramIndex}
    `, params);
    
    res.json({
      activities: result.rows,
      count: result.rows.length
    });
  } catch (error) {
    console.error('Error fetching activity:', error);
    res.status(500).json({ error: 'Failed to fetch activity' });
  }
});

// Catch-all for undefined API routes
app.use('/api/*', (req, res) => {
  res.status(404).json({
    error: 'API endpoint not found',
    path: req.originalUrl,
    method: req.method
  });
});

// Start server
const server = app.listen(config.port, config.host, () => {
  console.log(`🚀 Task Planner API Server running on http://${config.host}:${config.port}`);
  console.log(`📚 API Documentation: http://${config.host}:${config.port}/api/health`);
  console.log(`⚡ Environment: ${process.env.NODE_ENV || 'development'}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully...');
  server.close(() => {
    console.log('HTTP server closed');
    db.end();
    redis.disconnect();
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('SIGINT received, shutting down gracefully...');
  server.close(() => {
    console.log('HTTP server closed');
    db.end();
    redis.disconnect();
    process.exit(0);
  });
});

module.exports = app;