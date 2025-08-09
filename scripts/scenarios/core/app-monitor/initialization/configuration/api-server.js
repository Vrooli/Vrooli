const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const rateLimit = require('express-rate-limit');
const bodyParser = require('body-parser');
const { Pool } = require('pg');
const Redis = require('redis');
const Docker = require('dockerode');
const fs = require('fs');
const path = require('path');

// Load configuration
const configPath = path.join(__dirname, 'app-config.json');
const resourceUrlsPath = path.join(__dirname, 'resource-urls.json');

let config, resourceUrls;
try {
  config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
  resourceUrls = JSON.parse(fs.readFileSync(resourceUrlsPath, 'utf8'));
} catch (error) {
  console.error('Failed to load configuration:', error.message);
  process.exit(1);
}

const app = express();
const port = config.server.port || 8081;

// Security middleware
app.use(helmet());
app.use(cors({
  origin: config.server.cors.origins,
  credentials: true
}));

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: config.server.rateLimit.max || 100
});
app.use(limiter);

// Body parsing
app.use(bodyParser.json());
app.use(bodyParser.urlencoded({ extended: true }));

// Database connection
const pool = new Pool({
  connectionString: resourceUrls.resources.storage.postgres.url,
  max: resourceUrls.resources.storage.postgres.maxConnections
});

// Redis connection
const redis = Redis.createClient({
  url: resourceUrls.resources.storage.redis.url,
  retry_strategy: (times) => Math.min(times * 50, 2000)
});

// Docker connection
const docker = new Docker({
  socketPath: resourceUrls.resources.docker.api.socketPath
});

// Initialize connections
async function initializeConnections() {
  try {
    await pool.query('SELECT NOW()');
    console.log('✅ PostgreSQL connected');
    
    await redis.connect();
    console.log('✅ Redis connected');
    
    await docker.ping();
    console.log('✅ Docker connected');
  } catch (error) {
    console.error('❌ Failed to initialize connections:', error.message);
    process.exit(1);
  }
}

