const http = require('http');

// UI_PORT and API_PORT must be set by the lifecycle system
const UI_PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;
if (!UI_PORT) {
  console.error('ERROR: UI_PORT environment variable is required');
  console.error('This server must be started via the Vrooli lifecycle system.');
  console.error('Use: make start or vrooli scenario start scenario-to-android');
  process.exit(1);
}
if (!API_PORT) {
  console.error('ERROR: API_PORT environment variable is required');
  console.error('This server must be started via the Vrooli lifecycle system.');
  console.error('Use: make start or vrooli scenario start scenario-to-android');
  process.exit(1);
}

// Check API connectivity
async function checkApiConnectivity() {
  const apiUrl = `http://localhost:${API_PORT}/health`;
  const startTime = Date.now();

  try {
    const response = await fetch(apiUrl);
    const latency = Date.now() - startTime;

    if (response.ok) {
      return {
        connected: true,
        api_url: apiUrl,
        last_check: new Date().toISOString(),
        error: null,
        latency_ms: latency
      };
    } else {
      return {
        connected: false,
        api_url: apiUrl,
        last_check: new Date().toISOString(),
        error: {
          code: 'HTTP_ERROR',
          message: `API returned status ${response.status}`,
          category: 'network',
          retryable: true
        },
        latency_ms: null
      };
    }
  } catch (error) {
    return {
      connected: false,
      api_url: apiUrl,
      last_check: new Date().toISOString(),
      error: {
        code: 'CONNECTION_FAILED',
        message: error.message,
        category: 'network',
        retryable: true
      },
      latency_ms: null
    };
  }
}

const server = http.createServer(async (req, res) => {
  // Health check endpoint
  if (req.url === '/health' || req.url === '/api/health') {
    const apiConnectivity = await checkApiConnectivity();

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      status: apiConnectivity.connected ? 'healthy' : 'degraded',
      service: 'scenario-to-android-ui',
      timestamp: new Date().toISOString(),
      readiness: true,
      api_connectivity: apiConnectivity
    }));
    return;
  }

  // Root page
  if (req.url === '/' || req.url === '/index.html') {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Scenario to Android</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 20px;
    }
    .container {
      background: white;
      border-radius: 16px;
      box-shadow: 0 20px 60px rgba(0,0,0,0.3);
      padding: 40px;
      max-width: 600px;
      width: 100%;
    }
    h1 {
      color: #333;
      margin-bottom: 10px;
      font-size: 32px;
    }
    .subtitle {
      color: #666;
      margin-bottom: 30px;
      font-size: 16px;
    }
    .feature {
      padding: 15px;
      margin: 10px 0;
      background: #f5f5f5;
      border-radius: 8px;
      border-left: 4px solid #667eea;
    }
    .feature h3 {
      color: #667eea;
      margin-bottom: 5px;
      font-size: 18px;
    }
    .feature p {
      color: #666;
      font-size: 14px;
      line-height: 1.5;
    }
    .cli-command {
      background: #1e1e1e;
      color: #00ff00;
      padding: 15px;
      border-radius: 8px;
      font-family: 'Courier New', monospace;
      margin: 20px 0;
      font-size: 14px;
    }
    .status {
      display: flex;
      align-items: center;
      gap: 10px;
      margin: 20px 0;
      padding: 15px;
      background: #e8f5e9;
      border-radius: 8px;
    }
    .status-dot {
      width: 12px;
      height: 12px;
      background: #4caf50;
      border-radius: 50%;
      animation: pulse 2s infinite;
    }
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.5; }
    }
  </style>
</head>
<body>
  <div class="container">
    <h1>🤖 Scenario to Android</h1>
    <p class="subtitle">Transform Vrooli scenarios into native Android apps</p>

    <div class="status">
      <div class="status-dot"></div>
      <strong>Server Online</strong>
    </div>

    <div class="feature">
      <h3>📱 Universal Conversion</h3>
      <p>Convert any Vrooli scenario into a native Android application with full device capabilities</p>
    </div>

    <div class="feature">
      <h3>🔌 Native Features</h3>
      <p>Access camera, GPS, notifications, storage, and other Android features via JavaScript bridge</p>
    </div>

    <div class="feature">
      <h3>📴 Offline First</h3>
      <p>Apps work without internet and sync automatically when connected</p>
    </div>

    <h3 style="margin-top: 30px; color: #333;">Quick Start</h3>
    <div class="cli-command">
      $ scenario-to-android build my-scenario<br>
      $ scenario-to-android test my-app.apk --device
    </div>

    <p style="color: #666; margin-top: 20px; font-size: 14px;">
      For full documentation and API details, see the README.md file or use
      <code style="background: #f5f5f5; padding: 2px 6px; border-radius: 4px;">scenario-to-android help</code>
    </p>
  </div>
</body>
</html>
    `);
    return;
  }

  // 404 for everything else
  res.writeHead(404, { 'Content-Type': 'text/plain' });
  res.end('Not Found');
});

server.listen(UI_PORT, () => {
  console.log(`🚀 Scenario to Android UI server running on http://localhost:${UI_PORT}`);
  console.log(`   Health check: http://localhost:${UI_PORT}/health`);
});
