const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const morgan = require('morgan');
const path = require('path');
const WebSocket = require('ws');
const http = require('http');

const app = express();

// Configure API base URL from environment - NO HARD-CODED FALLBACKS
const apiPort = process.env.API_PORT;
if (!apiPort) {
  console.error('❌ API_PORT environment variable is required');
  process.exit(1);
}
const apiBaseUrl = process.env.TEST_GENIE_API_URL || `http://localhost:${apiPort}`;

async function proxyToApi(req, res, upstreamPath, options = {}) {
  const {
    method,
    headers: extraHeaders = {},
    body,
    autoSend = true,
    collectResponse = true,
    timeoutMs,
    sanitizeHeaders = true
  } = options;

  const resolvedMethod = method || (req ? req.method : 'GET');
  const baseForJoin = apiBaseUrl.endsWith('/') ? apiBaseUrl : `${apiBaseUrl}/`;
  const fallbackPath = req ? (req.originalUrl || req.url || '/') : '/';
  const targetPath = upstreamPath || fallbackPath;

  let targetUrl;
  if (/^https?:\/\//i.test(targetPath)) {
    targetUrl = new URL(targetPath);
  } else {
    const normalizedPath = targetPath.startsWith('/') ? targetPath : `/${targetPath}`;
    targetUrl = new URL(normalizedPath, baseForJoin);
  }

  const sourceHeaders = {
    ...(req && req.headers ? req.headers : {}),
    ...extraHeaders
  };

  const upstreamHeaders = {};
  for (const [key, value] of Object.entries(sourceHeaders)) {
    if (value === undefined) {
      continue;
    }
    const lowerKey = key.toLowerCase();
    if (lowerKey === 'host' || lowerKey === 'content-length') {
      continue;
    }
    upstreamHeaders[key] = value;
  }

  if (req && req.headers && req.headers.host && !upstreamHeaders['x-forwarded-host']) {
    upstreamHeaders['x-forwarded-host'] = req.headers.host;
  }

  let payload = body;

  const methodAllowsBody = resolvedMethod !== 'GET' && resolvedMethod !== 'HEAD';
  if (payload === undefined && req && methodAllowsBody) {
    if (req.rawBody && req.rawBody.length) {
      payload = req.rawBody;
    } else if (typeof req.body === 'string' || Buffer.isBuffer(req.body)) {
      payload = req.body;
    } else if (req.body && Object.keys(req.body).length > 0) {
      payload = JSON.stringify(req.body);
      if (!('content-type' in upstreamHeaders) && !('Content-Type' in upstreamHeaders)) {
        upstreamHeaders['Content-Type'] = 'application/json';
      }
    }
  }

  const controller = timeoutMs ? new AbortController() : null;
  const fetchImpl = typeof fetch === 'function' ? fetch : (await import('node-fetch')).default;

  const fetchOptions = {
    method: resolvedMethod,
    headers: upstreamHeaders,
    redirect: 'manual'
  };

  if (methodAllowsBody && payload !== undefined) {
    fetchOptions.body = payload;
  }

  let timeoutHandle;
  if (controller) {
    timeoutHandle = setTimeout(() => controller.abort(), timeoutMs);
    fetchOptions.signal = controller.signal;
  }

  let response;
  try {
    response = await fetchImpl(targetUrl, fetchOptions);
  } catch (error) {
    if (timeoutHandle) {
      clearTimeout(timeoutHandle);
    }
    if (res && autoSend && !res.headersSent) {
      res.status(error.name === 'AbortError' ? 504 : 502).json({
        error: 'API request failed',
        details: error.message,
        target: targetUrl.toString()
      });
      return {
        status: error.name === 'AbortError' ? 504 : 502,
        headers: {},
        bodyBuffer: Buffer.alloc(0),
        bodyText: '',
        error,
        targetUrl: targetUrl.toString()
      };
    }
    throw error;
  } finally {
    if (timeoutHandle) {
      clearTimeout(timeoutHandle);
    }
  }

  const headersMap = {};
  response.headers.forEach((value, key) => {
    if (sanitizeHeaders && key.toLowerCase() === 'content-encoding') {
      return;
    }
    headersMap[key] = value;
  });

  const rawBuffer = await response.arrayBuffer();
  const bodyBuffer = Buffer.from(rawBuffer);

  if (res && autoSend && !res.headersSent) {
    res.status(response.status);
    Object.entries(headersMap).forEach(([key, value]) => {
      res.setHeader(key, value);
    });
    if (bodyBuffer.length > 0) {
      res.send(bodyBuffer);
    } else {
      res.end();
    }
  }

  const bodyText = collectResponse && bodyBuffer.length > 0 ? bodyBuffer.toString('utf8') : '';

  return {
    status: response.status,
    headers: headersMap,
    bodyBuffer,
    bodyText,
    response,
    targetUrl: targetUrl.toString()
  };
}

