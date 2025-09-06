const express = require('express');
const path = require('path');
const app = express();

const PORT = process.env.PORT || process.env.UI_PORT || 8420;
const API_URL = process.env.API_MANAGER_URL || 'http://localhost:8421';

// Serve static files
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ 
    status: 'ok', 
    service: 'api-manager-ui',
    timestamp: new Date().toISOString()
  });
});

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