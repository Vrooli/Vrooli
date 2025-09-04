const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;

// Serve static files from current directory
app.use(express.static(__dirname));

// Provide config to frontend
app.get('/config', (req, res) => {
    res.json({
        apiUrl: `http://localhost:${API_PORT}`,
        version: '1.0.0',
        service: 'mass-update-tracker'
    });
});

// Serve main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        service: 'mass-update-tracker-ui',
        timestamp: new Date().toISOString()
    });
});

app.listen(PORT, () => {
    console.log(`🚀 Mass Update Tracker Dashboard running on http://localhost:${PORT}`);
    console.log(`📊 API: http://localhost:${API_PORT}`);
});
