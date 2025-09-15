const express = require('express');
const path = require('path');
const cors = require('cors');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Middleware
app.use(cors());
app.use(express.json());

// Serve the main page with injected config (BEFORE static files)
app.get('/', (req, res) => {
    const fs = require('fs');
    let html = fs.readFileSync(path.join(__dirname, 'index.html'), 'utf8');
    
    // Inject API port configuration
    const configScript = `
    <script>
        window.API_PORT = '${process.env.API_PORT || '20000'}';
    </script>`;
    
    html = html.replace('<script src="script.js"></script>', configScript + '\n    <script src="script.js"></script>');
    res.send(html);
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'agent-dashboard-ui' });
});

// Serve static files (CSS, JS, etc) AFTER custom routes
app.use(express.static(__dirname));

// Start server
app.listen(PORT, () => {
    console.log(`Agent Dashboard UI running on http://localhost:${PORT}`);
    console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
});