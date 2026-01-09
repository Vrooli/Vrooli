const express = require('express');
const path = require('path');
const fs = require('fs');
const http = require('http');
const { spawnSync } = require('child_process');

const app = express();

const PORT = Number(process.env.UI_PORT || process.env.PORT || 4173);
const UI_ROOT = __dirname;
const BUILD_DIR = path.join(UI_ROOT, 'build');
const BUILD_INDEX = path.join(BUILD_DIR, 'index.html');
const PROXY_HOST = '127.0.0.1';
const SCENARIO_NAME = 'retro-game-launcher';

const apiPortNumber = () => {
  const raw = process.env.API_PORT;
  const parsed = Number(raw);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
};

const hasBuildOutput = () => fs.existsSync(BUILD_INDEX);

const attemptAutoBuild = () => {
  if (process.env.AUTO_BUILD_UI === 'false') {
    return;
  }

  if (!fs.existsSync(path.join(UI_ROOT, 'node_modules'))) {
    console.warn('[retro-game-launcher-ui] Skipping auto build because node_modules is missing.');
    return;
  }

  console.warn('[retro-game-launcher-ui] UI build assets missing – running `npm run build`.');
  try {
    const result = spawnSync('npm', ['run', 'build'], {
      cwd: UI_ROOT,
      stdio: 'inherit',
      env: process.env,
    });
    if (result.status !== 0) {
      console.error(`[retro-game-launcher-ui] \`npm run build\` exited with code ${result.status}.`);
    }
  } catch (error) {
    console.error('[retro-game-launcher-ui] Failed to build UI assets automatically.', error);
  }
};

const resolveStaticRoot = () => {
  if (hasBuildOutput()) {
    return BUILD_DIR;
  }

  attemptAutoBuild();

  if (hasBuildOutput()) {
    return BUILD_DIR;
  }

  return null;
};

const STATIC_ROOT = resolveStaticRoot();
const STATIC_INDEX = STATIC_ROOT ? path.join(STATIC_ROOT, 'index.html') : null;

function proxyToApi(req, res, upstreamPath) {
  const targetPort = apiPortNumber();
  if (!targetPort) {
    res.status(502).json({
      error: 'API proxy not configured',
      details: 'API_PORT environment variable is missing or invalid.',
    });
    return;
  }

  const proxiedPath = upstreamPath || req.originalUrl || req.url || '/';
  const headers = {
    ...req.headers,
    host: `localhost:${targetPort}`,
  };

  const options = {
    hostname: PROXY_HOST,
    port: targetPort,
    path: proxiedPath,
    method: req.method,
    headers,
  };

  const proxyReq = http.request(options, (proxyRes) => {
    res.status(proxyRes.statusCode || 502);
    Object.entries(proxyRes.headers).forEach(([key, value]) => {
      if (typeof value !== 'undefined') {
        res.setHeader(key, value);
      }
    });
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (error) => {
    console.error(`[retro-game-launcher-ui] Proxy error for ${req.method} ${proxiedPath}:`, error.message);
    if (!res.headersSent) {
      res.status(502).json({
        error: 'API proxy request failed',
        details: error.message,
        target: `http://${PROXY_HOST}:${targetPort}${proxiedPath}`,
      });
    } else {
      res.end();
    }
  });

  if (req.method === 'GET' || req.method === 'HEAD') {
    proxyReq.end();
  } else if (req.readable) {
    req.pipe(proxyReq);
  } else {
    proxyReq.end();
  }
}

app.use('/api', (req, res) => {
  const suffix = req.url.startsWith('/') ? req.url : `/${req.url}`;
  const fullApiPath = `/api${suffix}`;
  proxyToApi(req, res, fullApiPath);
});

if (STATIC_ROOT) {
  app.use(express.static(STATIC_ROOT, {
    index: false,
    setHeaders: (res, servedPath) => {
      if (servedPath.endsWith('.js')) {
        res.setHeader('Cache-Control', 'no-cache');
      }
    },
  }));
} else {
  console.warn('[retro-game-launcher-ui] No built UI assets found; requests will receive guidance.');
}

app.get('/health', (req, res) => proxyToApi(req, res, '/health'));

app.get('*', (req, res) => {
  if (STATIC_INDEX && fs.existsSync(STATIC_INDEX)) {
    res.sendFile(STATIC_INDEX);
    return;
  }

  res.status(503).json({
    error: 'UI assets unavailable',
    hint: `Run npm run build in scenarios/${SCENARIO_NAME}/ui to generate the production bundle.`,
  });
});

const server = app.listen(PORT, () => {
  console.log(`✅ UI server running on http://localhost:${PORT}`);
  if (apiPortNumber()) {
    console.log(`📡 API proxy forwarding to http://${PROXY_HOST}:${apiPortNumber()}`);
  } else {
    console.warn('⚠️  API_PORT is not configured; API requests will fail until it is set.');
  }
  console.log(`🏷️  Scenario: ${SCENARIO_NAME}`);
});

process.on('SIGTERM', () => {
  server.close(() => {
    console.log('[retro-game-launcher-ui] UI server stopped');
  });
});
