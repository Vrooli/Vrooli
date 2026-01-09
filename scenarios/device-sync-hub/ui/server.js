#!/usr/bin/env node

// Device Sync Hub UI Server
// Serves static files with dynamic configuration injection

const http = require('http');
const fs = require('fs');
const path = require('path');
const url = require('url');
const { execSync } = require('child_process');

function populateAuthEnvFromCli() {
    if (process.env.AUTH_SERVICE_URL && process.env.AUTH_UI_URL && process.env.AUTH_PORT) {
        return;
    }

    try {
        const output = execSync('vrooli scenario port scenario-authenticator', {
            stdio: ['ignore', 'pipe', 'ignore']
        }).toString().trim();

        output.split('\n').forEach((line) => {
            const [key, value] = line.split('=');
            if (!key || !value) {
                return;
            }
            if (!process.env.SCENARIO_AUTHENTICATOR_API_PORT && key === 'API_PORT') {
                process.env.SCENARIO_AUTHENTICATOR_API_PORT = value;
            }
            if (!process.env.SCENARIO_AUTHENTICATOR_UI_PORT && key === 'UI_PORT') {
                process.env.SCENARIO_AUTHENTICATOR_UI_PORT = value;
            }
        });

        if (!process.env.AUTH_SERVICE_URL && process.env.SCENARIO_AUTHENTICATOR_API_PORT) {
            process.env.AUTH_SERVICE_URL = `http://localhost:${process.env.SCENARIO_AUTHENTICATOR_API_PORT}`;
        }
        if (!process.env.AUTH_PORT && process.env.SCENARIO_AUTHENTICATOR_API_PORT) {
            process.env.AUTH_PORT = process.env.SCENARIO_AUTHENTICATOR_API_PORT;
        }
        if (!process.env.AUTH_UI_URL && process.env.SCENARIO_AUTHENTICATOR_UI_PORT) {
            process.env.AUTH_UI_URL = `http://localhost:${process.env.SCENARIO_AUTHENTICATOR_UI_PORT}`;
        }
    } catch (error) {
        // CLI may be unavailable in some environments; rely on fallbacks
    }
}

populateAuthEnvFromCli();

// Configuration from environment variables
// UI_PORT and API_PORT MUST be provided by the lifecycle system - fail fast if missing
if (!process.env.UI_PORT) {
    console.error('❌ FATAL: UI_PORT environment variable is required');
    console.error('   This should be set by the Vrooli lifecycle system');
    console.error('   Run: vrooli scenario start device-sync-hub');
    process.exit(1);
}
if (!process.env.API_PORT) {
    console.error('❌ FATAL: API_PORT environment variable is required');
    console.error('   This should be set by the Vrooli lifecycle system');
    console.error('   Run: vrooli scenario start device-sync-hub');
    process.exit(1);
}

const UI_PORT = parseInt(process.env.UI_PORT, 10);
const API_PORT = parseInt(process.env.API_PORT, 10);
// AUTH_PORT is optional - used only for display/meta tags, not required for functionality
// The API handles all auth validation, so UI doesn't need direct auth service access
const AUTH_PORT = process.env.AUTH_PORT ? parseInt(process.env.AUTH_PORT, 10) : null;
const AUTH_SERVICE_URL = process.env.AUTH_SERVICE_URL || '';
const AUTH_UI_URL = process.env.AUTH_UI_URL || '';
const STATIC_ROOT = fs.existsSync(path.join(__dirname, 'dist'))
    ? path.join(__dirname, 'dist')
    : __dirname;

function parseServiceUrl(rawValue, label) {
    if (!rawValue) {
        return null;
    }
    try {
        const parsed = new URL(rawValue);
        return {
            protocol: parsed.protocol.replace(':', '') || 'http',
            hostname: parsed.hostname,
            port: parsed.port ? parseInt(parsed.port, 10) : (parsed.protocol === 'https:' ? 443 : 80),
            pathname: parsed.pathname === '/' ? '' : parsed.pathname.replace(/\/$/, ''),
            search: parsed.search || ''
        };
    } catch (error) {
        console.warn(`⚠️  Unable to parse ${label} (${rawValue}): ${error.message}`);
        return { raw: rawValue };
    }
}

