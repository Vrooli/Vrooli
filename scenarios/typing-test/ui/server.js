const express = require('express');
const cors = require('cors');
const path = require('path');
const http = require('http');
const { initIframeBridgeChild } = require('@vrooli/iframe-bridge/child');

if (typeof window !== 'undefined' && window.parent !== window) {
  initIframeBridgeChild({ appId: 'typing-test-ui' });
}

const app = express();
const PORT = process.env.UI_PORT || process.env.PORT;
const API_PORT = process.env.API_PORT || '9200';
const API_URL = `http://localhost:${API_PORT}`;

app.use(cors());
app.use(express.static(__dirname));

// Serve the main page
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check with API connectivity
app.get('/health', async (req, res) => {
    let apiConnected = false;
    let apiLatency = null;
    let apiError = null;
    const lastCheck = new Date().toISOString();

    try {
        const start = Date.now();
        await new Promise((resolve, reject) => {
            const healthReq = http.get(`${API_URL}/health`, { timeout: 2000 }, (apiRes) => {
                if (apiRes.statusCode === 200) {
                    apiConnected = true;
                    apiLatency = Date.now() - start;
                    resolve();
                } else {
                    reject(new Error(`API returned status ${apiRes.statusCode}`));
                }
            });
            healthReq.on('error', reject);
            healthReq.on('timeout', () => {
                healthReq.destroy();
                reject(new Error('Request timeout'));
            });
        });
    } catch (error) {
        apiError = {
            code: error.code || 'API_CHECK_FAILED',
            message: error.message,
            category: 'network',
            retryable: true
        };
    }

    const status = apiConnected ? 'healthy' : 'degraded';

    res.json({
        status,
        service: 'typing-test-ui',
        timestamp: new Date().toISOString(),
        readiness: apiConnected,
        api_connectivity: {
            connected: apiConnected,
            api_url: `${API_URL}/health`,
            last_check: lastCheck,
            latency_ms: apiLatency,
            error: apiError
        }
    });
});

app.listen(PORT, () => {
    console.log(`🎮 Typing Test UI running on http://localhost:${PORT}`);
    console.log(`📡 API URL configured as: ${API_URL}`);
});
