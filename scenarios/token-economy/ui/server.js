const express = require('express');
const path = require('path');
const cors = require('cors');
require('dotenv').config();

const app = express();
const PORT = process.env.PORT_TOKEN_ECONOMY_UI || 11081;

// Middleware
app.use(cors());
app.use(express.static(__dirname));

// API proxy configuration (if needed)
const API_URL = process.env.TOKEN_ECONOMY_API_URL || 'http://localhost:11080';

// Serve index.html for all routes (SPA)
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
    console.log(`Token Economy UI running on http://localhost:${PORT}`);
    console.log(`API URL: ${API_URL}`);
});