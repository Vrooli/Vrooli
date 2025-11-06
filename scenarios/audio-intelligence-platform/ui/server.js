const express = require('express');
const fs = require('fs');
const path = require('path');
const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT;
const N8N_PORT = process.env.N8N_PORT || process.env.N8N_HTTP_PORT || '';

const distDir = path.join(__dirname, 'dist');
const staticRoot = fs.existsSync(distDir) ? distDir : __dirname;
const modulesDir = path.join(__dirname, 'node_modules');

if (fs.existsSync(modulesDir)) {
    app.use('/node_modules', express.static(modulesDir));
}

// Serve static files from built bundle when available
app.use(express.static(staticRoot));

// Inject environment variables into HTML
app.get('/', (req, res) => {
    const indexPath = path.join(staticRoot, 'index.html');
    
    fs.readFile(indexPath, 'utf8', (err, data) => {
        if (err) {
            res.status(500).send('Error loading page');
            return;
        }
        
        // Inject environment variables as script tag
        const envScript = `
        <script>
            window.API_PORT = '${API_PORT ?? ''}';
            window.N8N_PORT = '${N8N_PORT}';
            window.UI_PORT = '${PORT ?? ''}';
        </script>`;
        
        // Insert before closing head tag
        const modifiedHtml = data.replace('</head>', `${envScript}\n</head>`);
        res.send(modifiedHtml);
    });
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'ok', 
        service: 'audio-intelligence-platform-ui',
        port: PORT,
        timestamp: new Date().toISOString()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`🎙️ Audio Intelligence Platform UI running on http://localhost:${PORT}`);
    console.log(`📡 API server expected at http://localhost:${API_PORT}`);
    console.log(`🔧 N8N workflows at http://localhost:${N8N_PORT}`);
});
