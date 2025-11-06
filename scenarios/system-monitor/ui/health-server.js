#!/usr/bin/env node

/**
 * Health Check Server for System Monitor UI
 * Runs alongside the Vite dev server to provide health endpoints
 */

const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');
const { URL } = require('url');

const uiPort = Number(process.env.UI_PORT || process.env.PORT || '3000');
const HEALTH_PORT = Number(process.env.HEALTH_PORT || (uiPort + 1));
const API_PORT = (process.env.API_PORT || '').trim();
const UI_PORT = (process.env.UI_PORT || process.env.PORT || '').trim();

const API_PROTOCOL = (process.env.API_PROTOCOL || 'http').trim().replace(/:+$/, '') || 'http';
const API_HOST = (process.env.API_HOST || '127.0.0.1').trim() || '127.0.0.1';
const UI_LISTEN_HOST = (process.env.HEALTH_HOST || process.env.UI_HOST || '0.0.0.0').trim() || '0.0.0.0';
const UI_CHECK_HOST = UI_LISTEN_HOST === '0.0.0.0' ? '127.0.0.1' : UI_LISTEN_HOST;
const RAW_API_BASE = (process.env.API_BASE_URL || process.env.VITE_API_BASE_URL || '').trim();

const toUrl = (value) => {
  if (!value) {
    return undefined;
  }
  try {
    return new URL(value);
  } catch (error) {
    console.warn(`Unable to parse API base URL "${value}":`, error.message || error);
    return undefined;
  }
};

const API_BASE = toUrl(RAW_API_BASE) || (API_PORT ? toUrl(`${API_PROTOCOL}://${API_HOST}:${API_PORT}`) : undefined);

const API_BASE_ORIGIN = API_BASE ? `${API_BASE.protocol}//${API_BASE.host}` : undefined;
const API_HEALTH_PATH = API_BASE ? `${API_BASE.pathname.replace(/\/$/, '') || ''}/health` : '/health';

const selectHttpModule = (url) => (url?.protocol === 'https:' ? https : http);

/**
 * Test API connectivity
 */
async function checkAPIConnectivity() {
  const result = {
    connected: false,
    api_url: API_BASE_ORIGIN || (API_PORT ? `${API_PROTOCOL}://${API_HOST}:${API_PORT}` : undefined),
    last_check: new Date().toISOString(),
    error: null,
    latency_ms: null
  };

  if (!API_BASE && !API_PORT) {
    result.error = {
      code: 'MISSING_CONFIG',
      message: 'API target not configured; set API_BASE_URL or API_PORT',
      category: 'configuration',
      retryable: false
    };
    return result;
  }

  const startTime = Date.now();

  try {
    await new Promise((resolve, reject) => {
      const apiTarget = API_BASE || new URL(`${API_PROTOCOL}://${API_HOST}:${API_PORT}`);
      const options = {
        protocol: apiTarget.protocol,
        hostname: apiTarget.hostname,
        port: apiTarget.port || (apiTarget.protocol === 'https:' ? 443 : 80),
        path: API_HEALTH_PATH,
        method: 'GET',
        timeout: 5000,
        headers: {
          'Accept': 'application/json'
        }
      };

      const transport = selectHttpModule(apiTarget);

      const req = transport.request(options, (res) => {
        const endTime = Date.now();
        result.latency_ms = endTime - startTime;

        if (res.statusCode >= 200 && res.statusCode < 300) {
          result.connected = true;
          result.error = null;
        } else {
          result.connected = false;
          result.error = {
            code: `HTTP_${res.statusCode}`,
            message: `API returned status ${res.statusCode}: ${res.statusMessage}`,
            category: 'network',
            retryable: res.statusCode >= 500 && res.statusCode < 600
          };
        }
        resolve();
      });

      req.on('error', (error) => {
        const endTime = Date.now();
        result.latency_ms = endTime - startTime;
        result.connected = false;

        let errorCode = 'CONNECTION_FAILED';
        let category = 'network';
        let retryable = true;

        if (error.code === 'ECONNREFUSED') {
          errorCode = 'CONNECTION_REFUSED';
        } else if (error.code === 'ENOTFOUND') {
          errorCode = 'HOST_NOT_FOUND';
          category = 'configuration';
        } else if (error.code === 'ETIMEOUT') {
          errorCode = 'TIMEOUT';
        }

        result.error = {
          code: errorCode,
          message: `Failed to connect to API: ${error.message}`,
          category: category,
          retryable: retryable,
          details: {
            error_code: error.code
          }
        };
        resolve();
      });

      req.on('timeout', () => {
        const endTime = Date.now();
        result.latency_ms = endTime - startTime;
        result.connected = false;
        result.error = {
          code: 'TIMEOUT',
          message: 'API health check timed out after 5 seconds',
          category: 'network',
          retryable: true
        };
        req.destroy();
        resolve();
      });

      req.end();
    });
  } catch (error) {
    const endTime = Date.now();
    result.latency_ms = endTime - startTime;
    result.connected = false;
    result.error = {
      code: 'UNEXPECTED_ERROR',
      message: `Unexpected error: ${error.message}`,
      category: 'internal',
      retryable: true
    };
  }

  return result;
}

/**
 * Check Vite dev server status
 */
