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
const cron = require('node-cron');

// Load configuration with template substitution
const configPath = path.join(__dirname, 'orchestrator-config.json');
const capabilitiesPath = path.join(__dirname, 'model-capabilities.json');
const resourceUrlsPath = path.join(__dirname, 'resource-urls.json');

let config, modelCapabilities, resourceUrls;
try {
  // Load and process templates in configuration
  const configRaw = fs.readFileSync(configPath, 'utf8');
  const capabilitiesRaw = fs.readFileSync(capabilitiesPath, 'utf8');
  const resourceUrlsRaw = fs.readFileSync(resourceUrlsPath, 'utf8');
  
  // Process templates ({{SECRET_NAME}} substitution would happen here in real deployment)
  // For now, we'll handle SERVICE_PORT via environment variable
  const processedConfig = configRaw.replace(/"\{\{SERVICE_PORT\}\}"/g, process.env.SERVICE_PORT || process.env.PORT || '8082');
  
  config = JSON.parse(processedConfig);
  modelCapabilities = JSON.parse(capabilitiesRaw);
  resourceUrls = JSON.parse(resourceUrlsRaw);
} catch (error) {
  console.error('Failed to load configuration:', error.message);
  if (error.message.includes('resource-urls.json')) {
    console.error('Hint: Run "bash initialization/scripts/create-resource-urls.sh" to generate resource URLs');
  }
  process.exit(1);
}

const app = express();
const port = config.server.port || process.env.SERVICE_PORT || process.env.PORT || '8082';

// Security middleware
app.use(helmet());
app.use(cors({
  origin: config.server.cors.origins,
  credentials: true
}));

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: config.server.rateLimit.max || 1000 // Higher limit for AI orchestration
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

// Global state for model metrics and system resources
let systemMetrics = {
  memory: { available: 0, free: 0, total: 0 },
  cpu: { usage: 0 },
  models: {}
};

let modelHealth = {};

// Initialize connections
async function initializeConnections() {
  try {
    await pool.query('SELECT NOW()');
    console.log('✅ PostgreSQL connected');
    
    await redis.connect();
    console.log('✅ Redis connected');
    
    await docker.ping();
    console.log('✅ Docker connected');
    
    // Initialize database tables
    await initializeTables();
    
    // Start background monitoring
    startResourceMonitoring();
    startModelHealthMonitoring();
    
  } catch (error) {
    console.error('❌ Failed to initialize connections:', error.message);
    process.exit(1);
  }
}

// Initialize database tables for orchestrator
async function initializeTables() {
  const createTables = `
    CREATE TABLE IF NOT EXISTS model_metrics (
      id SERIAL PRIMARY KEY,
      model_name VARCHAR(255) NOT NULL,
      request_count INTEGER DEFAULT 0,
      success_count INTEGER DEFAULT 0,
      error_count INTEGER DEFAULT 0,
      avg_response_time_ms FLOAT DEFAULT 0,
      current_load FLOAT DEFAULT 0,
      memory_usage_mb FLOAT DEFAULT 0,
      last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS orchestrator_requests (
      id SERIAL PRIMARY KEY,
      request_id VARCHAR(255) UNIQUE NOT NULL,
      task_type VARCHAR(100) NOT NULL,
      selected_model VARCHAR(255) NOT NULL,
      fallback_used BOOLEAN DEFAULT FALSE,
      response_time_ms INTEGER,
      success BOOLEAN DEFAULT TRUE,
      error_message TEXT,
      resource_pressure FLOAT,
      cost_estimate FLOAT,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS system_resources (
      id SERIAL PRIMARY KEY,
      memory_available_gb FLOAT,
      memory_free_gb FLOAT,
      memory_total_gb FLOAT,
      cpu_usage_percent FLOAT,
      swap_used_percent FLOAT,
      recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_model_metrics_name ON model_metrics(model_name);
    CREATE INDEX IF NOT EXISTS idx_orchestrator_requests_model ON orchestrator_requests(selected_model);
    CREATE INDEX IF NOT EXISTS idx_orchestrator_requests_created ON orchestrator_requests(created_at);
    CREATE INDEX IF NOT EXISTS idx_system_resources_recorded ON system_resources(recorded_at);
  `;
  
  await pool.query(createTables);
  console.log('✅ Database tables initialized');
}

