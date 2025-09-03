const express = require('express');
const path = require('path');
const http = require('http');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_URL = process.env.API_URL || 'http://localhost:20260';

app.use(express.static(__dirname));

app.get('/env.js', (req, res) => {
    res.type('application/javascript');
    res.send(`
        window.ENV = {
            API_URL: '${API_URL}',
            API_PORT: '${process.env.API_PORT || 20260}'
        };
    `);
});

app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'knowledge-observatory-ui' });
});

app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

const server = http.createServer(app);

server.listen(PORT, () => {
    console.log(`🔭 Knowledge Observatory UI running on port ${PORT}`);
    console.log(`📡 Connected to API at ${API_URL}`);
    console.log(`🌐 Access dashboard at http://localhost:${PORT}`);
});

process.on('SIGINT', () => {
    console.log('\n🛑 Shutting down Knowledge Observatory UI...');
    server.close(() => {
        process.exit(0);
    });
});