function formatInitialUrl(parsedConfig, fallback) {
    if (parsedConfig) {
        if (parsedConfig.raw) {
            return parsedConfig.raw;
        }
        const needsPort = (parsedConfig.protocol === 'https' && parsedConfig.port !== 443) ||
            (parsedConfig.protocol === 'http' && parsedConfig.port !== 80);
        const portSegment = needsPort ? `:${parsedConfig.port}` : '';
        return `${parsedConfig.protocol}://${parsedConfig.hostname}${portSegment}${parsedConfig.pathname || ''}${parsedConfig.search || ''}`;
    }
    return fallback;
}

const parsedAuthConfig = parseServiceUrl(AUTH_SERVICE_URL, 'AUTH_SERVICE_URL');
const parsedAuthUIConfig = parseServiceUrl(AUTH_UI_URL, 'AUTH_UI_URL');

const initialAuthLogUrl = formatInitialUrl(parsedAuthConfig, AUTH_PORT ? `http://localhost:${AUTH_PORT}` : 'http://localhost:15785');
const initialAuthUiLogUrl = formatInitialUrl(parsedAuthUIConfig, initialAuthLogUrl);

function resolveServiceUrl(parsedConfig, req) {
    if (!parsedConfig) {
        return { url: '', port: '' };
    }

    if (parsedConfig.raw) {
        return { url: parsedConfig.raw, port: '' };
    }

    const requestProtocol = req.headers['x-forwarded-proto'] || 'http';
    const hostHeader = req.headers.host || '';
    const hostnameFromRequest = hostHeader.split(':')[0] || 'localhost';

    const useSameHost = parsedConfig.hostname === 'localhost' || parsedConfig.hostname === '127.0.0.1';
    const protocol = parsedConfig.protocol || requestProtocol;
    const hostname = useSameHost ? hostnameFromRequest : parsedConfig.hostname;
    const port = parsedConfig.port;
    const needsPort = (protocol === 'https' && port !== 443) || (protocol === 'http' && port !== 80);
    const portSegment = needsPort ? `:${port}` : '';

    return {
        url: `${protocol}://${hostname}${portSegment}${parsedConfig.pathname || ''}${parsedConfig.search || ''}`,
        port: useSameHost ? String(port) : ''
    };
}

// Build URLs dynamically based on current host
function getConfigForRequest(req) {
    const protocol = req.headers['x-forwarded-proto'] || 'http';
    const host = req.headers.host || '';
    const hostname = host.split(':')[0] || 'localhost';

    let authPort = '';
    const resolvedAuth = resolveServiceUrl(parsedAuthConfig, req);
    let authUrl = resolvedAuth.url;
    if (resolvedAuth.port) {
        authPort = resolvedAuth.port;
    }

    if (!authUrl && AUTH_PORT) {
        authUrl = `${protocol}://${hostname}:${AUTH_PORT}`;
        authPort = String(AUTH_PORT);
    }

    if (!authUrl) {
        authUrl = `${protocol}://${hostname}:15785`;
        authPort = '15785';
    }

    const resolvedAuthUi = resolveServiceUrl(parsedAuthUIConfig, req);
    const authUiUrl = resolvedAuthUi.url || authUrl;

    return {
        apiUrl: `${protocol}://${hostname}:${API_PORT}`,
        authUrl,
        authUiUrl,
        apiPort: API_PORT,
        authPort
    };
}

// Mime types for static files
const mimeTypes = {
    '.html': 'text/html',
    '.js': 'application/javascript',
    '.css': 'text/css',
    '.json': 'application/json',
    '.png': 'image/png',
    '.jpg': 'image/jpeg',
    '.gif': 'image/gif',
    '.ico': 'image/x-icon',
    '.svg': 'image/svg+xml'
};

// Inject configuration into HTML files
function injectConfig(htmlContent, config) {
    return htmlContent
        .replace('<meta name="api-url" content="">', `<meta name="api-url" content="${config.apiUrl}">`)
        .replace('<meta name="auth-url" content="">', `<meta name="auth-url" content="${config.authUrl}">`)
        .replace('<meta name="auth-ui-url" content="">', `<meta name="auth-ui-url" content="${config.authUiUrl}">`)
        .replace('<meta name="api-port" content="">', `<meta name="api-port" content="${config.apiPort}">`)
        .replace('<meta name="auth-port" content="">', `<meta name="auth-port" content="${config.authPort}">`);
}

