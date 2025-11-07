const express = require('express');
const path = require('path');
const http = require('http');
const https = require('https');
const { pipeline } = require('stream');
const { URL } = require('url');

const app = express();
const PORT = Number(process.env.UI_PORT || process.env.PORT || 3000);
const UI_HOST = (process.env.UI_HOST || '0.0.0.0').trim() || '0.0.0.0';

const rawApiBase = (process.env.API_BASE_URL
  || process.env.VITE_API_BASE_URL
  || '').trim();
const fallbackApiHost = (process.env.API_HOST || '127.0.0.1').trim() || '127.0.0.1';
const apiPort = (process.env.API_PORT || '').trim();
const apiProtocol = (process.env.API_PROTOCOL || 'http').trim().replace(/:+$/, '') || 'http';

const toUrl = value => {
  if (!value) {
    return undefined;
  }
  try {
    return new URL(value);
  } catch (error) {
    console.warn(`Unable to parse API base URL "${value}":`, error);
    return undefined;
  }
};

const apiBaseUrl = toUrl(rawApiBase)
  || (apiPort ? toUrl(`${apiProtocol}://${fallbackApiHost}:${apiPort}`) : undefined);

const apiOrigin = apiBaseUrl ? `${apiBaseUrl.protocol}//${apiBaseUrl.host}` : undefined;
const apiPathPrefix = apiBaseUrl ? apiBaseUrl.pathname.replace(/\/$/, '') : '';

function proxyToApi(req, res) {
  if (!apiBaseUrl) {
    res.status(502).json({
      error: 'API_TARGET_UNDEFINED',
      message: 'API base URL is not configured; set API_BASE_URL or API_PORT',
    });
    return;
  }

  const suffix = req.url.startsWith('/') ? req.url : `/${req.url}`;
  const targetPath = `${apiPathPrefix}${suffix}` || '/';
  const requestOptions = {
    protocol: apiBaseUrl.protocol,
    hostname: apiBaseUrl.hostname,
    port: apiBaseUrl.port || (apiBaseUrl.protocol === 'https:' ? 443 : 80),
    method: req.method,
    path: targetPath,
    headers: {
      ...req.headers,
      host: apiBaseUrl.host,
    },
  };

  const httpModule = apiBaseUrl.protocol === 'https:' ? https : http;
  const proxyReq = httpModule.request(requestOptions, proxyRes => {
    res.status(proxyRes.statusCode || 500);
    Object.entries(proxyRes.headers || {}).forEach(([key, value]) => {
      if (typeof value !== 'undefined') {
        res.setHeader(key, value);
      }
    });

    pipeline(proxyRes, res, error => {
      if (error) {
        console.error('Proxy response pipeline failed:', error);
      }
    });
  });

  proxyReq.on('error', error => {
    console.error(`API proxy request to ${apiOrigin || 'unknown-target'}${targetPath} failed:`, error);
    if (!res.headersSent) {
      res.status(502).json({
        error: 'API_REQUEST_FAILED',
        message: 'Unable to reach API service',
        details: error.message,
      });
    } else {
      res.end();
    }
  });

  pipeline(req, proxyReq, error => {
    if (error) {
      console.error('Proxy request pipeline failed:', error);
      proxyReq.destroy(error);
    }
  });
}

app.disable('x-powered-by');

app.use('/api', proxyToApi);

// Serve static files from dist directory
app.use(express.static(path.join(__dirname, 'dist')));

// Health check endpoint (UI availability)
app.get('/health', (req, res) => {
  res.json({ status: 'healthy', service: 'system-monitor-ui' });
});

// Catch all route - serve index.html for SPA routing
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

app.listen(PORT, UI_HOST, () => {
  const displayHost = UI_HOST === '0.0.0.0' ? '127.0.0.1' : UI_HOST;
  console.log(`System Monitor UI running on http://${displayHost}:${PORT}`);
  if (apiBaseUrl) {
    console.log(`API proxying via ${apiOrigin}${apiPathPrefix || ''}`);
  } else if (apiPort) {
    console.warn('API base URL not resolved; configure API_BASE_URL or API_PORT for proxying');
  }
});
