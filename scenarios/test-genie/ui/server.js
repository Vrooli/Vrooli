const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const morgan = require('morgan');
const path = require('path');
const WebSocket = require('ws');
const http = require('http');

const app = express();

// Configure API base URL from environment
const apiBaseUrl = process.env.TEST_GENIE_API_URL || 'http://localhost:8200';

// Security and performance middleware
app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'", "https://fonts.googleapis.com"],
      fontSrc: ["'self'", "https://fonts.gstatic.com"],
      scriptSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'", apiBaseUrl, "ws:", "wss:"]
    }
  }
}));

app.use(cors());
app.use(compression());
app.use(morgan('combined'));
app.use(express.json());
app.use(express.static('.'));

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    timestamp: new Date().toISOString(),
    service: 'test-genie-ui'
  });
});

// Remove this duplicate - the enhanced API proxy is below

// Serve the main application
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

// Handle SPA routing
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

const port = process.env.PORT || 3000;

// Create HTTP server
const server = http.createServer(app);

// Create WebSocket server
const wss = new WebSocket.Server({ 
  server,
  path: '/ws',
  clientTracking: true
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

// Enhanced API proxy with WebSocket notifications
app.use('/api', async (req, res) => {
  
  try {
    const fetch = (await import('node-fetch')).default;
    const url = `${apiBaseUrl}${req.path}`;
    
    const options = {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        ...req.headers
      }
    };
    
    if (req.method !== 'GET' && req.method !== 'HEAD') {
      options.body = JSON.stringify(req.body);
    }
    
    delete options.headers.host;
    delete options.headers['content-length'];
    
    const response = await fetch(url, options);
    const data = await response.text();
    
    // Parse response to potentially broadcast updates
    try {
      const jsonData = JSON.parse(data);
      
      // Broadcast execution updates
      if (req.path.includes('/test-suite/') && req.path.includes('/execute') && req.method === 'POST') {
        broadcastToSubscribers('executions', 'execution_started', {
          execution_id: jsonData.execution_id,
          timestamp: new Date().toISOString()
        });
      }
      
      // Broadcast test completion
      if (req.path.includes('/test-execution') && jsonData.status === 'completed') {
        broadcastToSubscribers('executions', 'execution_update', {
          execution_id: jsonData.execution_id,
          status: jsonData.status,
          timestamp: new Date().toISOString()
        });
      }
    } catch (e) {
      // Ignore JSON parsing errors for non-JSON responses
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

// Start server
server.listen(port, () => {
  console.log(`🧪 Test Genie UI server running on port ${port}`);
  console.log(`📊 Dashboard: http://localhost:${port}`);
  console.log(`🔗 API Proxy: ${apiBaseUrl}`);
  console.log(`🔌 WebSocket: ws://localhost:${port}/ws`);
  console.log(`👥 WebSocket clients: ${wss.clients.size}`);
});