// Get current system resources
async function getCurrentResources() {
  try {
    // Get memory info
    const memInfo = fs.readFileSync('/proc/meminfo', 'utf8');
    const memTotal = parseInt(memInfo.match(/MemTotal:\s+(\d+)/)[1]) * 1024;
    const memFree = parseInt(memInfo.match(/MemFree:\s+(\d+)/)[1]) * 1024;
    const memAvailable = parseInt(memInfo.match(/MemAvailable:\s+(\d+)/)[1]) * 1024;
    
    // Get CPU info
    const loadAvg = fs.readFileSync('/proc/loadavg', 'utf8');
    const cpuLoad = parseFloat(loadAvg.split(' ')[0]);
    
    systemMetrics = {
      memory: {
        total: memTotal,
        free: memFree,
        available: memAvailable,
        total_gb: memTotal / 1024 / 1024 / 1024,
        free_gb: memFree / 1024 / 1024 / 1024,
        available_gb: memAvailable / 1024 / 1024 / 1024
      },
      cpu: {
        load: cpuLoad,
        usage: Math.min(cpuLoad * 20, 100) // Rough approximation
      }
    };
    
    // Store in database
    await pool.query(`
      INSERT INTO system_resources (memory_available_gb, memory_free_gb, memory_total_gb, cpu_usage_percent)
      VALUES ($1, $2, $3, $4)
    `, [
      systemMetrics.memory.available_gb,
      systemMetrics.memory.free_gb, 
      systemMetrics.memory.total_gb,
      systemMetrics.cpu.usage
    ]);
    
  } catch (error) {
    console.error('Error getting system resources:', error.message);
  }
}

// Check health of all available models
async function checkModelHealth() {
  try {
    const response = await fetch(`${resourceUrls.resources.ai.ollama.api_base}/tags`);
    const data = await response.json();
    
    if (data.models) {
      for (const model of data.models) {
        const modelName = model.name;
        
        // Basic health check - just verify model is listed
        modelHealth[modelName] = {
          healthy: true,
          last_check: new Date().toISOString(),
          size_gb: model.size ? model.size / (1024 * 1024 * 1024) : 0
        };
        
        // Update model metrics in database
        await pool.query(`
          INSERT INTO model_metrics (model_name, last_used)
          VALUES ($1, CURRENT_TIMESTAMP)
          ON CONFLICT (model_name) DO UPDATE SET last_used = CURRENT_TIMESTAMP
        `, [modelName]);
      }
    }
  } catch (error) {
    console.error('Error checking model health:', error.message);
  }
}

// Start background resource monitoring
function startResourceMonitoring() {
  // Update every 5 seconds
  setInterval(getCurrentResources, 5000);
  
  // Initial run
  getCurrentResources();
  
  console.log('✅ Resource monitoring started');
}

// Start background model health monitoring  
function startModelHealthMonitoring() {
  // Update every 30 seconds
  setInterval(checkModelHealth, 30000);
  
  // Initial run
  checkModelHealth();
  
  console.log('✅ Model health monitoring started');
}

// Calculate memory pressure (0-1 scale)
function calculateMemoryPressure() {
  if (!systemMetrics.memory.total_gb) return 0;
  return 1 - (systemMetrics.memory.available_gb / systemMetrics.memory.total_gb);
}

// Intelligent model selection algorithm
function selectOptimalModel(taskType, requirements = {}) {
  const {
    complexity = 'moderate',
    priority = 'normal',
    maxTokens = 2048,
    costLimit = null,
    qualityRequirement = 'balanced'
  } = requirements;
  
  // Get models capable of this task type
  const capableModels = Object.entries(modelCapabilities.models)
    .filter(([name, model]) => model.capabilities.includes(taskType))
    .map(([name, model]) => ({ name, ...model }));
  
  if (capableModels.length === 0) {
    return null;
  }
  
  // Filter by resource availability
  const memoryPressure = calculateMemoryPressure();
  const resourceViableModels = capableModels.filter(model => {
    const ramNeeded = model.ram_required_gb + 1.5; // Safety buffer
    return systemMetrics.memory.available_gb > ramNeeded;
  });
  
  if (resourceViableModels.length === 0) {
    // Fallback to smallest model if under memory pressure
    const smallest = capableModels.reduce((prev, curr) => 
      curr.ram_required_gb < prev.ram_required_gb ? curr : prev
    );
    return smallest.name;
  }
  
  // Score models based on requirements
  const scoredModels = resourceViableModels.map(model => {
    let score = 0;
    
    // Speed scoring (higher is better)
    const speedScore = model.speed === 'fast' ? 3 : model.speed === 'moderate' ? 2 : 1;
    
    // Cost scoring (lower cost is better)
    const costScore = costLimit ? Math.max(0, 3 - (model.cost_per_1k_tokens / costLimit * 3)) : 2;
    
    // Quality scoring based on model size (rough approximation)
    const qualityScore = Math.min(3, model.ram_required_gb / 2);
    
    // Resource efficiency scoring (lower RAM usage is better under pressure)
    const resourceScore = memoryPressure > 0.7 ? 
      Math.max(0, 3 - (model.ram_required_gb / 10)) : 2;
    
    // Weight scores based on requirements
    const weights = {
      speed: priority === 'critical' ? 0.6 : priority === 'high' ? 0.4 : 0.2,
      cost: costLimit ? 0.4 : 0.1,
      quality: complexity === 'complex' ? 0.5 : qualityRequirement === 'best' ? 0.4 : 0.2,
      resource: memoryPressure > 0.5 ? 0.3 : 0.1
    };
    
    score = (speedScore * weights.speed) + 
            (costScore * weights.cost) + 
            (qualityScore * weights.quality) + 
            (resourceScore * weights.resource);
    
    return { model, score };
  });
  
  // Sort by score and return best model
  scoredModels.sort((a, b) => b.score - a.score);
  return scoredModels[0].model.name;
}

