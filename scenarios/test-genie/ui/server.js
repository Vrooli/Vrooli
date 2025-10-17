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
app.use(express.json());
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

  // Check API connectivity
  const startTime = Date.now();
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 5000); // 5 second timeout
    
    const response = await fetch(`${apiBaseUrl}/health`, {
      signal: controller.signal,
      headers: {
        'User-Agent': 'test-genie-ui-health-check'
      }
    });
    
    clearTimeout(timeoutId);
    const latency = Date.now() - startTime;
    
    if (response.ok) {
      health.api_connectivity.connected = true;
      health.api_connectivity.latency_ms = latency;
    } else {
      health.api_connectivity.error = `API returned ${response.status}`;
      health.status = 'degraded';
    }
  } catch (error) {
    health.api_connectivity.error = error.message;
    health.status = 'degraded';
    
    // Provide specific error context
    if (error.name === 'AbortError') {
      health.api_connectivity.error = 'API connection timeout (>5s)';
    } else if (error.message.includes('ECONNREFUSED')) {
      health.api_connectivity.error = `API not reachable at ${apiBaseUrl}`;
    }
  }
  
  // Set appropriate HTTP status code
  const httpStatus = health.status === 'healthy' ? 200 : 503;
  res.status(httpStatus).json(health);
});

// Enhanced API proxy with WebSocket notifications (must live before SPA catch-all)
app.use('/api', async (req, res) => {
  try {
    const fetch = (await import('node-fetch')).default;

    // Preserve the full requested path (including /api prefix and query string)
    const baseForJoin = apiBaseUrl.endsWith('/') ? apiBaseUrl : `${apiBaseUrl}/`;
    const targetUrl = new URL(req.originalUrl, baseForJoin);

    const options = {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        ...req.headers,
      },
    };

    if (req.method !== 'GET' && req.method !== 'HEAD') {
      options.body = JSON.stringify(req.body ?? {});
    }

    delete options.headers.host;
    delete options.headers['content-length'];

    const response = await fetch(targetUrl, options);
    const data = await response.text();

    try {
      const jsonData = JSON.parse(data);

      if (req.path.includes('/test-suite/') && req.path.includes('/execute') && req.method === 'POST') {
        broadcastToSubscribers('executions', 'execution_started', {
          execution_id: jsonData.execution_id,
          timestamp: new Date().toISOString(),
        });
      }

      if (req.path.includes('/test-execution') && jsonData.status === 'completed') {
        broadcastToSubscribers('executions', 'execution_update', {
          execution_id: jsonData.execution_id,
          status: jsonData.status,
          timestamp: new Date().toISOString(),
        });
      }
    } catch (e) {
      // Ignore JSON parsing failures for non-JSON responses
    }

    res.status(response.status);
    response.headers.forEach((value, key) => {
      if (!key.toLowerCase().includes('content-encoding')) {
        res.setHeader(key, value);
      }
    });

    res.send(data);
  } catch (error) {
    console.error('API proxy error:', error);
    res.status(500).json({ error: 'API request failed' });
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
