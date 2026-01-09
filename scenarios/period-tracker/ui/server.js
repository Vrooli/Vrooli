const express = require('express');
const cors = require('cors');
const path = require('path');
const fs = require('fs');

const app = express();
const PORT = Number(process.env.UI_PORT || process.env.PORT || 36000);
const distPath = path.join(__dirname, 'dist');
const staticRoot = fs.existsSync(distPath) ? distPath : __dirname;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(staticRoot));
app.use('/node_modules', express.static(path.join(__dirname, 'node_modules')));

// Serve static files
const assetRoot = fs.existsSync(path.join(staticRoot, 'assets'))
  ? path.join(staticRoot, 'assets')
  : path.join(__dirname, 'assets');
app.use('/assets', express.static(assetRoot));

// Main route
app.get('/', (req, res) => {
    res.sendFile(path.join(staticRoot, 'index.html'));
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
