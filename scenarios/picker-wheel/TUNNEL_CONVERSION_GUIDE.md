# Picker Wheel - Secure Tunnel Conversion Guide
*Verified HTTPS tunnel compatibility for the Picker Wheel scenario*

---

## ✅ **Tunnel Status: READY**

**Good News!** The Picker Wheel scenario is already fully tunnel-compatible. This guide documents the configuration and testing procedures.

---

## 🔧 **Current Architecture**

### **Service Ports**
- **API Server**: Port `27100` (fixed)
- **UI Server**: Port `34300` (fixed) 
- **n8n Workflows**: Managed by Vrooli resource system

### **Tunnel-Compatible Features**
✅ **API Proxy**: UI server proxies all API calls to same origin  
✅ **Same-Origin URLs**: JavaScript uses `window.location.protocol/host`  
✅ **n8n Integration**: Webhook proxy for workflow communication  
✅ **Error Handling**: Fallback to local storage when API unavailable  
✅ **Fixed Ports**: No dynamic port ranges that could cause conflicts  

---

## 🌐 **How It Works**

### **UI Server Proxy Configuration** (`ui/server.js`)
```javascript
// API endpoints proxy - routes /api/* to API server
app.use('/api', (req, res) => {
    const fullApiPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, API_PORT, fullApiPath);
});

// n8n webhook proxy - routes /webhook/* to n8n
app.use('/webhook', (req, res) => {
    const webhookPath = '/webhook' + (req.url.startsWith('/') ? req.url : '/' + req.url);
    proxyToApi(req, res, N8N_PORT, webhookPath);
});
```

### **Client-Side URL Construction** (`ui/script.js`)
```javascript
// Same-origin URLs automatically inherit tunnel's HTTPS
const API_BASE = `${window.location.protocol}//${window.location.host}/api`;
const N8N_BASE = `${window.location.protocol}//${window.location.host}/webhook`;

// Example API calls
await fetch(`${API_BASE}/wheels`, { method: 'POST', ... });
await fetch(`${N8N_BASE}/wheel/spin`, { method: 'POST', ... });
```

---

## 🧪 **Testing Procedures**

### **Local Testing**
```bash
# 1. Start picker-wheel scenario
vrooli develop picker-wheel

# 2. Test API proxy endpoints
curl -s "http://localhost:34300/api/health"
curl -s "http://localhost:34300/webhook/health"

# 3. Test wheel functionality
curl -X POST "http://localhost:34300/api/spin" \
  -H "Content-Type: application/json" \
  -d '{"wheel_id":"yes-or-no","session_id":"test"}'

# 4. Test UI in browser
# Visit: http://localhost:34300
# Try spinning wheel, saving custom wheels, checking history
```

### **Tunnel Testing**
```bash
# 1. Access through tunnel
curl -s "https://picker-wheel.tunnel.com/" | grep -q "Picker Wheel"

# 2. Test API endpoints through tunnel
curl -s "https://picker-wheel.tunnel.com/api/health"
curl -s "https://picker-wheel.tunnel.com/webhook/health"

# 3. Test wheel spinning through tunnel
curl -X POST "https://picker-wheel.tunnel.com/api/spin" \
  -H "Content-Type: application/json" \
  -d '{"wheel_id":"dinner","session_id":"tunnel-test"}'

# 4. Test n8n webhook integration
curl -X POST "https://picker-wheel.tunnel.com/webhook/wheel/spin" \
  -H "Content-Type: application/json" \
  -d '{"wheel_id":"yes-or-no","options":[{"label":"Yes","weight":1}]}'
```

### **Browser Testing Checklist**
- [ ] Visit `https://picker-wheel.tunnel.com/` loads without errors
- [ ] Spin wheel works and shows results
- [ ] Custom wheel creation and saving works
- [ ] History tracking updates correctly
- [ ] Settings persist across page reloads
- [ ] No mixed content warnings in console
- [ ] n8n workflow integration functions properly

---

## 📋 **Key Implementation Details**

### **Why It Works**
1. **Unified Origin**: All requests go through the UI server at port 34300
2. **Smart Proxying**: API calls are forwarded to the actual API server (port 27100)
3. **n8n Integration**: Webhook calls proxy to n8n for intelligent suggestions
4. **Graceful Degradation**: Falls back to local functionality if API unavailable

### **Critical Files**
- `ui/server.js`: Contains proxy implementation
- `ui/script.js`: Uses same-origin URLs for all API calls
- `.vrooli/service.json`: Defines fixed port configuration

### **Proxy Routes**
- `/api/*` → `localhost:27100/api/*` (API server)
- `/webhook/*` → `localhost:${N8N_PORT}/webhook/*` (n8n workflows)
- `/health` → Direct UI server health check
- `/` → Serves static UI files

---

## 🚀 **Quick Start for Tunnel Testing**

```bash
# Start the scenario
vrooli develop picker-wheel

# Verify local functionality
open http://localhost:34300

# Test through tunnel (replace with actual tunnel URL)
open https://picker-wheel.tunnel.com

# Monitor logs for debugging
tail -f .vrooli/logs/vrooli.develop.picker-wheel.log
```

---

## 🎯 **Success Indicators**

### **Visual Confirmation**
- Wheel spins smoothly without console errors
- Results appear after animation completes
- Custom wheels can be created and saved
- History tab shows previous spins
- Settings changes take effect immediately

### **Technical Validation**
- Network tab shows all requests go to same origin
- No CORS or mixed content errors in console
- API calls return proper JSON responses
- n8n webhook integration works for intelligent suggestions

---

## 🔍 **Troubleshooting**

### **Common Issues**
**Wheel doesn't spin**: Check browser console for API errors  
**History not saving**: Verify API server is responding at `/api/wheels`  
**n8n features broken**: Check n8n resource is running and accessible  

### **Debug Commands**
```bash
# Check service status
vrooli status picker-wheel

# Test API directly
curl http://localhost:27100/health

# Check n8n connectivity
curl http://localhost:5678/api/v1/info

# View logs
tail -f .vrooli/logs/vrooli.develop.picker-wheel.log
```

---

**The Picker Wheel scenario is fully tunnel-ready with robust proxy architecture, graceful degradation, and comprehensive error handling.**