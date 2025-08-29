# Secure Tunnel Conversion Guide
*Converting Vrooli Scenarios for HTTPS Tunnel Compatibility*

---

## 🎯 **Problem Overview**

**Root Cause**: Modern browsers enforce **Mixed Content Policy** - HTTPS pages cannot make HTTP requests to different origins/ports.

**Tunnel Scenario**: 
- UI served over HTTPS: `https://scenario.tunnel.com/`
- API on different port: `http://scenario.tunnel.com:8080/api/data` ❌ **BLOCKED**

**Result**: JavaScript fetch() calls fail silently or with CORS errors.

---

## 🔍 **Step 1: Scenario Analysis**

### **A. Identify Architecture Pattern**

Run this diagnostic to understand your scenario's architecture:

```bash
# 1. Find all service.json files
find . -name "service.json" -exec echo "=== {} ===" \; -exec cat {} \;

# 2. Find all package.json files (Node.js servers) 
find . -name "package.json" -exec echo "=== {} ===" \; -exec cat {} \;

# 3. Find all server files
find . -name "*server*" -type f

# 4. Check for port configurations
grep -r "PORT\|port.*:" . --include="*.js" --include="*.json" --include="*.go" --include="*.py"

# 5. Find JavaScript files making API calls
grep -r "fetch\|axios\|XMLHttpRequest" . --include="*.js"
```

### **B. Classify Your Scenario**

**Pattern 1: Single UI Server** ✅ *Easy*
- One web server serving static files
- No API calls or only same-origin calls
- **Action**: Usually no changes needed

**Pattern 2: UI + Separate API Server** ⚠️ *Medium*
- Frontend (React/Vue/vanilla JS) on port A
- Backend API on port B  
- **Action**: Implement API proxy (this guide's main focus)

**Pattern 3: Multi-Service Architecture** 🔴 *Complex*
- Multiple APIs, microservices
- Inter-service communication
- **Action**: Gateway pattern or service mesh

**Pattern 4: WebSocket/Real-time Services** 🔴 *Complex*
- WebSocket connections
- Server-sent events
- **Action**: WebSocket proxy + connection handling

---

## 🛠️ **Step 2: Implementation by Pattern**

## **Pattern 2: UI + API Server** (Most Common)

This is the same pattern we fixed in system-monitor. Here's the reusable solution:

### **A. Identify the Files to Modify**

1. **UI Server File**: Usually `server.js`, `app.js`, or `index.js` in UI directory
2. **Client JavaScript**: Files containing `fetch()`, `axios.get()`, or similar API calls
3. **Configuration**: Any files with hardcoded API URLs

### **B. Step-by-Step Implementation**

#### **1. Update UI Server (Add API Proxy)**

**Before:**
```javascript
// Basic Express server
const express = require('express');
const app = express();
app.use(express.static('.'));
app.listen(3000);
```

**After:**
```javascript
const express = require('express');
const path = require('path');
const http = require('http');  // Add for proxy

const app = express();
const PORT = process.env.PORT || process.env.UI_PORT || 3000;
const API_PORT = process.env.API_PORT || process.env.SERVICE_PORT || 8080;

// === API PROXY IMPLEMENTATION ===
function proxyToApi(req, res, apiPath) {
    const options = {
        hostname: 'localhost',
        port: API_PORT,
        path: apiPath || req.url,
        method: req.method,
        headers: {
            ...req.headers,
            host: `localhost:${API_PORT}`
        }
    };
    
    console.log(`[PROXY] ${req.method} ${req.url} -> http://localhost:${API_PORT}${options.path}`);
    
    const proxyReq = http.request(options, (proxyRes) => {
        // Copy status and headers
        res.status(proxyRes.statusCode);
        Object.keys(proxyRes.headers).forEach(key => {
            res.setHeader(key, proxyRes.headers[key]);
        });
        
        // Stream response back
        proxyRes.pipe(res);
    });
    
    proxyReq.on('error', (err) => {
        console.error('Proxy error:', err.message);
        res.status(502).json({ 
            error: 'API server unavailable', 
            details: err.message,
            target: `http://localhost:${API_PORT}${options.path}`
        });
    });
    
    // Handle request body for POST/PUT requests
    if (req.method !== 'GET' && req.method !== 'HEAD') {
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}

// === PROXY ROUTES ===
// Health endpoint proxy (if exists)
app.use('/health', (req, res) => {
    proxyToApi(req, res, '/health');
});

// API endpoints proxy
app.use('/api', (req, res) => {
    // Express strips /api prefix, so add it back
    const fullApiPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, fullApiPath);
});

