import path from 'path';
import fs from 'fs';
import http from 'http';
import { spawnSync } from 'child_process';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const PORT = Number(process.env.UI_PORT || process.env.PORT || 4173);
const API_PORT = Number(process.env.API_PORT || 8090);
const API_URL = process.env.API_URL || `http://localhost:${API_PORT}`;

const distPath = path.join(__dirname, 'dist');
const publicPath = path.join(__dirname, 'public');
const nodeModulesPath = path.join(__dirname, 'node_modules');

function detectPackageManager() {
  if (fs.existsSync(path.join(__dirname, 'package-lock.json'))) {
    return 'npm';
  }
  if (fs.existsSync(path.join(__dirname, 'pnpm-lock.yaml'))) {
    return 'pnpm';
  }
  if (fs.existsSync(path.join(__dirname, 'yarn.lock'))) {
    return 'yarn';
  }
  return 'npm';
}

function runCommand(binary, args) {
  const result = spawnSync(binary, args, {
    cwd: __dirname,
    stdio: 'inherit',
    env: {
      ...process.env,
      CI: process.env.CI ?? '1',
    },
  });
  if (result.error) {
    console.error(`❌ Failed to run ${binary} ${args.join(' ')}:`, result.error.message);
    return false;
  }
  if (result.status !== 0) {
    console.error(`❌ ${binary} ${args.join(' ')} exited with code ${result.status}`);
    return false;
  }
  return true;
}

function ensureDependencies(packageManager) {
  if (fs.existsSync(nodeModulesPath)) {
    return true;
  }
  console.log('📦 Dependencies missing, installing UI workspace dependencies...');
  switch (packageManager) {
    case 'pnpm':
      return runCommand('pnpm', ['install', '--config.confirmModulesPurge=false']);
    case 'yarn':
      return runCommand('yarn', ['install', '--frozen-lockfile']);
    default:
      return runCommand('npm', ['install']);
  }
}

function ensureBuildArtifacts(packageManager) {
  if (fs.existsSync(distPath)) {
    return true;
  }

  console.log('⚙️  UI dist directory missing. Building assets automatically...');
  const buildArgs = packageManager === 'npm' ? ['run', 'build'] : ['build'];
  if (!runCommand(packageManager, buildArgs)) {
    return false;
  }

  return fs.existsSync(distPath);
}

const packageManager = detectPackageManager();
const depsReady = ensureDependencies(packageManager);
if (!depsReady) {
  console.error('❌ Unable to install UI dependencies. See logs above for details.');
  process.exit(1);
}

const hasDist = ensureBuildArtifacts(packageManager);
if (!hasDist) {
  console.warn('⚠️  UI build artifacts not available. Serving fallback page until build succeeds.');
}

const { default: express } = await import('express');
const app = express();

if (hasDist) {
  console.log('📦 Serving Vite build from dist directory');
  app.use(express.static(distPath));
} else {
  console.log('🛠️  Development mode: serving static assets from public directory');
  app.use(express.static(publicPath));
}

app.get('/config', (_req, res) => {
  res.json({
    apiUrl: API_URL,
    version: '2.0.0',
    service: 'ai-chatbot-manager-ui',
  });
});

app.get('/health', async (_req, res) => {
  const now = new Date();
  const healthResponse = {
    status: 'healthy',
    service: 'ai-chatbot-manager-ui',
    timestamp: now.toISOString(),
    readiness: true,
    api_connectivity: {
      connected: false,
      api_url: API_URL,
      last_check: now.toISOString(),
      error: null,
      latency_ms: null,
    },
  };

  const apiHost = new URL(API_URL);
  const startTime = Date.now();

  await new Promise((resolve) => {
    const req = http.request(
      {
        hostname: apiHost.hostname,
        port: apiHost.port || (apiHost.protocol === 'https:' ? 443 : 80),
        path: '/health',
        method: 'GET',
        timeout: 5000,
        headers: {
          Accept: 'application/json',
        },
      },
      (apiRes) => {
        const endTime = Date.now();
        healthResponse.api_connectivity.latency_ms = endTime - startTime;

        if ((apiRes.statusCode || 500) < 300) {
          healthResponse.api_connectivity.connected = true;
        } else {
          healthResponse.api_connectivity.connected = false;
          healthResponse.api_connectivity.error = {
            code: `HTTP_${apiRes.statusCode}`,
            message: `API returned status ${apiRes.statusCode}: ${apiRes.statusMessage}`,
            category: 'network',
            retryable: Boolean(apiRes.statusCode && apiRes.statusCode >= 500 && apiRes.statusCode < 600),
          };
          healthResponse.status = 'degraded';
        }
        resolve();
      }
    );

    req.on('error', (error) => {
      const endTime = Date.now();
      healthResponse.api_connectivity.latency_ms = endTime - startTime;
      healthResponse.api_connectivity.connected = false;

      let errorCode = 'CONNECTION_FAILED';
      let category = 'network';

      if (error.code === 'ECONNREFUSED') {
        errorCode = 'CONNECTION_REFUSED';
      } else if (error.code === 'ENOTFOUND') {
        errorCode = 'HOST_NOT_FOUND';
        category = 'configuration';
      } else if (error.code === 'ETIMEOUT') {
        errorCode = 'TIMEOUT';
      }

      healthResponse.api_connectivity.error = {
        code: errorCode,
        message: `Failed to connect to API: ${error.message}`,
        category,
        retryable: error.code !== 'ENOTFOUND',
        details: {
          error_code: error.code,
        },
      };
      healthResponse.status = error.code === 'ENOTFOUND' ? 'degraded' : 'unhealthy';
      resolve();
    });

    req.on('timeout', () => {
      const endTime = Date.now();
      healthResponse.api_connectivity.latency_ms = endTime - startTime;
      healthResponse.api_connectivity.connected = false;
      healthResponse.api_connectivity.error = {
        code: 'TIMEOUT',
        message: 'API health check timed out after 5 seconds',
        category: 'network',
        retryable: true,
      };
      healthResponse.status = 'unhealthy';
      req.destroy();
      resolve();
    });

    req.end();
  });

  const buildMode = hasDist ? 'production' : 'development';
  healthResponse.build_mode = buildMode;

  const widgetCandidatePaths = [
    path.join(distPath, 'widget.js'),
    path.join(publicPath, 'widget.js'),
  ];

  const widgetStatus = widgetCandidatePaths.some((file) => fs.existsSync(file));
  healthResponse.widget_files = widgetStatus
    ? {
        available: true,
        widget_js: true,
      }
    : {
        available: false,
        error: 'widget.js file not found in dist or public directories',
      };

  if (!widgetStatus && healthResponse.status === 'healthy') {
    healthResponse.status = 'degraded';
  }

  res.json(healthResponse);
});

app.get('*', (_req, res) => {
  const indexPath = hasDist
    ? path.join(distPath, 'index.html')
    : path.join(publicPath, 'index.html');

  if (fs.existsSync(indexPath)) {
    res.sendFile(indexPath);
  } else {
    res.status(404).send('<h1>AI Chatbot Manager UI</h1><p>No front-end build found. Run "pnpm build".</p>');
  }
});

app.listen(PORT, () => {
  console.log(`🚀 AI Chatbot Manager UI available at http://localhost:${PORT}`);
  console.log(`📡 API target: ${API_URL}`);
  console.log(`🏗️  Mode: ${hasDist ? 'Production (dist)' : 'Development (public)'}`);
});
