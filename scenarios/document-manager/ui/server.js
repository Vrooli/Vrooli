const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Serve static files
app.use(express.static(path.join(__dirname)));

// API proxy configuration - forward API calls to backend
const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:23250';

// Health check endpoint
app.get('/ui-health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'document-manager-ui',
        version: '2.0.0',
        timestamp: new Date().toISOString()
    });
});

// Serve the main application
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Handle SPA routing - serve index.html for any route
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`Document Manager UI running on port ${PORT}`);
    console.log(`Access the application at: http://localhost:${PORT}`);
    console.log(`API Backend: ${API_BASE_URL}`);
});

module.exports = app;