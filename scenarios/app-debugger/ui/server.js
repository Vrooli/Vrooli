import express from 'express'
import cors from 'cors'
import path from 'path'
import { fileURLToPath } from 'url'
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const app = express()
const PORT = process.env.UI_PORT || process.env.PORT
const BRIDGE_FLAG = '__appDebuggerBridgeInitialized'

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

  if (window.parent !== window) {
    initIframeBridgeChild({ appId: 'app-debugger' })
    window[BRIDGE_FLAG] = true
  }
}

try {
  initializeIframeBridge()
} catch (error) {
  console.error('[AppDebugger] Failed to initialize iframe bridge', error)
}

// Enable CORS for API communication
app.use(cors())

// Serve static files
app.use(express.static(__dirname))

// Main route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

// Health check endpoint
app.get('/health', (req, res) => {
    res.json({ 
        status: 'healthy', 
        service: 'app-debugger-ui',
        timestamp: new Date().toISOString()
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`App Debugger UI running on port ${PORT}`);
    console.log(`Access the debugger at http://localhost:${PORT}`);
});