// Health check endpoint
app.get('/health', async (req, res) => {
  try {
    const dbCheck = await pool.query('SELECT 1');
    const redisCheck = await redis.ping();
    const dockerCheck = await docker.ping();
    
    res.json({
      status: 'healthy',
      timestamp: new Date().toISOString(),
      services: {
        database: dbCheck.rowCount === 1 ? 'ok' : 'error',
        redis: redisCheck === 'PONG' ? 'ok' : 'error',
        docker: dockerCheck ? 'ok' : 'error'
      }
    });
  } catch (error) {
    res.status(500).json({
      status: 'unhealthy',
      error: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

// API health endpoint
app.get('/api/health', (req, res) => {
  res.json({
    status: 'healthy',
    api: 'app-monitor',
    version: '1.0.0',
    timestamp: new Date().toISOString()
  });
});

// Get all apps
app.get('/api/apps', async (req, res) => {
  try {
    const result = await pool.query(`
      SELECT a.*, 
             s.status as current_status,
             s.cpu_usage,
             s.memory_usage,
             s.timestamp as last_update
      FROM apps a
      LEFT JOIN LATERAL (
        SELECT status, cpu_usage, memory_usage, timestamp
        FROM app_status 
        WHERE app_id = a.id 
        ORDER BY timestamp DESC 
        LIMIT 1
      ) s ON true
      ORDER BY a.name
    `);
    
    res.json(result.rows);
  } catch (error) {
    console.error('Error fetching apps:', error);
    res.status(500).json({ error: 'Failed to fetch apps' });
  }
});

// Get app by ID
app.get('/api/apps/:id', async (req, res) => {
  try {
    const { id } = req.params;
    const result = await pool.query(`
      SELECT a.*, 
             s.status as current_status,
             s.cpu_usage,
             s.memory_usage,
             s.network_in,
             s.network_out,
             s.timestamp as last_update
      FROM apps a
      LEFT JOIN LATERAL (
        SELECT status, cpu_usage, memory_usage, network_in, network_out, timestamp
        FROM app_status 
        WHERE app_id = a.id 
        ORDER BY timestamp DESC 
        LIMIT 1
      ) s ON true
      WHERE a.id = $1
    `, [id]);
    
    if (result.rows.length === 0) {
      return res.status(404).json({ error: 'App not found' });
    }
    
    res.json(result.rows[0]);
  } catch (error) {
    console.error('Error fetching app:', error);
    res.status(500).json({ error: 'Failed to fetch app' });
  }
});

// Start app
app.post('/api/apps/:id/start', async (req, res) => {
  try {
    const { id } = req.params;
    
    // Get app details
    const appResult = await pool.query('SELECT * FROM apps WHERE id = $1', [id]);
    if (appResult.rows.length === 0) {
      return res.status(404).json({ error: 'App not found' });
    }
    
    const app = appResult.rows[0];
    const containerName = app.config?.container_name || app.name;
    
    // Start container using Docker API
    const container = docker.getContainer(containerName);
    await container.start();
    
    // Update app status
    await pool.query(
      'UPDATE apps SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2',
      ['running', id]
    );
    
    // Log action
    await pool.query(`
      INSERT INTO app_logs (app_id, level, message, source, timestamp)
      VALUES ($1, 'info', 'App started via API', 'api-server', CURRENT_TIMESTAMP)
    `, [id]);
    
    // Publish event
    await redis.publish('app-events', JSON.stringify({
      type: 'app_started',
      app_id: id,
      app_name: app.name,
      timestamp: new Date().toISOString()
    }));
    
    res.json({ message: 'App started successfully' });
  } catch (error) {
    console.error('Error starting app:', error);
    res.status(500).json({ error: 'Failed to start app' });
  }
});

// Stop app
app.post('/api/apps/:id/stop', async (req, res) => {
  try {
    const { id } = req.params;
    
    // Get app details
    const appResult = await pool.query('SELECT * FROM apps WHERE id = $1', [id]);
    if (appResult.rows.length === 0) {
      return res.status(404).json({ error: 'App not found' });
    }
    
    const app = appResult.rows[0];
    const containerName = app.config?.container_name || app.name;
    
    // Stop container using Docker API
    const container = docker.getContainer(containerName);
    await container.stop();
    
    // Update app status
    await pool.query(
      'UPDATE apps SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2',
      ['stopped', id]
    );
    
    // Log action
    await pool.query(`
      INSERT INTO app_logs (app_id, level, message, source, timestamp)
      VALUES ($1, 'info', 'App stopped via API', 'api-server', CURRENT_TIMESTAMP)
    `, [id]);
    
    // Publish event
    await redis.publish('app-events', JSON.stringify({
      type: 'app_stopped',
      app_id: id,
      app_name: app.name,
      timestamp: new Date().toISOString()
    }));
    
    res.json({ message: 'App stopped successfully' });
  } catch (error) {
    console.error('Error stopping app:', error);
    res.status(500).json({ error: 'Failed to stop app' });
  }
});

// Get app logs
app.get('/api/apps/:id/logs', async (req, res) => {
  try {
    const { id } = req.params;
    const limit = parseInt(req.query.limit) || 100;
    const offset = parseInt(req.query.offset) || 0;
    
    const result = await pool.query(`
      SELECT * FROM app_logs 
      WHERE app_id = $1 
      ORDER BY timestamp DESC 
      LIMIT $2 OFFSET $3
    `, [id, limit, offset]);
    
    res.json(result.rows);
  } catch (error) {
    console.error('Error fetching logs:', error);
    res.status(500).json({ error: 'Failed to fetch logs' });
  }
});

// Get app metrics
app.get('/api/apps/:id/metrics', async (req, res) => {
  try {
    const { id } = req.params;
    const hours = parseInt(req.query.hours) || 24;
    
    const result = await pool.query(`
      SELECT * FROM app_status 
      WHERE app_id = $1 
        AND timestamp > NOW() - INTERVAL '${hours} hours'
      ORDER BY timestamp DESC
    `, [id]);
    
    res.json(result.rows);
  } catch (error) {
    console.error('Error fetching metrics:', error);
    res.status(500).json({ error: 'Failed to fetch metrics' });
  }
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error(err.stack);
  res.status(500).json({ error: 'Internal server error' });
});

// 404 handler
app.use('*', (req, res) => {
  res.status(404).json({ error: 'Endpoint not found' });
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('Shutting down gracefully...');
  await pool.end();
  await redis.quit();
  process.exit(0);
});

// Start server
async function startServer() {
  await initializeConnections();
  
  app.listen(port, config.server.host, () => {
    console.log(`🚀 App Monitor API server running on ${config.server.host}:${port}`);
    console.log(`📊 Health check: http://${config.server.host}:${port}/health`);
  });
}

startServer().catch(console.error);