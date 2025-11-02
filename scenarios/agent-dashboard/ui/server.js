import express from 'express'
import path from 'path'
import cors from 'cors'
import fs from 'fs'
import http from 'http'
import https from 'https'
import { fileURLToPath } from 'url'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const app = express()
const PORT = process.env.UI_PORT || process.env.PORT
const BRIDGE_FLAG = '__agentDashboardBridgeInitialized'
const API_HOST = process.env.API_HOST || '127.0.0.1'
const API_PORT = process.env.API_PORT || '15000'
const API_PROTOCOL = (process.env.API_PROTOCOL || 'http').toLowerCase()
const apiBaseCandidate = process.env.API_BASE_URL || `${API_PROTOCOL}://${API_HOST}:${API_PORT}`

if (!PORT) {
  console.error('[AgentDashboard] UI_PORT environment variable is required')
  process.exit(1)
}

let parsedApiBase
try {
  parsedApiBase = new URL(apiBaseCandidate)
} catch (error) {
  console.warn('[AgentDashboard] Invalid API_BASE_URL provided:', apiBaseCandidate, error.message)
  parsedApiBase = null
}

function deriveParentOrigin() {
  if (typeof document === 'undefined' || !document.referrer) {
    return undefined
  }
  try {
    return new URL(document.referrer).origin
  } catch (error) {
    console.warn('[AgentDashboard] Unable to parse parent origin for iframe bridge', error)
    return undefined
  }
}

function initializeIframeBridge() {
  if (typeof window === 'undefined') {
    return
  }
  if (window.parent === window) {
    return
  }
  if (window[BRIDGE_FLAG]) {
    return
  }

  const parentOrigin = deriveParentOrigin()
  if (window.parent !== window) {
    initIframeBridgeChild({ appId: 'agent-dashboard', parentOrigin })
    window[BRIDGE_FLAG] = true
  }
}

try {
  initializeIframeBridge()
} catch (error) {
  console.error('[AgentDashboard] Failed to initialize iframe bridge', error)
}

// Middleware
app.use(cors())

// Security headers middleware
app.use((req, res, next) => {
  const csp = [
    "default-src 'self'",
    "script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.jsdelivr.net",
    "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
    "font-src 'self' https://fonts.gstatic.com",
    "img-src 'self' data: blob:",
    "connect-src 'self' https: http: ws: wss: data:",
    "frame-ancestors 'self' https://app-monitor.itsagitime.com https://*.itsagitime.com"
  ].join('; ')

  res.setHeader('Content-Security-Policy', `${csp};`)
  res.setHeader('X-Content-Type-Options', 'nosniff')
  res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin')
  res.setHeader('Strict-Transport-Security', 'max-age=31536000; includeSubDomains')
  next()
})

const proxyToApi = (req, res, upstreamPath) => {
  if (!parsedApiBase) {
    res.status(502).json({
      error: 'API_PROXY_UNCONFIGURED',
      message: 'Unable to resolve API_BASE_URL for proxying requests',
      target: apiBaseCandidate
    })
    return
  }

  const targetPath = upstreamPath || req.originalUrl || req.url || '/api'
  let targetUrl
  try {
    targetUrl = new URL(targetPath, parsedApiBase)
  } catch (error) {
    console.error('[AgentDashboard] Failed to build API proxy URL:', error.message)
    res.status(500).json({
      error: 'API_PROXY_TARGET_INVALID',
      message: error.message,
      target: targetPath
    })
    return
  }

  const client = targetUrl.protocol === 'https:' ? https : http
  const headers = { ...req.headers, host: targetUrl.host }
  delete headers['content-length']
  delete headers['host']

  const options = {
    hostname: targetUrl.hostname,
    port: targetUrl.port || (targetUrl.protocol === 'https:' ? 443 : 80),
    path: `${targetUrl.pathname}${targetUrl.search}`,
    method: req.method,
    headers
  }

  const proxyReq = client.request(options, (proxyRes) => {
    res.status(proxyRes.statusCode || 502)
    Object.entries(proxyRes.headers).forEach(([key, value]) => {
      if (typeof value !== 'undefined') {
        res.setHeader(key, value)
      }
    })
    proxyRes.pipe(res)
  })

  proxyReq.on('error', (error) => {
    console.error('[AgentDashboard] API proxy error:', error.message)
    if (!res.headersSent) {
      res.status(502).json({
        error: 'API_PROXY_ERROR',
        message: error.message,
        target: targetUrl.toString()
      })
    } else {
      res.end()
    }
  })

  req.on('aborted', () => {
    proxyReq.destroy()
  })

  const method = req.method ? req.method.toUpperCase() : 'GET'
  if (['GET', 'HEAD', 'OPTIONS'].includes(method)) {
    proxyReq.end()
    return
  }

  req.pipe(proxyReq)
}

app.use('/api', (req, res) => {
  const requestedPath = req.originalUrl || req.url || '/api'
  const upstreamPath = requestedPath.startsWith('/api') ? requestedPath : `/api${requestedPath}`
  proxyToApi(req, res, upstreamPath)
})

// Function to inject API port into HTML pages
function servePageWithConfig(htmlFile) {
    return (req, res) => {
        try {
            let html = fs.readFileSync(path.join(__dirname, htmlFile), 'utf8');
            
            // Inject API port configuration before the first script tag
            const apiPortValue = process.env.API_PORT ? String(process.env.API_PORT) : '';
            const configScript = `
    <script>
        window.API_PORT = '${apiPortValue}';
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
        uiPort: PORT,
        apiBase: parsedApiBase ? parsedApiBase.toString() : apiBaseCandidate
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
    console.log(`[AgentDashboard] UI running on http://localhost:${PORT}`)
    console.log(`[AgentDashboard] Proxying API requests to ${parsedApiBase ? parsedApiBase.toString() : apiBaseCandidate}`)
    console.log(`[AgentDashboard] Environment: ${process.env.NODE_ENV || 'development'}`)
})

export { proxyToApi }