// Security and performance middleware
const localFrameAncestors = [
  "'self'",
  'http://localhost:*',
  'http://127.0.0.1:*',
  'http://[::1]:*'
];

const extraFrameAncestors = (process.env.FRAME_ANCESTORS || '')
  .split(/\s+/)
  .filter(Boolean);

// Configure allowed origins for CORS
const allowedOrigins = [
  'http://localhost:*',
  'http://127.0.0.1:*',
  'https://test-genie.itsagitime.com'
];

app.use(helmet({
  frameguard: false,
  contentSecurityPolicy: {
    useDefaults: true,
    directives: {
      defaultSrc: ["'self'", "https://test-genie.itsagitime.com"],
      styleSrc: ["'self'", "'unsafe-inline'", "https://fonts.googleapis.com"],
      fontSrc: ["'self'", "https://fonts.gstatic.com"],
      scriptSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'", apiBaseUrl, "ws:", "wss:", "https://test-genie.itsagitime.com"],
      frameAncestors: [...localFrameAncestors, ...extraFrameAncestors]
    }
  }
}));

app.use(cors({
  origin: (origin, callback) => {
    // Allow requests with no origin (like mobile apps or curl)
    if (!origin) return callback(null, true);

    // Check if origin matches any allowed pattern
    const isAllowed = allowedOrigins.some(pattern => {
      if (pattern.includes('*')) {
        const regex = new RegExp('^' + pattern.replace(/\*/g, '.*') + '$');
        return regex.test(origin);
      }
      return pattern === origin;
    });

    if (isAllowed) {
      callback(null, true);
    } else {
      callback(new Error('Not allowed by CORS'));
    }
  },
  credentials: true
}));
app.use(compression());
app.use(morgan('combined'));
app.use(express.json({
  limit: '2mb',
  verify: (req, res, buf) => {
    if (buf && buf.length) {
      req.rawBody = Buffer.from(buf);
    }
  }
}));
app.use(express.static('.'));

// Health check endpoint
app.get('/health', async (req, res) => {
  const health = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    service: 'test-genie-ui',
    api_connectivity: {
      connected: false,
      api_url: apiBaseUrl,
      last_check: new Date().toISOString(),
      error: null,
      latency_ms: null
    }
  };

  const startTime = Date.now();
  try {
    const result = await proxyToApi(null, null, '/health', {
      method: 'GET',
      headers: {
        'User-Agent': 'test-genie-ui-health-check'
      },
      timeoutMs: 5000,
      autoSend: false,
      collectResponse: true
    });

    const latency = Date.now() - startTime;
    if (result.status >= 200 && result.status < 300) {
      health.api_connectivity.connected = true;
      health.api_connectivity.latency_ms = latency;
    } else {
      health.api_connectivity.error = `API returned ${result.status}`;
      health.status = 'degraded';
    }
  } catch (error) {
    health.api_connectivity.error = error.name === 'AbortError'
      ? 'API connection timeout (>5s)'
      : error.message.includes('ECONNREFUSED')
        ? `API not reachable at ${apiBaseUrl}`
        : error.message;
    health.status = 'degraded';
  }

  const httpStatus = health.status === 'healthy' ? 200 : 503;
  res.status(httpStatus).json(health);
});

// Enhanced API proxy with WebSocket notifications (must live before SPA catch-all)
app.use('/api', async (req, res) => {
  try {
    const result = await proxyToApi(req, res, req.originalUrl || req.url, {
      autoSend: false,
      collectResponse: true
    });

    try {
      const jsonData = result.bodyText ? JSON.parse(result.bodyText) : null;

      if (jsonData && req.path.includes('/test-suite/') && req.path.includes('/execute') && req.method === 'POST') {
        broadcastToSubscribers('executions', 'execution_started', {
          execution_id: jsonData.execution_id,
          timestamp: new Date().toISOString(),
        });
      }

      if (jsonData && req.path.includes('/test-execution') && jsonData.status === 'completed') {
        broadcastToSubscribers('executions', 'execution_update', {
          execution_id: jsonData.execution_id,
          status: jsonData.status,
          timestamp: new Date().toISOString(),
        });
      }
    } catch (e) {
      // Ignore JSON parsing failures for non-JSON responses
    }

    if (!res.headersSent) {
      res.status(result.status);
      Object.entries(result.headers).forEach(([key, value]) => {
        res.setHeader(key, value);
      });

      if (result.bodyBuffer.length > 0) {
        res.send(result.bodyBuffer);
      } else {
        res.end();
      }
    }
  } catch (error) {
    console.error('API proxy error:', error);
    if (!res.headersSent) {
      res.status(500).json({ error: 'API request failed', details: error.message });
    }
  }
});

