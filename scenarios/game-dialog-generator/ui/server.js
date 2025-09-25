const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const rateLimit = require('express-rate-limit');
const path = require('path');

// 🌿 Game Dialog Generator UI Server 🎮
// Jungle Platformer Adventure Theme

const app = express();
const PORT = process.env.UI_PORT || 3200;
const API_PORT = process.env.API_PORT || 8080;
const API_BASE_URL = `http://localhost:${API_PORT}`;

const localFrameAncestors = [
    "'self'",
    'http://localhost:*',
    'http://127.0.0.1:*',
    'http://[::1]:*'
];

const extraFrameAncestors = (process.env.FRAME_ANCESTORS || '')
    .split(/\s+/)
    .filter(Boolean);

// Security middleware
app.use(helmet({
    frameguard: false,
    contentSecurityPolicy: {
        useDefaults: true,
        directives: {
            defaultSrc: ["'self'"],
            scriptSrc: ["'self'", "'unsafe-inline'", "'unsafe-eval'"],
            styleSrc: ["'self'", "'unsafe-inline'", "https://fonts.googleapis.com"],
            fontSrc: ["'self'", "https://fonts.gstatic.com"],
            imgSrc: ["'self'", "data:", "https:"],
            connectSrc: ["'self'", API_BASE_URL, 'ws:', 'wss:'],
            mediaSrc: ["'self'"],
            frameAncestors: [...localFrameAncestors, ...extraFrameAncestors],
        },
    },
}));

// CORS configuration
app.use(cors({
    origin: process.env.NODE_ENV === 'production' ? false : '*',
    credentials: true
}));

// Compression and parsing
app.use(compression());
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true, limit: '10mb' }));

// Rate limiting for API proxies
const apiLimiter = rateLimit({
    windowMs: 15 * 60 * 1000, // 15 minutes
    max: 100, // limit each IP to 100 requests per windowMs
    message: {
        error: 'Too many API requests from this IP, please try again later.',
        jungle_message: '🌿 Easy there, adventurer! The jungle needs time to grow back! 🎮'
    }
});

// Apply rate limiting to API routes
app.use('/api', apiLimiter);

// Serve static files
app.use(express.static(path.join(__dirname, 'public')));

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({
        status: 'healthy',
        service: 'game-dialog-generator-ui',
        theme: 'jungle-platformer',
        timestamp: new Date().toISOString(),
        port: PORT,
        api_url: API_BASE_URL
    });
});

// API proxy endpoints - forward requests to the Go API
app.use('/api/v1', async (req, res) => {
    try {
        const fetch = (await import('node-fetch')).default;
        const apiUrl = `${API_BASE_URL}/api/v1${req.path}`;
        
        const options = {
            method: req.method,
            headers: {
                'Content-Type': 'application/json',
                ...req.headers
            }
        };
        
        if (req.method !== 'GET' && req.method !== 'HEAD') {
            options.body = JSON.stringify(req.body);
        }
        
        const response = await fetch(apiUrl, options);
        const data = await response.json();
        
        res.status(response.status).json(data);
    } catch (error) {
        console.error('API Proxy Error:', error);
        res.status(500).json({
            error: 'API proxy error',
            jungle_message: '🌿 The jungle spirits are having trouble communicating! 🎮',
            details: process.env.NODE_ENV === 'development' ? error.message : undefined
        });
    }
});

// Main application route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Character management routes
app.get('/characters', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/characters/create', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/characters/:id', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Dialog generation routes
app.get('/dialog', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/dialog/generate', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/dialog/batch', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Project management routes
app.get('/projects', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/projects/create', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.get('/projects/:id', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Catch-all route for SPA
app.get('*', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Error handling middleware
app.use((error, req, res, next) => {
    console.error('Server Error:', error);
    res.status(500).json({
        error: 'Internal server error',
        jungle_message: '🌿 The jungle spirits encountered an unexpected challenge! 🎮',
        details: process.env.NODE_ENV === 'development' ? error.message : undefined
    });
});

// 404 handler
app.use((req, res) => {
    res.status(404).json({
        error: 'Route not found',
        jungle_message: '🌿 This path through the jungle doesn\'t exist! 🎮',
        path: req.path
    });
});

// Start server
app.listen(PORT, () => {
    console.log('🌿='.repeat(50));
    console.log('🎮 Game Dialog Generator UI Server Started! 🌿');
    console.log('🌿='.repeat(50));
    console.log(`🌲 Jungle Adventure UI: http://localhost:${PORT}`);
    console.log(`🔧 API Backend: ${API_BASE_URL}`);
    console.log(`🎯 Environment: ${process.env.NODE_ENV || 'development'}`);
    console.log(`🌿 Theme: Jungle Platformer Adventure`);
    console.log('🌿='.repeat(50));
    console.log('🐒 Ready for character creation and dialog generation!');
    console.log('🎮 Time to swing into action!');
    console.log('🌿='.repeat(50));
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('🌿 Jungle adventure ending gracefully... 🎮');
    process.exit(0);
});

process.on('SIGINT', () => {
    console.log('🌿 Jungle adventure interrupted! Goodbye, adventurer! 🎮');
    process.exit(0);
});

module.exports = app;
