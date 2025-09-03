const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Serve static files
app.use(express.static(path.join(__dirname)));

// Main dashboard route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'web-scraper-manager-ui',
        timestamp: new Date().toISOString(),
        port: PORT
    });
});

// Start the server
app.listen(PORT, () => {
    console.log(`🚀 Web Scraper Manager Dashboard running on port ${PORT}`);
    console.log(`📊 Dashboard available at: http://localhost:${PORT}`);
    console.log(`🔍 Health check: http://localhost:${PORT}/health`);
});

module.exports = app;