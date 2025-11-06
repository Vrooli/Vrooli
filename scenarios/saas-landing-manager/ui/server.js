import fs from 'fs';
import path from 'path';
import { spawnSync } from 'child_process';
import express from 'express';
import cors from 'cors';
import { fileURLToPath } from 'url';
import {
  DEFAULT_UI_PORT,
  DEFAULT_API_PORT,
  HEALTH_CHECK_TIMEOUT,
  SERVICE_NAME,
} from './src/constants.ts';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/**
 * Normalizes a base path by removing trailing slashes and handling empty/root cases
 */
const normalizeBasePath = (value) => {
  if (!value) {
    return '';
  }

  const trimmed = String(value).trim();
  if (!trimmed || trimmed === '/') {
    return '';
  }

  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed;
};

const normalizedPort = (() => {
  const raw = process.env.UI_PORT || process.env.SAAS_LANDING_UI_PORT || process.env.PORT || String(DEFAULT_UI_PORT);
  const parsed = Number.parseInt(raw, 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    console.warn(`⚠️  Invalid UI port "${raw}". Falling back to ${DEFAULT_UI_PORT}.`);
    return DEFAULT_UI_PORT;
  }
  return parsed;
})();

const apiBaseUrl = (() => {
  const explicit = process.env.API_BASE_URL || process.env.API_URL || process.env.SAAS_LANDING_API_URL;
  if (explicit) {
    return explicit.replace(/\/$/, '');
  }

  const apiPort = process.env.API_PORT || process.env.SAAS_LANDING_API_PORT || String(DEFAULT_API_PORT);
  return `http://127.0.0.1:${apiPort}/api/v1`;
})();

const apiHealthUrl = (() => {
  const base = apiBaseUrl.replace(/\/$/, '');
  if (base.endsWith('/api/v1')) {
    return `${base.replace(/\/api\/v1$/, '')}/health`;
  }
  return `${base}/health`;
})();

const proxyBasePath = normalizeBasePath(process.env.VROOLI_PROXY_BASE_PATH || '');
const distPath = path.join(__dirname, 'dist');
const publicPath = path.join(__dirname, 'public');
const indexPath = path.join(__dirname, 'index.html');
const nodeModulesPath = path.join(__dirname, 'node_modules');

function detectPackageManager() {
  if (fs.existsSync(path.join(__dirname, 'pnpm-lock.yaml'))) {
    return 'pnpm';
  }
  if (fs.existsSync(path.join(__dirname, 'yarn.lock'))) {
    return 'yarn';
  }
  return 'npm';
}

function runCommand(command, args) {
  const result = spawnSync(command, args, {
    cwd: __dirname,
    stdio: 'inherit',
    env: {
      ...process.env,
      CI: process.env.CI ?? '1',
    },
  });

  if (result.error) {
    console.error(`❌ Failed to run ${command} ${args.join(' ')}`, result.error);
    return false;
  }

  if (result.status !== 0) {
    console.error(`❌ ${command} ${args.join(' ')} exited with code ${result.status}`);
    return false;
  }

  return true;
}

function ensureDependencies(packageManager) {
  if (fs.existsSync(nodeModulesPath)) {
    return true;
  }

  console.log('📦 Installing SaaS Landing Manager UI dependencies...');
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

  console.log('⚙️  Building SaaS Landing Manager UI assets...');
  const args = packageManager === 'npm' ? ['run', 'build'] : ['build'];
  return runCommand(packageManager, args) && fs.existsSync(distPath);
}

const packageManager = detectPackageManager();
const depsReady = ensureDependencies(packageManager);
if (!depsReady) {
  console.error('❌ Unable to ensure UI dependencies are installed.');
  process.exit(1);
}

const hasDist = ensureBuildArtifacts(packageManager);
if (!hasDist) {
  console.warn('⚠️  UI build artifacts not found. Falling back to static index.html.');
}

const createHealthError = (code, message, category, retryable) => ({
  code,
  message,
  category,
  retryable,
});