// Serve the main application
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

// Handle SPA routing
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

// Configure UI port from environment - NO HARD-CODED FALLBACKS
const port = process.env.UI_PORT;
if (!port) {
  console.error('❌ UI_PORT environment variable is required');
  process.exit(1);
}

// Create HTTP server
const server = http.createServer(app);

// Create WebSocket server
const wss = new WebSocket.Server({
  server,
  path: '/ws',
  clientTracking: true,
  verifyClient: (info) => {
    const origin = info.origin || info.req.headers.origin;
    if (!origin) return true; // Allow connections without origin

    return allowedOrigins.some(pattern => {
      if (pattern.includes('*')) {
        const regex = new RegExp('^' + pattern.replace(/\*/g, '.*') + '$');
        return regex.test(origin);
      }
      return pattern === origin;
    });
  }
});

// WebSocket connection handling
wss.on('connection', (ws, req) => {
  console.log(`WebSocket client connected from ${req.socket.remoteAddress}`);
  
  // Send welcome message
  ws.send(JSON.stringify({
    type: 'connection',
    payload: {
      message: 'Connected to Test Genie WebSocket server',
      timestamp: new Date().toISOString()
    }
  }));
  
  // Handle client messages
  ws.on('message', (message) => {
    try {
      const data = JSON.parse(message);
      console.log('WebSocket message received:', data);
      
      // Handle different message types
      switch (data.type) {
        case 'subscribe':
          // Handle subscription requests
          ws.subscriptions = data.payload.topics || [];
          ws.send(JSON.stringify({
            type: 'subscribed',
            payload: { topics: ws.subscriptions }
          }));
          break;
          
        case 'ping':
          // Handle ping for keep-alive
          ws.send(JSON.stringify({
            type: 'pong',
            payload: { timestamp: new Date().toISOString() }
          }));
          break;
          
        default:
          console.log('Unknown WebSocket message type:', data.type);
      }
    } catch (error) {
      console.error('WebSocket message parsing error:', error);
    }
  });
  
  // Handle connection close
  ws.on('close', (code, reason) => {
    console.log(`WebSocket client disconnected: ${code} - ${reason}`);
  });
  
  // Handle errors
  ws.on('error', (error) => {
    console.error('WebSocket error:', error);
  });
  
  // Send periodic updates for subscribed topics
  const updateInterval = setInterval(() => {
    if (ws.readyState === WebSocket.OPEN) {
      // Check if client is subscribed to system status
      if (ws.subscriptions && ws.subscriptions.includes('system_status')) {
        // In a real implementation, this would check actual system health
        ws.send(JSON.stringify({
          type: 'system_status',
          payload: { 
            healthy: true,
            timestamp: new Date().toISOString()
          }
        }));
      }
    } else {
      clearInterval(updateInterval);
    }
  }, 30000); // Every 30 seconds
});

// Broadcast message to all connected clients
function broadcastMessage(type, payload) {
  wss.clients.forEach((client) => {
    if (client.readyState === WebSocket.OPEN) {
      client.send(JSON.stringify({ type, payload }));
    }
  });
}

// Broadcast to specific subscribers
function broadcastToSubscribers(topic, type, payload) {
  wss.clients.forEach((client) => {
    if (client.readyState === WebSocket.OPEN && 
        client.subscriptions && 
        client.subscriptions.includes(topic)) {
      client.send(JSON.stringify({ type, payload }));
    }
  });
}

// Start server
server.listen(port, () => {
  console.log(`🧪 Test Genie UI server running on port ${port}`);
  console.log(`📊 Dashboard: http://localhost:${port}`);
  console.log(`🔗 API Proxy: ${apiBaseUrl}`);
  console.log(`🔌 WebSocket: ws://localhost:${port}/ws`);
  console.log(`👥 WebSocket clients: ${wss.clients.size}`);
});

module.exports = {
  proxyToApi,
};
