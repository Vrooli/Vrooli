const express = require('express');
const fs = require('fs');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

const DIST_DIR = path.join(__dirname, 'dist');
const SRC_DIR = path.join(__dirname, 'src');
const distIndex = path.join(DIST_DIR, 'index.html');

function resolveStaticDir() {
    if (fs.existsSync(distIndex)) {
        return DIST_DIR;
    }

    console.warn('⚠️  UI bundle not built yet. Falling back to src/. Run `npm run build` for optimized assets.');
    return SRC_DIR;
}

const STATIC_DIR = resolveStaticDir();

// Serve static files
app.use(express.static(STATIC_DIR));

// Health check endpoint for orchestrator
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy',
        scenario: 'competitor-change-monitor',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});


// API proxy configuration
app.get('/config', (req, res) => {
    res.json({
        apiUrl: process.env.API_URL || 'http://localhost:8140'
    });
});

// Serve index.html for all routes
app.get('*', (req, res) => {
    res.sendFile(path.join(STATIC_DIR, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`🔍 Competitor Monitor UI running on http://localhost:${PORT}`);
});