// Health check endpoint
app.get('/health', async (req, res) => {
  try {
    const dbCheck = await pool.query('SELECT 1');
    const redisCheck = await redis.ping();
    const dockerCheck = await docker.ping();
    
    const memoryPressure = calculateMemoryPressure();
    const availableModels = Object.keys(modelHealth).length;
    
    res.json({
      status: 'healthy',
      timestamp: new Date().toISOString(),
      services: {
        database: dbCheck.rowCount === 1 ? 'ok' : 'error',
        redis: redisCheck === 'PONG' ? 'ok' : 'error',
        docker: dockerCheck ? 'ok' : 'error'
      },
      system: {
        memory_pressure: memoryPressure,
        available_models: availableModels,
        memory_available_gb: systemMetrics.memory.available_gb,
        cpu_usage_percent: systemMetrics.cpu.usage
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
    service: 'ai-model-orchestra-controller',
    version: '1.0.0',
    timestamp: new Date().toISOString(),
    capabilities: ['model-selection', 'load-balancing', 'resource-monitoring']
  });
});

// Model selection endpoint
app.post('/api/ai/select-model', async (req, res) => {
  try {
    const { taskType, requirements = {} } = req.body;
    
    if (!taskType) {
      return res.status(400).json({ error: 'taskType is required' });
    }
    
    const selectedModel = selectOptimalModel(taskType, requirements);
    
    if (!selectedModel) {
      return res.status(404).json({ 
        error: 'No suitable model found',
        taskType,
        availableModels: Object.keys(modelCapabilities.models)
      });
    }
    
    // Log the selection
    const requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    await pool.query(`
      INSERT INTO orchestrator_requests (request_id, task_type, selected_model, resource_pressure)
      VALUES ($1, $2, $3, $4)
    `, [requestId, taskType, selectedModel, calculateMemoryPressure()]);
    
    res.json({
      requestId,
      selectedModel,
      taskType,
      systemMetrics: {
        memoryPressure: calculateMemoryPressure(),
        availableMemoryGb: systemMetrics.memory.available_gb,
        cpuUsage: systemMetrics.cpu.usage
      },
      modelInfo: modelCapabilities.models[selectedModel] || null
    });
    
  } catch (error) {
    console.error('Error selecting model:', error);
    res.status(500).json({ error: 'Model selection failed' });
  }
});

// Route AI request endpoint (full orchestration)
app.post('/api/ai/route-request', async (req, res) => {
  const startTime = Date.now();
  let requestId = null;
  
  try {
    const { 
      taskType, 
      prompt, 
      requirements = {},
      retryAttempts = 3 
    } = req.body;
    
    if (!taskType || !prompt) {
      return res.status(400).json({ error: 'taskType and prompt are required' });
    }
    
    requestId = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    
    // Select optimal model
    const selectedModel = selectOptimalModel(taskType, requirements);
    
    if (!selectedModel) {
      await pool.query(`
        INSERT INTO orchestrator_requests (request_id, task_type, selected_model, success, error_message)
        VALUES ($1, $2, 'none', FALSE, 'No suitable model found')
      `, [requestId, taskType]);
      
      return res.status(404).json({ 
        error: 'No suitable model found',
        requestId
      });
    }
    
    // Route to Ollama with selected model
    const ollamaResponse = await fetch(`${resourceUrls.resources.ai.ollama.api_base}/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: selectedModel,
        prompt: prompt,
        stream: false,
        options: {
          num_predict: requirements.maxTokens || 2048
        }
      })
    });
    
    const responseTime = Date.now() - startTime;
    
    if (!ollamaResponse.ok) {
      throw new Error(`Ollama request failed: ${ollamaResponse.status}`);
    }
    
    const ollamaData = await ollamaResponse.json();
    
    // Log successful request
    await pool.query(`
      INSERT INTO orchestrator_requests (request_id, task_type, selected_model, response_time_ms, success, resource_pressure)
      VALUES ($1, $2, $3, $4, TRUE, $5)
    `, [requestId, taskType, selectedModel, responseTime, calculateMemoryPressure()]);
    
    // Update model metrics
    await pool.query(`
      UPDATE model_metrics 
      SET request_count = request_count + 1,
          success_count = success_count + 1,
          avg_response_time_ms = (avg_response_time_ms * request_count + $2) / (request_count + 1),
          last_used = CURRENT_TIMESTAMP
      WHERE model_name = $1
    `, [selectedModel, responseTime]);
    
    res.json({
      requestId,
      selectedModel,
      response: ollamaData.response,
      metrics: {
        responseTimeMs: responseTime,
        memoryPressure: calculateMemoryPressure(),
        modelUsed: selectedModel,
        fallbackUsed: false
      }
    });
    
  } catch (error) {
    const responseTime = Date.now() - startTime;
    console.error('Error routing AI request:', error);
    
    if (requestId) {
      await pool.query(`
        UPDATE orchestrator_requests 
        SET success = FALSE, error_message = $1, response_time_ms = $2
        WHERE request_id = $3
      `, [error.message, responseTime, requestId]);
    }
    
    res.status(500).json({ 
      error: 'Request routing failed',
      requestId,
      message: error.message
    });
  }
});

// Get model status
app.get('/api/ai/models/status', async (req, res) => {
  try {
    const modelMetrics = await pool.query(`
      SELECT * FROM model_metrics ORDER BY last_used DESC
    `);
    
    res.json({
      models: modelMetrics.rows,
      systemHealth: modelHealth,
      totalModels: Object.keys(modelCapabilities.models).length,
      healthyModels: Object.keys(modelHealth).length
    });
  } catch (error) {
    console.error('Error getting model status:', error);
    res.status(500).json({ error: 'Failed to get model status' });
  }
});

// Get resource metrics
app.get('/api/ai/resources/metrics', async (req, res) => {
  try {
    const hours = parseInt(req.query.hours) || 1;
    
    const resourceHistory = await pool.query(`
      SELECT * FROM system_resources 
      WHERE recorded_at > NOW() - INTERVAL '${hours} hours'
      ORDER BY recorded_at DESC
    `);
    
    const requestStats = await pool.query(`
      SELECT 
        DATE_TRUNC('minute', created_at) as minute,
        COUNT(*) as request_count,
        AVG(response_time_ms) as avg_response_time,
        COUNT(*) FILTER (WHERE success = TRUE) as success_count,
        COUNT(*) FILTER (WHERE success = FALSE) as error_count
      FROM orchestrator_requests 
      WHERE created_at > NOW() - INTERVAL '${hours} hours'
      GROUP BY DATE_TRUNC('minute', created_at)
      ORDER BY minute DESC
    `);
    
    res.json({
      current: systemMetrics,
      history: resourceHistory.rows,
      requests: requestStats.rows,
      memoryPressure: calculateMemoryPressure()
    });
  } catch (error) {
    console.error('Error getting resource metrics:', error);
    res.status(500).json({ error: 'Failed to get resource metrics' });
  }
});

// Dashboard endpoint
app.get('/dashboard', (req, res) => {
  res.sendFile(path.join(__dirname, '../../ui/dashboard.html'));
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

// Cleanup old data (run daily)
cron.schedule('0 2 * * *', async () => {
  try {
    console.log('🧹 Cleaning up old data...');
    
    // Clean up old resource metrics (keep 7 days)
    await pool.query(`
      DELETE FROM system_resources 
      WHERE recorded_at < NOW() - INTERVAL '7 days'
    `);
    
    // Clean up old orchestrator requests (keep 30 days)
    await pool.query(`
      DELETE FROM orchestrator_requests 
      WHERE created_at < NOW() - INTERVAL '30 days'
    `);
    
    console.log('✅ Data cleanup completed');
  } catch (error) {
    console.error('❌ Data cleanup failed:', error.message);
  }
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
    console.log(`🚀 AI Model Orchestra Controller running on ${config.server.host}:${port}`);
    console.log(`📊 Health check: http://${config.server.host}:${port}/health`);
    console.log(`🎛️  Dashboard: http://${config.server.host}:${port}/dashboard`);
    console.log(`🔀 Model selection: POST http://${config.server.host}:${port}/api/ai/select-model`);
    console.log(`🚦 Request routing: POST http://${config.server.host}:${port}/api/ai/route-request`);
  });
}

startServer().catch(console.error);