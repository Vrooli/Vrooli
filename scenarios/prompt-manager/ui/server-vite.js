const express = require('express');
const path = require('path');
const fs = require('fs');
const app = express();

// Port configuration - REQUIRED, no defaults
const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

const API_PORT = process.env.API_PORT;
if (!API_PORT) {
    console.error('❌ API_PORT environment variable is required');
    process.exit(1);
}

const SCENARIO_NAME = process.env.SCENARIO_NAME || 'prompt-manager';

// Enable CORS for API communication
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', 'http://localhost:3000');
    res.header('Access-Control-Allow-Headers', 'Content-Type');
    next();
});

// Serve static files from dist directory
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        port: PORT,
        scenario: SCENARIO_NAME,
        mode: 'production'
    });
});

// API proxy configuration
app.get('/api/config', (req, res) => {
    res.json({ 
        apiUrl: `http://localhost:${API_PORT}`,
        uiPort: PORT,
        resources: process.env.RESOURCE_PORTS ? JSON.parse(process.env.RESOURCE_PORTS) : {}
    });
});

// Catch all handler: send back React's index.html file for client-side routing
app.get('*', (req, res) => {
    const indexFile = path.join(__dirname, 'dist', 'index.html');
    if (fs.existsSync(indexFile)) {
        res.sendFile(indexFile);
    } else {
        res.status(404).json({ error: 'Build not found. Please run npm run build first.' });
    }
});

const server = app.listen(PORT, () => {
    console.log(`✅ Prompt Manager UI server running on http://localhost:${PORT}`);
    console.log(`📡 API endpoint: http://localhost:${API_PORT}`);
    console.log(`🏷️  Scenario: ${SCENARIO_NAME}`);
    console.log(`🎨 Serving modern React UI from /dist/`);
    console.log(`🚀 Production mode with Vite build`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    server.close(() => {
        console.log('UI server stopped');
    });
});