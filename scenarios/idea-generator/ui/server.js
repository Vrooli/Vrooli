const express = require('express');
const path = require('path');
const cors = require('cors');
const axios = require('axios');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_URL = process.env.API_URL || 'http://localhost:23750';

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static(__dirname));

// Serve the main HTML file
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'idea-generator-ui', port: PORT });
});

// Proxy API requests to the backend
app.use('/api', async (req, res) => {
    try {
        const response = await axios({
            method: req.method,
            url: `${API_URL}/api${req.path}`,
            data: req.body,
            headers: {
                'Content-Type': 'application/json',
                ...req.headers
            }
        });
        res.status(response.status).json(response.data);
    } catch (error) {
        console.error('API proxy error:', error.message);
        if (error.response) {
            res.status(error.response.status).json(error.response.data);
        } else {
            res.status(500).json({ error: 'Internal server error', message: error.message });
        }
    }
});

// Start server
app.listen(PORT, () => {
    console.log(`🚀 Idea Generator UI server running on http://localhost:${PORT}`);
    console.log(`📡 Proxying API requests to ${API_URL}`);
});

module.exports = app;