const express = require('express');
const path = require('path');
const cors = require('cors');
const fs = require('fs');

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;

// Middleware
app.use(cors());
app.use(express.json());

// Security headers middleware
app.use((req, res, next) => {
    // Content Security Policy - restrict resource loading
    res.setHeader('Content-Security-Policy', 
        "default-src 'self'; " +
        "script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.jsdelivr.net; " +
        "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
        "font-src 'self' https://fonts.gstatic.com; " +
        "img-src 'self' data:; " +
        "connect-src 'self' http://localhost:*"
    );
    
    // Security headers for XSS and clickjacking protection
    res.setHeader('X-Content-Type-Options', 'nosniff');
    res.setHeader('X-Frame-Options', 'DENY');
    res.setHeader('X-XSS-Protection', '1; mode=block');
    res.setHeader('Referrer-Policy', 'no-referrer-when-downgrade');
    res.setHeader('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');
    
    next();
});

// Function to inject API port into HTML pages
function servePageWithConfig(htmlFile) {
    return (req, res) => {
        try {
            let html = fs.readFileSync(path.join(__dirname, htmlFile), 'utf8');
            
            // Inject API port configuration before the first script tag
            const configScript = `
    <script>
        window.API_PORT = '${process.env.API_PORT}';
    </script>`;
            
            // Find the first script tag and inject before it
            const scriptIndex = html.indexOf('<script');
            if (scriptIndex > -1) {
                html = html.slice(0, scriptIndex) + configScript + '\n    ' + html.slice(scriptIndex);
            } else {
                // If no script tag, add before closing body
                html = html.replace('</body>', configScript + '\n</body>');
            }
            
            res.send(html);
        } catch (error) {
            console.error(`Error serving ${htmlFile}:`, error);
            res.status(500).send('Internal Server Error');
        }
    };
}

// Serve all HTML pages with injected config
app.get('/', servePageWithConfig('index.html'));
app.get('/index.html', servePageWithConfig('index.html'));
app.get('/agents.html', servePageWithConfig('agents.html'));
app.get('/agent.html', servePageWithConfig('agent.html'));
app.get('/logs.html', servePageWithConfig('logs.html'));
app.get('/metrics.html', servePageWithConfig('metrics.html'));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'agent-dashboard-ui',
        apiPort: process.env.API_PORT,
        uiPort: PORT
    });
});

// Serve static files (CSS, JS, images, etc) AFTER custom routes
app.use(express.static(__dirname));

// Catch-all route for any unmatched paths - redirect to dashboard
app.get('*', (req, res) => {
    res.redirect('/');
});

// Start server
app.listen(PORT, () => {
    console.log(`Agent Dashboard UI running on http://localhost:${PORT}`);
    console.log(`API endpoint: http://localhost:${process.env.API_PORT}`);
    console.log(`Environment: ${process.env.NODE_ENV || 'development'}`);
    console.log('Available pages:');
    console.log(`  - Dashboard: http://localhost:${PORT}/`);
    console.log(`  - Agents: http://localhost:${PORT}/agents.html`);
    console.log(`  - Logs: http://localhost:${PORT}/logs.html`);
    console.log(`  - Metrics: http://localhost:${PORT}/metrics.html`);
});