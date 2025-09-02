const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();
const PORT = process.env.UI_PORT || process.env.VEGAN_UI_PORT || 3000;
const API_PORT = process.env.API_PORT || 8080;

// Inject API URL configuration into the HTML
app.get('/', (req, res) => {
    const indexPath = path.join(__dirname, 'index.html');
    fs.readFile(indexPath, 'utf8', (err, html) => {
        if (err) {
            return res.status(500).send('Error loading page');
        }

        // Inject API configuration before app.js loads
        const configScript = `
            <script>
                window.API_URL = 'http://localhost:${API_PORT}/api';
            </script>
        `;

        // Insert config right before the app.js script tag
        html = html.replace('<script src="app.js"></script>',
            configScript + '<script src="app.js"></script>');

        res.send(html);
    });
});

// Serve static files for other resources
app.use(express.static(__dirname));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'make-it-vegan-ui' });
});

app.listen(PORT, () => {
    console.log(`🌱 Make It Vegan UI running at http://localhost:${PORT}`);
    console.log(`💚 Ready to help with plant-based alternatives!`);
});