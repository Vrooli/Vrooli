const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;

if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

// Serve static files from current directory
app.use(express.static(__dirname));

// Provide config to frontend
app.get('/config', (req, res) => {
    res.json({
        apiUrl: `http://localhost:${API_PORT}`,
        version: '1.0.0',
        service: 'authentication-ui'
    });
});

// Serve main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Serve dashboard page - redirect to admin for now
app.get('/dashboard', (req, res) => {
    res.redirect('/admin');
});

// Serve admin dashboard
app.get('/admin', (req, res) => {
    res.sendFile(path.join(__dirname, 'admin.html'));
});

// Health check
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'authentication-ui',
        version: '1.0.0',
        timestamp: new Date().toISOString()
    });
});

const server = app.listen(PORT, () => {
    console.log(`Authentication UI server running on port ${PORT}`);
    console.log(`Access at: http://localhost:${PORT}`);
    console.log(`Health check: http://localhost:${PORT}/health`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('Received SIGTERM, shutting down gracefully');
    server.close(() => {
        console.log('Authentication UI server closed');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('Received SIGINT, shutting down gracefully');
    server.close(() => {
        console.log('Authentication UI server closed');
        process.exit(0);
    });
});