const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.TEXT_TOOLS_UI_PORT || 24000;

// Middleware
app.use(cors());
app.use(express.static('.'));

// Health check
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', timestamp: Date.now() });
});

// Serve index.html for all routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Start server
app.listen(PORT, () => {
    console.log(`Text Tools UI running on http://localhost:${PORT}`);
});