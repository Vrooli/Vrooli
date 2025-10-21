const express = require('express');
const path = require('path');
const http = require('http');
const https = require('https');
const helmet = require('helmet');
const cors = require('cors');
const rateLimit = require('express-rate-limit');

const app = express();
const PORT = process.env.UI_PORT || 3200;
const API_BASE_URL = process.env.API_BASE_URL || `http://localhost:${process.env.API_PORT || 15200}`;

let parsedApiBaseUrl;
try {
  parsedApiBaseUrl = new URL(API_BASE_URL);
} catch (error) {
  console.error('Invalid API_BASE_URL provided for UI proxy:', API_BASE_URL, error);
  parsedApiBaseUrl = null;
}

function proxyToApi(req, res, upstreamPath) {
  if (!parsedApiBaseUrl) {
    res.status(502).json({
      error: 'API proxy misconfigured',
      message: 'API_BASE_URL could not be parsed',
      target: API_BASE_URL
    });
    return;
  }

  const targetPath = upstreamPath || req.originalUrl || req.url || '/api';
  const targetUrl = new URL(targetPath, parsedApiBaseUrl);
  const client = targetUrl.protocol === 'https:' ? https : http;

  const headers = {
    ...req.headers,
    host: targetUrl.host
  };

  // Let Node calculate the correct content length after potential body transforms
  delete headers['content-length'];

  const options = {
    hostname: targetUrl.hostname,
    port: targetUrl.port || (targetUrl.protocol === 'https:' ? 443 : 80),
    path: `${targetUrl.pathname}${targetUrl.search}`,
    method: req.method,
    headers
  };

  console.log(`[proxyToApi] ${req.method} ${req.originalUrl} -> ${targetUrl.href}`);

  const proxyReq = client.request(options, (proxyRes) => {
    res.status(proxyRes.statusCode || 500);
    Object.entries(proxyRes.headers).forEach(([key, value]) => {
      if (value !== undefined) {
        res.setHeader(key, value);
      }
    });
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (error) => {
    console.error('API proxy error:', error);
    if (!res.headersSent) {
      res.status(502).json({
        error: 'API connection failed',
        message: error.message,
        target: targetUrl.href
      });
    } else {
      res.end();
    }
  });

  if (req.method === 'GET' || req.method === 'HEAD' || req.method === 'OPTIONS') {
    proxyReq.end();
    return;
  }

  if (req.readable && typeof req.body === 'undefined') {
    req.pipe(proxyReq);
    return;
  }

  if (req.body) {
    const bodyPayload = typeof req.body === 'string' ? req.body : JSON.stringify(req.body);
    proxyReq.write(bodyPayload);
  }

  proxyReq.end();
}

const localFrameAncestors = [
  "'self'",
  'http://localhost:*',
  'http://127.0.0.1:*',
  'http://[::1]:*'
];

const extraFrameAncestors = (process.env.FRAME_ANCESTORS || '')
  .split(/\s+/)
  .filter(Boolean);

// Security middleware
app.use(helmet({
  frameguard: false,
  contentSecurityPolicy: {
    useDefaults: true,
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'", "https://cdnjs.cloudflare.com"],
      scriptSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'", API_BASE_URL, 'ws:', 'wss:'],
      frameAncestors: [...localFrameAncestors, ...extraFrameAncestors]
    }
  }
}));

app.use(cors());

// Rate limiting
const limiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100 // limit each IP to 100 requests per windowMs
});
app.use(limiter);

// Middleware
app.use(express.static(path.join(__dirname)));

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ 
    status: 'healthy', 
    service: 'bookmark-intelligence-hub-ui',
    timestamp: new Date().toISOString(),
    api_connection: API_BASE_URL
  });
});

// API proxy routes to avoid CORS issues
app.use('/api', (req, res) => {
  proxyToApi(req, res, req.originalUrl || `/api${req.url}`);
});

// Serve main application
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

// Catch-all handler
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('Server error:', err);
  res.status(500).json({ 
    error: 'Internal server error',
    message: process.env.NODE_ENV === 'development' ? err.message : 'Something went wrong'
  });
});

// Start server
app.listen(PORT, () => {
  console.log(`🔖 Bookmark Intelligence Hub UI running on http://localhost:${PORT}`);
  console.log(`📡 API connection: ${API_BASE_URL}`);
  console.log(`🏥 Health check: http://localhost:${PORT}/health`);
});