// Create HTTP server
const server = http.createServer((req, res) => {
    const parsedUrl = url.parse(req.url);
    let pathname = parsedUrl.pathname;

    // Health check endpoint
    if (pathname === '/health') {
        const now = new Date().toISOString();
        const apiUrl = `http://localhost:${API_PORT}`;

        // Try to check API connectivity
        const apiCheckStart = Date.now();
        let responseHandled = false;

        const sendResponse = (healthResponse) => {
            if (!responseHandled) {
                responseHandled = true;
                res.writeHead(200, {'Content-Type': 'application/json'});
                res.end(JSON.stringify(healthResponse));
            }
        };

        const apiCheckReq = http.get(`${apiUrl}/health`, { timeout: 2000 }, (apiRes) => {
            const latencyMs = Date.now() - apiCheckStart;
            sendResponse({
                status: apiRes.statusCode === 200 ? 'healthy' : 'degraded',
                service: 'device-sync-hub-ui',
                timestamp: now,
                readiness: true,
                api_connectivity: {
                    connected: apiRes.statusCode === 200,
                    api_url: apiUrl,
                    last_check: now,
                    latency_ms: latencyMs,
                    error: apiRes.statusCode !== 200 ? {
                        code: 'API_UNHEALTHY',
                        message: `API returned status ${apiRes.statusCode}`,
                        category: 'resource',
                        retryable: true
                    } : null
                }
            });
        });

        apiCheckReq.on('error', (err) => {
            sendResponse({
                status: 'degraded',
                service: 'device-sync-hub-ui',
                timestamp: now,
                readiness: true,
                api_connectivity: {
                    connected: false,
                    api_url: apiUrl,
                    last_check: now,
                    latency_ms: null,
                    error: {
                        code: 'API_CONNECTION_FAILED',
                        message: err.message || 'Failed to connect to API',
                        category: 'network',
                        retryable: true
                    }
                }
            });
        });

        apiCheckReq.on('timeout', () => {
            apiCheckReq.destroy();
            sendResponse({
                status: 'degraded',
                service: 'device-sync-hub-ui',
                timestamp: now,
                readiness: true,
                api_connectivity: {
                    connected: false,
                    api_url: apiUrl,
                    last_check: now,
                    latency_ms: null,
                    error: {
                        code: 'API_TIMEOUT',
                        message: 'API health check timed out after 2000ms',
                        category: 'network',
                        retryable: true
                    }
                }
            });
        });

        return;
    }

    // Default to index.html for root path
    if (pathname === '/') {
        pathname = '/index.html';
    }
    
    // Security: prevent directory traversal
    if (pathname.includes('..')) {
        res.writeHead(403, {'Content-Type': 'text/plain'});
        res.end('Forbidden');
        return;
    }
    
    const filePath = path.join(STATIC_ROOT, pathname);
    const ext = path.extname(filePath);
    
    // Check if file exists
    fs.access(filePath, fs.constants.F_OK, (err) => {
        if (err) {
            // File not found
            res.writeHead(404, {'Content-Type': 'text/plain'});
            res.end('Not Found');
            return;
        }
        
        // Read and serve file
        fs.readFile(filePath, (err, data) => {
            if (err) {
                res.writeHead(500, {'Content-Type': 'text/plain'});
                res.end('Internal Server Error');
                return;
            }
            
            const mimeType = mimeTypes[ext] || 'application/octet-stream';
            res.writeHead(200, {'Content-Type': mimeType});
            
            // Inject configuration for HTML files
            if (ext === '.html') {
                const config = getConfigForRequest(req);
                const injectedContent = injectConfig(data.toString(), config);
                res.end(injectedContent);
            } else {
                res.end(data);
            }
        });
    });
});

// Start server
server.listen(UI_PORT, () => {
    console.log(`🌐 Device Sync Hub UI server running on port ${UI_PORT}`);
    console.log(`📡 API URL: http://localhost:${API_PORT}`);
    if (initialAuthLogUrl) {
        console.log(`🔐 Auth URL: ${initialAuthLogUrl}`);
    }
    if (initialAuthUiLogUrl && initialAuthUiLogUrl !== initialAuthLogUrl) {
        console.log(`🪪 Auth UI: ${initialAuthUiLogUrl}`);
    }
    console.log(`🎯 UI URL: http://localhost:${UI_PORT}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
    console.log('📴 UI server shutting down...');
    server.close(() => {
        console.log('✅ UI server stopped');
        process.exit(0);
    });
});

process.on('SIGINT', () => {
    console.log('📴 UI server shutting down...');
    server.close(() => {
        console.log('✅ UI server stopped');
        process.exit(0);
    });
});
