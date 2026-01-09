const express = require('express');
const path = require('path');
const cors = require('cors');
const fs = require('fs');

const app = express();
const PORT = process.env.UI_PORT || 35500;

const DIST_DIR = path.join(__dirname, 'dist');
const SRC_DIR = path.join(__dirname, 'src');
const STATIC_ROOT = fs.existsSync(DIST_DIR) ? DIST_DIR : (fs.existsSync(SRC_DIR) ? SRC_DIR : __dirname);

// Enable CORS
app.use(cors());

// Serve static files
app.use(express.static(STATIC_ROOT));

// Default route
app.get('/', (req, res) => {
    res.sendFile(path.join(STATIC_ROOT, 'index.html'));
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
