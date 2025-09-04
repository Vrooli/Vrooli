const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = process.env.UI_PORT || 36850;
const API_PORT = process.env.API_PORT || 30400;

const mimeTypes = {
    '.html': 'text/html',
    '.css': 'text/css',
    '.js': 'text/javascript',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.gif': 'image/gif',
    '.svg': 'image/svg+xml',
    '.ico': 'image/x-icon'
};

const server = http.createServer((req, res) => {
    // Handle CORS
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

    if (req.method === 'OPTIONS') {
        res.writeHead(204);
        res.end();
        return;
    }

    // Serve static files
    let filePath = '.' + req.url;
    if (filePath === './') {
        filePath = './index.html';
    }

    const extname = path.extname(filePath);
    const contentType = mimeTypes[extname] || 'application/octet-stream';

    // Inject API_PORT into JavaScript files
    if (extname === '.js' || filePath === './index.html') {
        fs.readFile(filePath, 'utf8', (error, content) => {
            if (error) {
                if (error.code === 'ENOENT') {
                    res.writeHead(404);
                    res.end('File not found');
                } else {
                    res.writeHead(500);
                    res.end('Server error: ' + error.code);
                }
            } else {
                // Inject environment variables
                content = content.replace(/window\.API_PORT \|\| \d+/g, `window.API_PORT || ${API_PORT}`);
                
                res.writeHead(200, { 'Content-Type': contentType });
                res.end(content, 'utf-8');
            }
        });
    } else {
        fs.readFile(filePath, (error, content) => {
            if (error) {
                if (error.code === 'ENOENT') {
                    res.writeHead(404);
                    res.end('File not found');
                } else {
                    res.writeHead(500);
                    res.end('Server error: ' + error.code);
                }
            } else {
                res.writeHead(200, { 'Content-Type': contentType });
                res.end(content);
            }
        });
    }
});

server.listen(PORT, () => {
    console.log(`Elo Swipe UI server running at http://localhost:${PORT}`);
    console.log(`Expecting API at http://localhost:${API_PORT}`);
});