async function checkViteServer() {
  const result = {
    connected: false,
    error: null
  };

  if (!UI_PORT) {
    result.error = {
      code: 'UI_PORT_MISSING',
      message: 'UI_PORT environment variable not configured',
      category: 'configuration',
      retryable: false
    };
    return result;
  }

  try {
    await new Promise((resolve, reject) => {
      const options = {
        hostname: UI_CHECK_HOST,
        port: UI_PORT || uiPort,
        path: '/',
        method: 'HEAD',
        timeout: 3000
      };

      const req = http.request(options, (res) => {
        if (res.statusCode >= 200 && res.statusCode < 400) {
          result.connected = true;
        } else {
          result.error = {
            code: `HTTP_${res.statusCode}`,
            message: `Vite server returned status ${res.statusCode}`,
            category: 'network',
            retryable: true
          };
        }
        resolve();
      });

      req.on('error', (error) => {
        result.error = {
          code: 'VITE_CONNECTION_FAILED',
          message: `Cannot connect to Vite server: ${error.message}`,
          category: 'network',
          retryable: true
        };
        resolve();
      });

      req.on('timeout', () => {
        result.error = {
          code: 'TIMEOUT',
          message: 'Vite server check timed out',
          category: 'network',
          retryable: true
        };
        req.destroy();
        resolve();
      });

      req.end();
    });
  } catch (error) {
    result.error = {
      code: 'UNEXPECTED_ERROR',
      message: `Unexpected error checking Vite: ${error.message}`,
      category: 'internal',
      retryable: true
    };
  }

  return result;
}

/**
 * Check React app build status
 */
function checkReactBuild() {
  const result = {
    connected: false,
    error: null,
    build_mode: 'development'
  };

  // Check if dist directory exists (production build)
  const distPath = path.join(__dirname, 'dist');
  const hasDist = fs.existsSync(distPath);

  if (hasDist) {
    // Check if dist has content
    try {
      const distContents = fs.readdirSync(distPath);
      if (distContents.length > 0) {
        result.connected = true;
        result.build_mode = 'production';
        return result;
      }
    } catch (err) {
      result.error = {
        code: 'DIST_READ_ERROR',
        message: `Cannot read dist directory: ${err.message}`,
        category: 'resource',
        retryable: false
      };
      return result;
    }
  }

  // Development mode - check source files
  const srcPath = path.join(__dirname, 'src');
  if (!fs.existsSync(srcPath)) {
    result.error = {
      code: 'SOURCE_MISSING',
      message: 'React source directory not found',
      category: 'resource',
      retryable: false
    };
    return result;
  }

  // Check if we have key React files
  const appPath = path.join(srcPath, 'App.tsx');
  const mainPath = path.join(srcPath, 'main.tsx');
  
  if (!fs.existsSync(appPath) && !fs.existsSync(mainPath)) {
    result.error = {
      code: 'REACT_FILES_MISSING',
      message: 'Core React files (App.tsx, main.tsx) not found',
      category: 'resource',
      retryable: false
    };
    return result;
  }

  result.connected = true;
  result.build_mode = 'development';
  return result;
}

/**
 * Health check handler
 */
async function healthHandler(req, res) {
  const healthResponse = {
    status: 'healthy',
    service: 'system-monitor-ui',
    timestamp: new Date().toISOString(),
    readiness: true,
    api_connectivity: await checkAPIConnectivity()
  };

  // Check Vite dev server
  const viteStatus = await checkViteServer();
  healthResponse.vite_server = viteStatus;

  // Check React build
  const buildStatus = checkReactBuild();
  healthResponse.react_build = buildStatus;

  // Determine overall status
  if (!healthResponse.api_connectivity.connected) {
    healthResponse.status = 'unhealthy';
  } else if (!viteStatus.connected || !buildStatus.connected) {
    healthResponse.status = 'degraded';
  }

  res.writeHead(200, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(healthResponse, null, 2));
}

/**
 * Create and start health server
 */
const server = http.createServer(async (req, res) => {
  // Enable CORS
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  if (req.url === '/health' && req.method === 'GET') {
    await healthHandler(req, res);
  } else {
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Not Found', message: 'Only /health endpoint is available' }));
  }
});

server.listen(HEALTH_PORT, UI_LISTEN_HOST, () => {
  const displayHost = UI_LISTEN_HOST === '0.0.0.0' ? '127.0.0.1' : UI_LISTEN_HOST;
  console.log(`🏥 System Monitor UI Health Server running on http://${displayHost}:${HEALTH_PORT}/health`);
  console.log(`📊 Monitoring UI on port ${UI_PORT || uiPort}, API target ${resultTargetLabel()}`);
});

function resultTargetLabel() {
  if (API_BASE_ORIGIN) {
    return `${API_BASE_ORIGIN}${API_HEALTH_PATH}`;
  }
  if (API_PORT) {
    return `${API_PROTOCOL}://${API_HOST}:${API_PORT}${API_HEALTH_PATH}`;
  }
  return 'unconfigured';
}

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('🛑 Health server shutting down...');
  server.close(() => {
    console.log('✅ Health server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('🛑 Health server shutting down...');
  server.close(() => {
    console.log('✅ Health server closed');
    process.exit(0);
  });
});
