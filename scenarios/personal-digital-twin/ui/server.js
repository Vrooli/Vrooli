const express = require('express');
const cors = require('cors');
const morgan = require('morgan');
const path = require('path');
const fs = require('fs');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;
const API_URL = process.env.API_URL || `http://localhost:${process.env.API_PORT || 8090}`;
const CHAT_URL = process.env.CHAT_URL || `http://localhost:${process.env.CHAT_PORT || 8000}`;
const distPath = path.join(__dirname, 'dist');
const publicPath = path.join(__dirname, 'public');
const staticRoot = fs.existsSync(distPath) ? distPath : publicPath;

// Middleware
app.use(cors());
app.use(morgan('dev'));
app.use(express.json());
app.use(express.static(staticRoot));
app.use('/node_modules', express.static(path.join(__dirname, 'node_modules')));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        service: 'personal-digital-twin-ui',
        api_url: API_URL,
        chat_url: CHAT_URL
    });
});

// Config endpoint for frontend
app.get('/config', (req, res) => {
    res.json({
        apiUrl: API_URL,
        chatUrl: CHAT_URL,
        appName: 'Digital Twin Console',
        version: '1.0.0'
    });
});

// Serve the main page
app.get('*', (req, res) => {
    res.sendFile(path.join(staticRoot, 'index.html'));
});

// Start server
app.listen(PORT, () => {
    console.log(`
╔══════════════════════════════════════════════════╗
║        DIGITAL TWIN CONSOLE v1.0                  ║
║        Consciousness Transfer Protocol Active     ║
╚══════════════════════════════════════════════════╝

🌐 UI Server running on http://localhost:${PORT}
🔗 API endpoint: ${API_URL}
💬 Chat endpoint: ${CHAT_URL}
    `);
});