// === STATIC FILE SERVING ===
app.use(express.static(__dirname, { 
    index: false,  // Disable auto-index serving
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Cache-Control', 'no-cache');  // Prevent JS caching issues
        }
    }
}));

// Main page route
app.get('/', (req, res) => {
    res.sendFile(path.join(__dirname, 'index.html'));
});

app.listen(PORT, () => {
    console.log(`UI Server running on http://localhost:${PORT}`);
    console.log(`API proxy configured for http://localhost:${API_PORT}`);
});
```

#### **2. Update Client JavaScript (Same-Origin URLs)**

**Find API URL construction patterns:**
```bash
# Common patterns to find:
grep -r "http://.*:\${.*}" . --include="*.js"
grep -r "fetch.*localhost:" . --include="*.js" 
grep -r "axios.*localhost:" . --include="*.js"
grep -r "baseURL.*http" . --include="*.js"
```

**Before:**
```javascript
// Hard-coded API URLs
const API_URL = 'http://localhost:8080';
fetch('http://localhost:8080/api/data');

// Dynamic port URLs  
const baseUrl = `http://${window.location.hostname}:8080`;
fetch(`${baseUrl}/api/users`);

// Environment-based URLs
const apiUrl = process.env.API_URL || 'http://localhost:8080';
```

**After:**
```javascript
// Same-origin URLs (automatic HTTPS when page is HTTPS)
const baseUrl = `${window.location.protocol}//${window.location.host}`;
fetch(`${baseUrl}/api/data`);

// Or simple relative URLs
fetch('/api/users');
fetch('/api/health');

// For classes/modules:
class ApiClient {
    constructor() {
        this.baseUrl = `${window.location.protocol}//${window.location.host}`;
    }
    
    async getData() {
        const response = await fetch(`${this.baseUrl}/api/data`);
        return response.json();
    }
}
```

#### **3. Update Package Dependencies**

If using http-proxy-middleware (alternative approach):
```bash
npm install http-proxy-middleware
```

```javascript
// Alternative using middleware (may have compatibility issues)
const { createProxyMiddleware } = require('http-proxy-middleware');

app.use('/api', createProxyMiddleware({
    target: `http://localhost:${API_PORT}`,
    changeOrigin: true,
    pathRewrite: {
        '^/api': '/api'  // Keep /api prefix
    }
}));
```

---

## 🧪 **Step 3: Testing Procedure**

### **A. Local Testing**

```bash
# 1. Start API server (usually port 8080)
cd api/
./start-api-server.sh  # or node server.js, go run main.go, etc.

# 2. Start UI server (usually port 3000-3003) 
cd ui/
node server.js

# 3. Test endpoints directly
curl -s "http://localhost:3003/api/health"
curl -s "http://localhost:3003/api/data" 

# 4. Test browser functionality
# Open http://localhost:3003 and check browser console for errors
```

### **B. Tunnel Testing**

```bash
# 1. Test tunnel accessibility
curl -s "https://your-scenario.tunnel.com/" | head -10

# 2. Test API endpoints through tunnel
curl -s "https://your-scenario.tunnel.com/api/health"
curl -s "https://your-scenario.tunnel.com/api/data"

# 3. Test POST/PUT requests
curl -s -X POST "https://your-scenario.tunnel.com/api/submit" -H "Content-Type: application/json" -d '{"test": true}'

