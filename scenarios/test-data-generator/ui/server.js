const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.UI_PORT || 3002;
const API_PORT = process.env.API_PORT || 3001;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(path.join(__dirname)));

// Serve the main HTML file
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'test-data-generator-ui',
    api_url: `http://localhost:${API_PORT}`,
    timestamp: new Date().toISOString()
  });
});

// Proxy API requests to the API server
app.use('/api', (req, res) => {
  const apiUrl = `http://localhost:${API_PORT}${req.url}`;
  
  // This is a simple proxy - in production you'd want to use http-proxy-middleware
  res.json({
    message: 'API proxy not implemented - call API directly',
    api_url: apiUrl,
    method: req.method,
    body: req.body
  });
});

app.listen(PORT, () => {
  console.log(`Test Data Generator UI running on port ${PORT}`);
  console.log(`View at: http://localhost:${PORT}`);
  console.log(`API server should be running on port ${API_PORT}`);
});

module.exports = app;