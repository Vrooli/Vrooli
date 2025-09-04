const express = require('express');
const path = require('path');
const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Get port from environment variable or default to 31005

// Serve static files from current directory
app.use(express.static(__dirname));

// Inject environment variables into HTML
app.get('/', (req, res) => {
    const indexPath = path.join(__dirname, 'index.html');
    const fs = require('fs');
    
    fs.readFile(indexPath, 'utf8', (err, data) => {
        if (err) {
            res.status(500).send('Error loading page');
            return;
        }
        
        // Inject environment variables as script tag
        const envScript = `
        <script>
            window.API_PORT = '${API_PORT}';
            window.N8N_PORT = '${N8N_PORT}';
            window.UI_PORT = '${PORT}';
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
