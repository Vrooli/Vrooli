const WebSocket = require('ws');
const Redis = require('redis');
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

const port = config.websocket.port || 3002;

// Create WebSocket server
const wss = new WebSocket.Server({ 
  port: port,
  path: config.websocket.path || '/ws'
});

// Redis connection for pub/sub
const redis = Redis.createClient({
  url: resourceUrls.resources.storage.redis.url
});

const redisSubscriber = redis.duplicate();

// Track connected clients
const clients = new Set();

// Initialize Redis connections
async function initializeRedis() {
  try {
    await redis.connect();
    await redisSubscriber.connect();
    console.log('✅ Redis connections established');
    
    // Subscribe to relevant channels
    await redisSubscriber.subscribe('app-events', (message) => {
      broadcast('app-event', JSON.parse(message));
    });
    
    await redisSubscriber.subscribe('app-alerts', (message) => {
      broadcast('app-alert', { 
        type: 'health-alert',
        message: message,
        timestamp: new Date().toISOString()
      });
    });
    
    await redisSubscriber.subscribe('resource-alerts', (message) => {
      broadcast('resource-alert', { 
        type: 'resource-alert',
        message: message,
        timestamp: new Date().toISOString()
      });
    });
    
    await redisSubscriber.subscribe('app-notifications', (message) => {
      broadcast('notification', { 
        type: 'info',
        message: message,
        timestamp: new Date().toISOString()
      });
    });
    
    console.log('✅ Subscribed to Redis channels');
  } catch (error) {
    console.error('❌ Failed to initialize Redis:', error.message);
    process.exit(1);
  }
}

// Broadcast message to all connected clients
function broadcast(type, data) {
  const message = JSON.stringify({
    type: type,
    data: data,
    timestamp: new Date().toISOString()
  });
  
  clients.forEach(ws => {
    if (ws.readyState === WebSocket.OPEN) {
      try {
        ws.send(message);
      } catch (error) {
        console.error('Error sending message to client:', error.message);
        clients.delete(ws);
      }
    }
  });
}

// Send message to specific client
function sendToClient(ws, type, data) {
  const message = JSON.stringify({
    type: type,
    data: data,
    timestamp: new Date().toISOString()
  });
  
  if (ws.readyState === WebSocket.OPEN) {
    try {
      ws.send(message);
    } catch (error) {
      console.error('Error sending message to client:', error.message);
      clients.delete(ws);
    }
  }
}

// Handle WebSocket connections
wss.on('connection', (ws, req) => {
  console.log(`New client connected from ${req.socket.remoteAddress}`);
  clients.add(ws);
  
  // Send welcome message
  sendToClient(ws, 'connected', {
    message: 'Connected to App Monitor WebSocket',
    server: 'app-monitor-ws',
    version: '1.0.0'
  });
  
  // Handle incoming messages
  ws.on('message', async (data) => {
    try {
      const message = JSON.parse(data.toString());
      
      switch (message.type) {
        case 'subscribe':
          // Handle subscription to specific channels
          handleSubscription(ws, message.data);
          break;
          
        case 'ping':
          sendToClient(ws, 'pong', {
            timestamp: new Date().toISOString()
          });
          break;
          
        case 'get-stats':
          // Send current stats
          await sendCurrentStats(ws);
          break;
          
        default:
          sendToClient(ws, 'error', {
            message: 'Unknown message type',
            type: message.type
          });
      }
    } catch (error) {
      sendToClient(ws, 'error', {
        message: 'Invalid message format',
        error: error.message
      });
    }
  });
  
  // Handle client disconnect
  ws.on('close', () => {
    console.log('Client disconnected');
    clients.delete(ws);
  });
  
  // Handle errors
  ws.on('error', (error) => {
    console.error('WebSocket error:', error.message);
    clients.delete(ws);
  });
});

// Handle subscription requests
function handleSubscription(ws, subscription) {
  // Store subscription preferences on the WebSocket instance
  ws.subscriptions = ws.subscriptions || new Set();
  
  if (subscription.channels) {
    subscription.channels.forEach(channel => {
      ws.subscriptions.add(channel);
    });
  }
  
  sendToClient(ws, 'subscribed', {
    channels: Array.from(ws.subscriptions),
    message: 'Successfully subscribed to channels'
  });
}

// Send current statistics
async function sendCurrentStats(ws) {
  try {
    // Get basic stats from Redis if available
    const stats = {
      connected_clients: clients.size,
      uptime: process.uptime(),
      memory_usage: process.memoryUsage(),
      timestamp: new Date().toISOString()
    };
    
    sendToClient(ws, 'stats', stats);
  } catch (error) {
    sendToClient(ws, 'error', {
      message: 'Failed to get stats',
      error: error.message
    });
  }
}

// Periodic health broadcast
setInterval(() => {
  broadcast('system-status', {
    server: 'app-monitor-ws',
    status: 'healthy',
    clients: clients.size,
    uptime: process.uptime(),
    memory: process.memoryUsage(),
    timestamp: new Date().toISOString()
  });
}, 30000); // Every 30 seconds

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('Shutting down WebSocket server gracefully...');
  
  // Notify clients of shutdown
  broadcast('server-shutdown', {
    message: 'Server is shutting down',
    timestamp: new Date().toISOString()
  });
  
  // Close WebSocket server
  wss.close(() => {
    console.log('WebSocket server closed');
  });
  
  // Close Redis connections
  await redis.quit();
  await redisSubscriber.quit();
  
  process.exit(0);
});

// Start server
async function startServer() {
  await initializeRedis();
  
  console.log(`🔌 App Monitor WebSocket server running on port ${port}`);
  console.log(`📡 WebSocket endpoint: ws://localhost:${port}${config.websocket.path || '/ws'}`);
  console.log(`👥 Ready to accept connections`);
}

startServer().catch(console.error);

// Handle server errors
wss.on('error', (error) => {
  console.error('WebSocket server error:', error);
});

process.on('uncaughtException', (error) => {
  console.error('Uncaught exception:', error);
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  console.error('Unhandled rejection at:', promise, 'reason:', reason);
  process.exit(1);
});