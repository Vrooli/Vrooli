const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = Number(process.env.UI_PORT || process.env.PORT || 36000);

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// Serve static files
app.use('/assets', express.static(path.join(__dirname, 'assets')));

// Main route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'period-tracker-ui',
        version: '1.0.0',
        privacy: 'enabled',
        timestamp: new Date().toISOString()
    });
});

// API proxy (optional - for development)
app.use('/api', (req, res) => {
    res.status(404).json({
        error: 'API requests should go directly to the API server',
        api_url: `http://localhost:${process.env.API_PORT || 16000}/api`
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`🩸 Period Tracker UI running on port ${PORT}`);
    console.log(`🌐 Access at: http://localhost:${PORT}`);
    console.log(`🔒 Privacy: All data encrypted and stored locally only`);
});

// Graceful shutdown
process.on('SIGINT', () => {
    console.log('\n🛑 Shutting down Period Tracker UI server...');
    process.exit(0);
});

process.on('SIGTERM', () => {
    console.log('\n🛑 Shutting down Period Tracker UI server...');
    process.exit(0);
});
