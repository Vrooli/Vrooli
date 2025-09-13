const express = require('express');
const path = require('path');
const fs = require('fs');
const http = require('http');
const app = express();

const PORT = process.env.PORT || process.env.UI_PORT || 8420;
const API_URL = process.env.API_MANAGER_URL || 'http://localhost:8421';

let startTime = Date.now();

// Serve static files
app.use(express.static(__dirname));

// Comprehensive health check endpoint
app.get('/health', async (req, res) => {
  const health = {
    status: 'healthy',
    service: 'api-manager-ui',
    timestamp: new Date().toISOString(),
    readiness: true,
    api_connectivity: {
      connected: false,
      api_url: API_URL,
      last_check: new Date().toISOString(),
      latency_ms: null
    },
    static_files: {
      available: true,
      files: {}
    },
    dashboard_features: {
      scenario_management: true,
      vulnerability_scanner: true,
      endpoint_discovery: true,
      security_audit: true,
      lifecycle_protection_validation: true
    },
    metrics: {
      uptime_seconds: Math.floor((Date.now() - startTime) / 1000),
      node_version: process.version,
      memory_usage: process.memoryUsage()
    }
  };
  
  // Check API connectivity
  const apiHealthCheck = await checkAPIConnectivity();
  health.api_connectivity = { ...health.api_connectivity, ...apiHealthCheck };
  
  if (!apiHealthCheck.connected) {
    health.status = 'degraded';
    health.readiness = false;
    if (apiHealthCheck.error) {
      health.errors = health.errors || [];
      health.errors.push(apiHealthCheck.error);
    }
  }
  
  // Check static files
  const staticFiles = checkStaticFiles();
  health.static_files = staticFiles;
  
  if (!staticFiles.available) {
    health.status = 'degraded';
    if (staticFiles.error) {
      health.errors = health.errors || [];
      health.errors.push(staticFiles.error);
    }
  }
  
  // Check API proxy functionality
  const proxyHealth = await checkProxyFunctionality();
  health.proxy_status = proxyHealth;
  
  if (!proxyHealth.functional) {
    if (health.status === 'healthy') {
      health.status = 'degraded';
    }
    if (proxyHealth.error) {
      health.errors = health.errors || [];
      health.errors.push(proxyHealth.error);
    }
  }
  
  const statusCode = health.status === 'healthy' ? 200 : 503;
  res.status(statusCode).json(health);
});

// Helper function to check API connectivity
async function checkAPIConnectivity() {
  const health = {
    connected: false,
    api_url: API_URL,
    last_check: new Date().toISOString(),
    latency_ms: null
  };
  
  const startTime = Date.now();
  
  return new Promise((resolve) => {
    const url = new URL('/api/v1/health', API_URL);
    const options = {
      hostname: url.hostname,
      port: url.port || (url.protocol === 'https:' ? 443 : 80),
      path: url.pathname,
      method: 'GET',
      timeout: 5000
    };
    
    const req = http.request(options, (res) => {
      const endTime = Date.now();
      health.latency_ms = endTime - startTime;
      
      if (res.statusCode === 200) {
        health.connected = true;
        
        let body = '';
        res.on('data', chunk => body += chunk);
        res.on('end', () => {
          try {
            const apiHealth = JSON.parse(body);
            health.api_status = apiHealth.status;
            health.api_dependencies = apiHealth.dependencies;
            
            // Check for critical vulnerabilities from API
            if (apiHealth.api_manager_stats) {
              health.open_vulnerabilities = apiHealth.api_manager_stats.open_vulnerabilities;
              health.active_scenarios = apiHealth.api_manager_stats.active_scenarios;
            }
          } catch (e) {
            health.api_status = 'unknown';
          }
          resolve(health);
        });
      } else {
        health.error = {
          code: 'API_UNHEALTHY',
          message: `API returned status ${res.statusCode}`,
          category: 'resource',
          retryable: true
        };
        resolve(health);
      }
    });
    
    req.on('error', (err) => {
      health.error = {
        code: 'API_CONNECTION_FAILED',
        message: `Failed to connect to API: ${err.message}`,
        category: 'network',
        retryable: true
      };
      resolve(health);
    });
    
    req.on('timeout', () => {
      health.error = {
        code: 'API_CONNECTION_TIMEOUT',
        message: 'API health check timed out after 5 seconds',
        category: 'network',
        retryable: true
      };
      req.destroy();
      resolve(health);
    });
    
    req.end();
  });
}

// Helper function to check static files
function checkStaticFiles() {
  const health = {
    available: true,
    files: {}
  };
  
  const requiredFiles = {
    'index.html': path.join(__dirname, 'index.html'),
    'package.json': path.join(__dirname, 'package.json')
  };
  
  for (const [name, filePath] of Object.entries(requiredFiles)) {
    try {
      if (fs.existsSync(filePath)) {
        const stats = fs.statSync(filePath);
        health.files[name] = {
          exists: true,
          size: stats.size,
          modified: stats.mtime.toISOString()
        };
      } else {
        health.available = false;
        health.files[name] = { exists: false };
        health.error = {
          code: 'STATIC_FILE_MISSING',
          message: `Required file ${name} is missing`,
          category: 'configuration',
          retryable: false
        };
      }
    } catch (err) {
      health.available = false;
      health.files[name] = { 
        exists: false, 
        error: err.message 
      };
    }
  }
  
  return health;
}

// Helper function to check proxy functionality
async function checkProxyFunctionality() {
  const health = {
    functional: true,
    test_endpoint: '/api/v1/system/status'
  };
  
  try {
    // Check if node-fetch is available
    const fetch = (await import('node-fetch')).default;
    if (!fetch) {
      health.functional = false;
      health.error = {
        code: 'PROXY_MODULE_MISSING',
        message: 'node-fetch module not available for proxy functionality',
        category: 'configuration',
        retryable: false
      };
    }
  } catch (err) {
    health.functional = false;
    health.error = {
      code: 'PROXY_MODULE_ERROR',
      message: `Failed to load proxy module: ${err.message}`,
      category: 'configuration',
      retryable: false
    };
  }
  
  return health;
}

// API proxy to avoid CORS issues
app.get('/api/*', async (req, res) => {
  try {
    const fetch = (await import('node-fetch')).default;
    const apiPath = req.path.replace('/api', '/api/v1');
    const response = await fetch(`${API_URL}${apiPath}`);
    const data = await response.json();
    
    res.status(response.status).json(data);
  } catch (error) {
    res.status(500).json({ 
      error: 'API request failed', 
      message: error.message 
    });
  }
});

// Serve main page
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
  console.log(`API Manager UI running on http://localhost:${PORT}`);
  console.log(`Connected to API at ${API_URL}`);
});