# 4. Test browser with cache clearing
# Open https://your-scenario.tunnel.com/
# Press Ctrl+Shift+R (hard refresh) or clear browser cache
# Check Network tab in DevTools for failed requests
```

### **C. Browser Testing Checklist**

1. **Hard Refresh**: `Ctrl+Shift+R` or `Cmd+Shift+R`
2. **Clear Cache**: DevTools → Application → Storage → Clear site data
3. **Private/Incognito Mode**: Test without any cached data
4. **Network Tab**: Check for failed requests or 404/405 errors
5. **Console Tab**: Look for CORS, Mixed Content, or fetch errors

---

## ⚠️ **Common Pitfalls & Solutions**

### **Pitfall 1: Browser Caching**
**Symptom**: Old HTTP URLs still being used despite code changes
**Solution**: 
```javascript
// Add cache-busting headers
app.use(express.static(__dirname, { 
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Cache-Control', 'no-cache, no-store, must-revalidate');
        }
    }
}));
```

### **Pitfall 2: Wrong Proxy Path**
**Symptom**: 404 errors for API calls through proxy
**Debug**:
```javascript
// Add detailed logging
console.log(`[PROXY] ${req.method} ${req.url} -> http://localhost:${API_PORT}${options.path}`);
console.log('[HEADERS]', req.headers);
```

**Common Issues**:
- Express strips route prefix: `/api/users` becomes `/users`
- Solution: Manually reconstruct full path

### **Pitfall 3: CORS Headers Missing**
**Symptom**: CORS errors in browser console
**Solution**: 
```javascript
// Add CORS middleware
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    
    if (req.method === 'OPTIONS') {
        res.sendStatus(200);
    } else {
        next();
    }
});
```

### **Pitfall 4: WebSocket Connections**
**Symptom**: Real-time features not working through tunnel
**Solution**:
```javascript
// WebSocket proxy (more complex)
const { createProxyMiddleware } = require('http-proxy-middleware');

app.use('/socket.io', createProxyMiddleware({
    target: `http://localhost:${API_PORT}`,
    ws: true,  // Enable WebSocket proxying
    changeOrigin: true
}));
```

### **Pitfall 5: POST Request Bodies**
**Symptom**: POST/PUT requests fail or have empty bodies
**Solution**:
```javascript
function proxyToApi(req, res, apiPath) {
    // ... options setup ...
    
    const proxyReq = http.request(options, (proxyRes) => {
        // ... response handling ...
    });
    
    // CRITICAL: Handle request body properly
    if (req.method !== 'GET' && req.method !== 'HEAD') {
        // Stream the request body
        req.pipe(proxyReq);
    } else {
        proxyReq.end();
    }
}
```

---

## 📋 **Pattern 3: Multi-Service Architecture**

For complex scenarios with multiple services:

### **Option A: Gateway Pattern**
Create a single gateway that proxies to multiple services:

```javascript
// gateway.js
app.use('/api/auth', proxyTo('http://localhost:8081'));
app.use('/api/data', proxyTo('http://localhost:8082'));
app.use('/api/files', proxyTo('http://localhost:8083'));
```

### **Option B: Service Consolidation**
Combine related services into fewer endpoints:

```javascript
// Combine auth + data APIs into single service
app.use('/api', proxyTo('http://localhost:8080'));
```

---

## 📋 **Pattern 4: WebSocket/Real-time**

For scenarios with WebSocket connections:

```javascript
const { createProxyMiddleware } = require('http-proxy-middleware');

// WebSocket proxy
app.use('/socket.io', createProxyMiddleware({
    target: `ws://localhost:${API_PORT}`,
    ws: true,
    changeOrigin: true,
    onError: (err, req, res) => {
        console.error('WebSocket proxy error:', err.message);
    }
}));

// Client-side: Use same-origin WebSocket URLs
const socket = io(`${window.location.protocol}//${window.location.host}`);
```

---

## 🚀 **Quick Conversion Checklist**

### **Phase 1: Analysis** (5-10 minutes)
- [ ] Run architecture diagnostic commands
- [ ] Identify all services and ports
- [ ] Find API call patterns in JavaScript
- [ ] Classify scenario pattern

### **Phase 2: Implementation** (15-30 minutes) 
- [ ] Add proxy function to UI server
- [ ] Add proxy routes for API endpoints
- [ ] Update client JavaScript URLs
- [ ] Add error handling and logging

### **Phase 3: Testing** (10-15 minutes)
- [ ] Test local proxy functionality
- [ ] Test tunnel endpoints with curl
- [ ] Test browser with cache clearing
- [ ] Verify real-time features work

### **Total Time**: ~30-60 minutes per scenario

---

## 📝 **Code Templates**

### **UI Server Template** (`ui/server.js`)
```javascript
const express = require('express');
const path = require('path');
const http = require('http');

