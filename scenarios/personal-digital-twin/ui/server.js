const express = require('express');
const cors = require('cors');
const morgan = require('morgan');
const path = require('path');

const app = express();
const PORT = process.env.PORT || process.env.UI_PORT || 3080;
const API_URL = process.env.API_URL || `http://localhost:${process.env.SERVICE_PORT || 8090}`;
const CHAT_URL = process.env.CHAT_URL || `http://localhost:${process.env.CHAT_PORT || 8000}`;

// Middleware
app.use(cors());
app.use(morgan('dev'));
app.use(express.json());
app.use(express.static('public'));

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
    res.sendFile(path.join(__dirname, 'public', 'index.html'));
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