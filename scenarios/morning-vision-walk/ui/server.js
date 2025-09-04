const express = require('express');
const cors = require('cors');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_URL = process.env.API_URL || 'http://localhost:8900';

// Middleware
app.use(cors());
app.use(express.static(__dirname));

// Inject API URL into the frontend
app.get('/script.js', (req, res) => {
    const fs = require('fs');
    let script = fs.readFileSync(path.join(__dirname, 'script.js'), 'utf8');
    script = `window.API_URL = '${API_URL}';\n` + script;
    res.type('application/javascript');
    res.send(script);
});

// Serve index.html for all routes
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🌅 Morning Vision Walk UI running on http://localhost:${PORT}`);
    console.log(`📡 Connected to API at ${API_URL}`);
});