const express = require('express');
const path = require('path');
const fs = require('fs');
const { spawnSync } = require('child_process');

const SCENARIO_NAME = 'core-debugger';
const UI_PORT = Number(process.env.UI_PORT || process.env.PORT || 3000);
const API_PORT = process.env.API_PORT || process.env.API_HTTP_PORT;

const app = express();

app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Ensure the production bundle exists before serving static assets.
const distDir = path.join(__dirname, 'dist');
const indexFile = path.join(distDir, 'index.html');

function ensureDistBuilt() {
    if (fs.existsSync(indexFile)) {
        return true;
    }

    console.warn('⚠️  UI build artifacts not found. Running `npm run build`...');
    const buildResult = spawnSync('npm', ['run', 'build'], {
        cwd: __dirname,
        stdio: 'inherit',
        env: process.env
    });

    if (buildResult.status !== 0) {
        console.error('❌ Failed to build UI assets. The dashboard will not load.');
        return false;
    }

    return fs.existsSync(indexFile);
}

const hasDist = ensureDistBuilt();

if (hasDist) {
    app.use(express.static(distDir, { extensions: ['html'] }));
}

app.get('/health', (_req, res) => {
    res.json({
        status: 'healthy',
        scenario: SCENARIO_NAME,
        port: UI_PORT,
        apiPort: API_PORT || null,
        timestamp: new Date().toISOString()
    });
});

app.get('/api/config', (_req, res) => {
    res.json({
        scenario: SCENARIO_NAME,
        apiUrl: API_PORT ? `http://localhost:${API_PORT}/api/v1` : null,
        apiPort: API_PORT || null,
        refreshIntervalMs: Number(process.env.REFRESH_INTERVAL_MS) || 30000
    });
});

if (API_PORT) {
    app.all(['/api', '/api/*'], (req, res) => {
        const target = `http://localhost:${API_PORT}${req.originalUrl}`;
        res.redirect(307, target);
    });
}

app.get('*', (req, res, next) => {
    if (req.path.startsWith('/api/')) {
        return next();
    }

    // For asset requests (contain a dot), return 404 when bundle is unavailable.
    if (!hasDist && req.path.includes('.')) {
        return res.status(404).json({
            error: 'Requested asset not found because UI build failed.',
            path: req.path,
            scenario: SCENARIO_NAME
        });
    }

    if (!hasDist) {
        return res.status(503).json({
            error: "UI build not found. Run `npm install` then `npm run build` inside scenarios/core-debugger/ui. See server logs for details.",
            scenario: SCENARIO_NAME
        });
    }

    res.sendFile(indexFile);
});

const server = app.listen(UI_PORT, '0.0.0.0', () => {
    console.log('=====================================');
    console.log(`✅ ${SCENARIO_NAME} UI Server Started`);
    console.log('=====================================');
    console.log(`📍 UI:     http://localhost:${UI_PORT}`);
    if (API_PORT) {
        console.log(`🔌 API:    http://localhost:${API_PORT}`);
    }
    console.log(`💚 Health: http://localhost:${UI_PORT}/health`);
    console.log('=====================================');
});

const shutdown = () => {
    console.log(`\n🛑 Stopping ${SCENARIO_NAME} UI server...`);
    server.close(() => {
        console.log('UI server stopped gracefully');
        process.exit(0);
    });
};

process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);

server.on('error', (err) => {
    if (err.code === 'EADDRINUSE') {
        console.error(`❌ Port ${UI_PORT} is already in use`);
    } else {
        console.error('Server error:', err);
    }
    process.exit(1);
});
