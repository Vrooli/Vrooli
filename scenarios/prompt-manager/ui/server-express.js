const express = require('express');
const path = require('path');
const fs = require('fs');
const app = express();

// Port configuration - REQUIRED, no defaults
const PORT = process.env.UI_PORT;
if (!PORT) {
    console.error('❌ UI_PORT environment variable is required');
    process.exit(1);
}

const API_PORT = process.env.API_PORT;
if (!API_PORT) {
    console.error('❌ API_PORT environment variable is required');
    process.exit(1);
}

const SCENARIO_NAME = process.env.SCENARIO_NAME || 'prompt-manager';

// Enable CORS for API communication
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', 'http://localhost:3000');
    res.header('Access-Control-Allow-Headers', 'Content-Type');
    next();
});

// Serve static files with proper MIME types
app.use('/components', express.static(path.join(__dirname, 'components'), {
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Content-Type', 'application/javascript');
        }
    }
}));

app.use('/styles', express.static(path.join(__dirname, 'styles')));
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        port: PORT,
        scenario: SCENARIO_NAME
    });
});

// API proxy configuration
app.get('/api/config', (req, res) => {
    res.json({ 
        apiUrl: `http://localhost:${API_PORT}`,
        uiPort: PORT,
        resources: process.env.RESOURCE_PORTS ? JSON.parse(process.env.RESOURCE_PORTS) : {}
    });
});

// Route for dashboard (default)
app.get('/', (req, res) => {
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else {
        res.status(404).json({ error: 'Dashboard not found' });
    }
});

// Route for dashboard (explicit)
app.get('/dashboard.html', (req, res) => {
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else {
        res.status(404).json({ error: 'Dashboard not found' });
    }
});

// Route for prompt detail page
app.get('/prompt-detail.html', (req, res) => {
    const detailFile = path.join(__dirname, 'prompt-detail.html');
    if (fs.existsSync(detailFile)) {
        res.sendFile(detailFile);
    } else {
        res.status(404).json({ error: 'Prompt detail page not found' });
    }
});

// Fallback routing for SPAs - redirect to appropriate page
app.get('*', (req, res) => {
    // Check if it's a component or style request
    if (req.path.startsWith('/components/') || req.path.startsWith('/styles/')) {
        res.status(404).json({ error: 'File not found' });
        return;
    }

    // Default fallback to dashboard
    const dashboardFile = path.join(__dirname, 'dashboard.html');
    const indexFile = path.join(__dirname, 'index.html');
    
    if (fs.existsSync(dashboardFile)) {
        res.sendFile(dashboardFile);
    } else if (fs.existsSync(indexFile)) {
        res.sendFile(indexFile);
    } else {
        res.status(404).json({ error: 'No UI files found' });
    }
});

const server = app.listen(PORT, () => {
    console.log(`✅ UI server running on http://localhost:${PORT}`);
    console.log(`📡 API endpoint: http://localhost:${API_PORT}`);
    console.log(`🏷️  Scenario: ${SCENARIO_NAME}`);
    console.log(`📁 Serving modular UI components from /components/`);
    console.log(`🎨 Serving styles from /styles/`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    server.close(() => {
        console.log('UI server stopped');
    });
});