const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const morgan = require('morgan');
const path = require('path');

const app = express();

// Security and performance middleware
app.use(helmet({
  contentSecurityPolicy: {
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'", "https://fonts.googleapis.com"],
      fontSrc: ["'self'", "https://fonts.gstatic.com"],
      scriptSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'", "http://localhost:8200", "ws:", "wss:"]
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

// API proxy endpoints to avoid CORS issues
app.use('/api', async (req, res) => {
  const apiBaseUrl = process.env.TEST_GENIE_API_URL || 'http://localhost:8200';
  
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

const port = process.env.PORT || 3000;

app.listen(port, () => {
  console.log(`🧪 Test Genie UI server running on port ${port}`);
  console.log(`📊 Dashboard: http://localhost:${port}`);
  console.log(`🔗 API Proxy: ${process.env.TEST_GENIE_API_URL || 'http://localhost:8200'}`);
});