const app = express();
const PORT = process.env.PORT || process.env.UI_PORT || 3003;
const API_PORT = process.env.API_PORT || process.env.SERVICE_PORT || 8080;

// Proxy function
function proxyToApi(req, res, apiPath) {
    const options = {
        hostname: 'localhost',
        port: API_PORT,
        path: apiPath || req.url,
        method: req.method,
        headers: { ...req.headers, host: `localhost:${API_PORT}` }
    };
    
    console.log(`[PROXY] ${req.method} ${req.url} -> http://localhost:${API_PORT}${options.path}`);
    
    const proxyReq = http.request(options, (proxyRes) => {
        res.status(proxyRes.statusCode);
        Object.keys(proxyRes.headers).forEach(key => {
            res.setHeader(key, proxyRes.headers[key]);
        });
        proxyRes.pipe(res);
    });
    
    proxyReq.on('error', (err) => {
        console.error('Proxy error:', err.message);
        res.status(502).json({ error: 'API server unavailable', details: err.message });
    });
    
    if (['GET', 'HEAD'].includes(req.method)) {
        proxyReq.end();
    } else {
        req.pipe(proxyReq);
    }
}

// Proxy routes
app.use('/health', (req, res) => proxyToApi(req, res, '/health'));
app.use('/api', (req, res) => {
    const fullApiPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, fullApiPath);
});

// Static files
app.use(express.static(__dirname, { 
    index: false,
    setHeaders: (res, path) => {
        if (path.endsWith('.js')) {
            res.setHeader('Cache-Control', 'no-cache');
        }
    }
}));

// Routes
app.get('/', (req, res) => res.sendFile(path.join(__dirname, 'index.html')));

app.listen(PORT, () => {
    console.log(`Server running on http://localhost:${PORT}`);
    console.log(`API proxy: http://localhost:${API_PORT}`);
});
```

### **Client JavaScript Template**
```javascript
// API Client class
class ApiClient {
    constructor() {
        // Same-origin URLs for tunnel compatibility
        this.baseUrl = `${window.location.protocol}//${window.location.host}`;
    }
    
    async request(endpoint, options = {}) {
        try {
            const response = await fetch(`${this.baseUrl}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });
            
            if (!response.ok) {
                throw new Error(`API Error: ${response.status} ${response.statusText}`);
            }
            
            return await response.json();
        } catch (error) {
            console.error(`API request failed: ${endpoint}`, error);
            throw error;
        }
    }
    
    // Convenience methods
    async get(endpoint) { return this.request(endpoint); }
    async post(endpoint, data) { 
        return this.request(endpoint, {
            method: 'POST',
            body: JSON.stringify(data)
        });
    }
    async put(endpoint, data) { 
        return this.request(endpoint, {
            method: 'PUT', 
            body: JSON.stringify(data)
        });
    }
    async delete(endpoint) { 
        return this.request(endpoint, { method: 'DELETE' });
    }
}

// Usage
const api = new ApiClient();
const data = await api.get('/api/health');
const result = await api.post('/api/submit', { key: 'value' });
```

---

## 🏆 **Success Metrics**

### **Technical Validation**
- [ ] All API calls return data when accessed via tunnel
- [ ] No mixed content errors in browser console  
- [ ] No CORS errors in browser console
- [ ] WebSocket connections (if any) work through tunnel
- [ ] File uploads/downloads work through tunnel

### **User Experience**
- [ ] Page loads completely on first visit via tunnel
- [ ] All interactive features work via tunnel
- [ ] Real-time updates (if any) work via tunnel
- [ ] Performance is acceptable (< 2x local latency)

### **Maintenance**
- [ ] Code is clean with debug files removed
- [ ] Logging provides useful debugging information
- [ ] Error handling provides meaningful user feedback
- [ ] Documentation updated with tunnel usage instructions

---

**This guide provides a systematic approach to converting Vrooli scenarios for secure tunnel compatibility. The key insight is implementing a reverse proxy pattern that consolidates multiple HTTP endpoints into a single HTTPS endpoint, eliminating mixed content violations while maintaining full functionality.**