const buildUiHealthPayload = async () => {
  const timestamp = new Date();
  const connectivity = {
    connected: false,
    api_url: apiHealthUrl,
    last_check: timestamp.toISOString(),
    error: null,
    latency_ms: null,
  };

  let status = 'healthy';
  let readiness = true;

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), HEALTH_CHECK_TIMEOUT);

  try {
    const start = Date.now();
    const response = await fetch(apiHealthUrl, { signal: controller.signal });
    if (!response.ok) {
      throw new Error(`API responded with status ${response.status}`);
    }
    connectivity.connected = true;
    connectivity.latency_ms = Date.now() - start;
  } catch (error) {
    status = 'degraded';
    readiness = false;
    connectivity.error = createHealthError(
      error.name === 'AbortError' ? 'TIMEOUT' : 'API_UNREACHABLE',
      error.message,
      'network',
      true
    );
  } finally {
    clearTimeout(timeout);
  }

  const memoryUsage = process.memoryUsage();
  const metrics = {
    memory_usage_mb: Math.round((memoryUsage.rss / (1024 * 1024)) * 100) / 100,
    build_artifacts: hasDist,
  };

  return {
    status,
    service: SERVICE_NAME,
    timestamp: timestamp.toISOString(),
    readiness,
    api_connectivity: connectivity,
    metrics,
    build: hasDist ? 'dist' : 'fallback',
  };
};

const sendHealthResponse = async (res) => {
  try {
    const payload = await buildUiHealthPayload();
    res.json(payload);
  } catch (error) {
    const timestamp = new Date().toISOString();
    res.status(500).json({
      status: 'unhealthy',
      service: SERVICE_NAME,
      timestamp,
      readiness: false,
      api_connectivity: {
        connected: false,
        api_url: apiHealthUrl,
        last_check: timestamp,
        latency_ms: null,
        error: createHealthError(
          error.name === 'AbortError' ? 'TIMEOUT' : 'HEALTH_CHECK_FAILURE',
          error.message,
          'internal',
          true
        ),
      },
      metrics: {
        memory_usage_mb: Math.round((process.memoryUsage().rss / (1024 * 1024)) * 100) / 100,
        build_artifacts: hasDist,
      },
    });
  }
};

const router = express.Router();
router.use(cors());
router.use(express.json());

if (hasDist) {
  router.use(express.static(distPath));
} else {
  router.use(express.static(__dirname));
  if (fs.existsSync(publicPath)) {
    router.use(express.static(publicPath));
  }
  router.use('/node_modules', express.static(nodeModulesPath));
}

router.get('/config', (_req, res) => {
  res.json({
    apiBaseUrl,
    proxyBasePath,
  });
});

router.get('/health', async (_req, res) => {
  await sendHealthResponse(res);
});

router.use('/api/v1', async (req, res) => {
  try {
    const targetBase = apiBaseUrl.endsWith('/') ? apiBaseUrl : `${apiBaseUrl}/`;
    const relativePath = req.url.startsWith('/') ? req.url.slice(1) : req.url;
    const target = `${targetBase}${relativePath}`;
    const headers = { ...req.headers };
    delete headers.host;

    const fetchOptions = {
      method: req.method,
      headers,
      redirect: 'follow',
    };

    const hasBody = req.method !== 'GET' && req.method !== 'HEAD';
    if (hasBody && req.body && Object.keys(req.body).length > 0) {
      fetchOptions.body = JSON.stringify(req.body);
      fetchOptions.headers['content-type'] = 'application/json';
    }

    const response = await fetch(target, fetchOptions);

    res.status(response.status);
    response.headers.forEach((value, key) => {
      if (key.toLowerCase() === 'content-encoding') {
        return;
      }
      res.setHeader(key, value);
    });

    const buffer = Buffer.from(await response.arrayBuffer());
    res.send(buffer);
  } catch (error) {
    console.error('API proxy error:', error);
    res.status(502).json({ error: 'API request failed' });
  }
});

const fallbackIndexPath = hasDist && fs.existsSync(path.join(distPath, 'index.html'))
  ? path.join(distPath, 'index.html')
  : indexPath;

router.get('*', (_req, res, next) => {
  if (!fs.existsSync(fallbackIndexPath)) {
    return next();
  }
  res.sendFile(fallbackIndexPath);
});

const app = express();
app.use(proxyBasePath || '/', router);

// Catch-all for unmatched routes (health check is handled by router above)
app.get('*', (_req, res) => {
  if (fs.existsSync(fallbackIndexPath)) {
    res.sendFile(fallbackIndexPath);
  } else {
    res.status(404).send('<h1>SaaS Landing Manager UI</h1><p>No front-end build found. Run "npm run build".</p>');
  }
});

app.listen(normalizedPort, () => {
  console.log(`🚀 SaaS Landing Manager UI running on http://localhost:${normalizedPort}${proxyBasePath || ''}`);
  console.log(`📡 API proxy to: ${apiBaseUrl}`);
  if (proxyBasePath) {
    console.log(`🔀 Mounted behind proxy base path: ${proxyBasePath}`);
  }
  console.log(`🏗️  Serving ${hasDist ? 'production build' : 'fallback static assets'}`);
});

const shutdown = () => {
  console.log('\n🛑 Shutting down SaaS Landing Manager UI...');
  process.exit(0);
};

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);
