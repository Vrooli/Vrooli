const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.UI_PORT || 35500;

// Enable CORS
app.use(cors());

// Serve static files
app.use(express.static(__dirname));

// Default route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Standardized health endpoint for lifecycle monitoring
app.get('/health', (req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.json({
        status: 'healthy',
        service: 'job-to-scenario-pipeline-ui',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`Job Pipeline Dashboard running on http://localhost:${PORT}`);
});
