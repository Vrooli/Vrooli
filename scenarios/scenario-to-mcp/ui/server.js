const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT || 3000;
const API_PORT = process.env.API_PORT;
const REGISTRY_PORT = process.env.REGISTRY_PORT;

// Serve static files from dist directory
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'scenario-to-mcp-ui' });
});

// Catch all route - serve index.html for SPA routing
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

app.listen(PORT, () => {
    console.log(`Scenario to MCP Dashboard running on http://localhost:${PORT}`);
    if (API_PORT) {
        console.log(`API available at http://localhost:${API_PORT}`);
    }
    if (REGISTRY_PORT) {
        console.log(`MCP Registry available at http://localhost:${REGISTRY_PORT}/registry`);
    }
});
