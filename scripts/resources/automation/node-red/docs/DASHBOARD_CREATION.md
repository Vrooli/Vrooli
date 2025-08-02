# Node-RED Dashboard Creation Guide

Comprehensive guide for creating real-time AI monitoring dashboards and interactive interfaces using Node-RED's dashboard capabilities.

## 🎯 Overview

Node-RED provides powerful **real-time dashboard capabilities** through the built-in `node-red-dashboard` module. These dashboards excel at **live monitoring, event-driven interactions**, and **real-time AI workflow visualization** - perfect for system monitoring, IoT integration, and rapid prototyping of AI interfaces.

## 🏗️ Dashboard Architecture

### **Real-time Data Flow**
```javascript
// Dashboard Architecture
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   AI Resources  │    │  Node-RED Flow   │    │  Dashboard UI   │
│                 │    │                  │    │                 │
│ - Ollama        │───▶│ - Data Collection│───▶│ - Live Metrics  │
│ - Whisper       │    │ - Processing     │    │ - Status Panels │
│ - ComfyUI       │    │ - Formatting     │    │ - Control Forms │
│ - Agent-S2      │    │ - WebSocket      │    │ - Real-time Logs│
└─────────────────┘    └──────────────────┘    └─────────────────┘
        ▲                        │                        │
        │                        ▼                        │
        └────────────────── Trigger Actions ◄─────────────┘
```

### **Dashboard Access**
- **Editor Interface**: `http://localhost:1880/` (flow development)
- **Dashboard Interface**: `http://localhost:1880/ui/` (user interface)
- **WebSocket Communication**: `ws://localhost:1880/comms` (real-time updates)

## 🎨 UI Components for AI Dashboards

### **Display Components**
```javascript
// Resource Status Panel
{
  type: "ui_template",
  name: "AI Resource Status",
  template: `
    <div class="resource-grid">
      <div class="resource-card" ng-class="msg.payload.ollama ? 'online' : 'offline'">
        <h3>🧠 Ollama</h3>
        <p>{{msg.payload.ollama_models}} models loaded</p>
      </div>
      <div class="resource-card" ng-class="msg.payload.whisper ? 'online' : 'offline'">
        <h3>🎤 Whisper</h3>
        <p>{{msg.payload.whisper_status}}</p>
      </div>
      <div class="resource-card" ng-class="msg.payload.comfyui ? 'online' : 'offline'">
        <h3>🎨 ComfyUI</h3>
        <p>{{msg.payload.comfyui_queue}} jobs in queue</p>
      </div>
    </div>
  `
}

// Real-time Metrics Chart
{
  type: "ui_chart",
  name: "AI Processing Metrics",
  chartType: "line",
  xAxisType: "time",
  properties: {
    ymin: 0,
    interpolate: "linear",
    colors: ["#1f77b4", "#ff7f0e", "#2ca02c"]
  }
}

// Live Activity Log
{
  type: "ui_table",
  name: "AI Activity Log", 
  columns: [
    { field: "timestamp", title: "Time" },
    { field: "service", title: "Service" },
    { field: "action", title: "Action" },
    { field: "status", title: "Status" }
  ]
}
```

### **Control Components**
```javascript
// AI Service Controls
{
  type: "ui_form",
  name: "Ollama Query",
  fields: [
    { type: "text", name: "model", label: "Model", options: ["llama3.1:8b", "codellama"] },
    { type: "textarea", name: "prompt", label: "Prompt" },
    { type: "number", name: "temperature", label: "Temperature", min: 0, max: 2, step: 0.1 }
  ]
}

// File Upload for AI Processing
{
  type: "ui_upload",
  name: "Audio Upload",
  accept: ".wav,.mp3,.m4a",
  multiple: false,
  swallowNewlines: true
}

// Emergency Controls
{
  type: "ui_button",
  name: "Emergency Stop",
  color: "red",
  bgcolor: "#ff0000",
  payload: { action: "emergency_stop", timestamp: new Date() }
}
```

## 🤖 Multi-Modal AI Dashboard Implementation

### **Phase 1: Resource Monitoring Dashboard**

**1. Health Check Flow**
```javascript
// health-monitor-flow.json
[
  {
    "id": "health-inject",
    "type": "inject",
    "name": "Health Check Timer",
    "props": [{"p": "payload"}, {"p": "topic", "vt": "str"}],
    "repeat": "10",
    "crontab": "",
    "once": true,
    "payload": "",
    "topic": "health-check"
  },
  {
    "id": "check-ollama",
    "type": "http request",
    "name": "Check Ollama",
    "method": "GET",
    "ret": "obj",
    "url": "http://localhost:11434/api/tags",
    "timeout": 5000
  },
  {
    "id": "check-whisper", 
    "type": "http request",
    "name": "Check Whisper",
    "method": "GET", 
    "ret": "obj",
    "url": "http://localhost:8090/health",
    "timeout": 5000
  },
  {
    "id": "format-status",
    "type": "function",
    "name": "Format Status",
    "func": `
      const status = {
        timestamp: new Date(),
        ollama: msg.payload.models ? msg.payload.models.length : 0,
        whisper: msg.statusCode === 200,
        comfyui: false, // Add ComfyUI check
        agent_s2: false // Add Agent-S2 check
      };
      
      return { payload: status };
    `
  },
  {
    "id": "status-dashboard",
    "type": "ui_template",
    "name": "Status Display",
    "template": "<!-- Status display template -->"
  }
]
```

**2. AI Processing Dashboard**
```javascript
// ai-processing-flow.json  
[
  {
    "id": "audio-upload",
    "type": "ui_upload",
    "name": "Upload Audio",
    "accept": ".wav,.mp3,.m4a"
  },
  {
    "id": "process-audio",
    "type": "function",
    "name": "Process with Whisper",
    "func": `
      // Send audio to Whisper API
      const formData = new FormData();
      formData.append('audio', msg.payload, 'audio.wav');
      
      fetch('http://localhost:8090/transcribe', {
        method: 'POST',
        body: formData
      })
      .then(response => response.json())
      .then(data => {
        node.send({ 
          payload: {
            text: data.text,
            language: data.language,
            timestamp: new Date()
          }
        });
      });
    `
  },
  {
    "id": "analyze-intent",
    "type": "function", 
    "name": "Analyze with Ollama",
    "func": `
      // Send transcription to Ollama for intent analysis
      const prompt = \`Analyze this voice command: "\${msg.payload.text}"\`;
      
      fetch('http://localhost:11434/api/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: 'llama3.1:8b',
          prompt: prompt,
          stream: false
        })
      })
      .then(response => response.json())
      .then(data => {
        node.send({
          payload: {
            original_text: msg.payload.text,
            analysis: data.response,
            timestamp: new Date()
          }
        });
      });
    `
  },
  {
    "id": "results-display",
    "type": "ui_template",
    "name": "Results Panel",
    "template": `
      <div class="results-panel">
        <h3>Processing Results</h3>
        <div class="result-item">
          <strong>Original Text:</strong> {{msg.payload.original_text}}
        </div>
        <div class="result-item">
          <strong>AI Analysis:</strong> {{msg.payload.analysis}}
        </div>
        <div class="timestamp">
          Processed: {{msg.payload.timestamp | date:'short'}}
        </div>
      </div>
    `
  }
]
```

### **Phase 2: Interactive Control Interface**

**1. AI Service Control Panel**
```javascript
// control-panel-flow.json
[
  {
    "id": "model-selector",
    "type": "ui_dropdown",
    "name": "Select Model",
    "options": [
      { "label": "Llama 3.1 8B", "value": "llama3.1:8b" },
      { "label": "Code Llama", "value": "codellama" },
      { "label": "Mistral 7B", "value": "mistral:7b" }
    ]
  },
  {
    "id": "prompt-input",
    "type": "ui_form",
    "name": "AI Query Form",
    "fields": [
      { "type": "textarea", "name": "prompt", "label": "Enter your prompt:" },
      { "type": "number", "name": "temperature", "label": "Temperature", "min": 0, "max": 2 },
      { "type": "number", "name": "max_tokens", "label": "Max Tokens", "min": 1, "max": 4000 }
    ]
  },
  {
    "id": "execute-query",
    "type": "function",
    "name": "Execute AI Query",
    "func": `
      const query = {
        model: flow.get('selected_model') || 'llama3.1:8b',
        prompt: msg.payload.prompt,
        temperature: msg.payload.temperature || 0.7,
        max_tokens: msg.payload.max_tokens || 1000,
        stream: false
      };
      
      // Update processing status
      node.send([
        { payload: { status: 'Processing...', timestamp: new Date() }},
        { payload: query }
      ]);
    `
  },
  {
    "id": "processing-status",
    "type": "ui_text",
    "name": "Processing Status"
  }
]
```

**2. Real-time Monitoring Components**
```javascript
// monitoring-flow.json
[
  {
    "id": "metrics-collector",
    "type": "inject",
    "name": "Collect Metrics",
    "repeat": "5",
    "payload": "",
    "topic": "metrics"
  },
  {
    "id": "resource-metrics",
    "type": "function",
    "name": "Get Resource Metrics", 
    "func": `
      // Collect system and AI service metrics
      const metrics = {
        timestamp: Date.now(),
        ollama_requests: flow.get('ollama_count') || 0,
        whisper_transcriptions: flow.get('whisper_count') || 0,
        comfyui_generations: flow.get('comfyui_count') || 0,
        agent_s2_actions: flow.get('agent_count') || 0,
        system_load: Math.random() * 100, // Replace with real metrics
        memory_usage: Math.random() * 100
      };
      
      return { payload: metrics };
    `
  },
  {
    "id": "metrics-chart",
    "type": "ui_chart",
    "name": "AI Usage Metrics",
    "chartType": "line",
    "xAxisType": "time"
  },
  {
    "id": "system-gauges",
    "type": "ui_gauge",
    "name": "System Load",
    "min": 0,
    "max": 100,
    "colors": ["green", "yellow", "red"],
    "segments": [50, 80, 100]
  }
]
```

### **Phase 3: Advanced Dashboard Features**

**1. Multi-Tab Dashboard Layout**
```javascript
// Dashboard layout configuration
const dashboardLayout = {
  "tabs": [
    {
      "name": "System Overview",
      "groups": [
        { "name": "Resource Status", "width": 6 },
        { "name": "Performance Metrics", "width": 6 },
        { "name": "Recent Activity", "width": 12 }
      ]
    },
    {
      "name": "AI Processing",
      "groups": [
        { "name": "Input Controls", "width": 4 },
        { "name": "Processing Status", "width": 4 },
        { "name": "Results Display", "width": 4 }
      ]
    },
    {
      "name": "Automation",
      "groups": [
        { "name": "Agent Controls", "width": 6 },
        { "name": "Automation Log", "width": 6 }
      ]
    }
  ]
}
```

**2. Custom CSS for AI Theme**
```css
/* Dashboard styling for AI interface */
.resource-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 10px;
  padding: 10px;
}

.resource-card {
  padding: 15px;
  border-radius: 8px;
  border: 2px solid #ccc;
  transition: all 0.3s ease;
}

.resource-card.online {
  border-color: #4CAF50;
  background-color: #f1f8e9;
}

.resource-card.offline {
  border-color: #f44336;
  background-color: #ffebee;
}

.ai-processing-status {
  background: linear-gradient(45deg, #1e3c72, #2a5298);
  color: white;
  padding: 20px;
  border-radius: 10px;
}

.results-panel {
  background-color: #f5f5f5;
  padding: 15px;
  border-radius: 8px;
  margin: 10px 0;
}

.timestamp {
  color: #666;
  font-size: 0.9em;
  text-align: right;
  margin-top: 10px;
}
```

## 🚀 Advanced Dashboard Patterns

### **1. Event-Driven AI Workflows**
```javascript
// event-driven-ai-flow.json
[
  {
    "id": "event-listener",
    "type": "websocket in",
    "name": "AI Events",
    "server": "ai-event-server"
  },
  {
    "id": "event-router",
    "type": "switch",
    "name": "Route AI Events",
    "property": "payload.event_type",
    "rules": [
      { "t": "eq", "v": "audio_uploaded" },
      { "t": "eq", "v": "transcription_complete" },
      { "t": "eq", "v": "image_generated" },
      { "t": "eq", "v": "automation_triggered" }
    ]
  },
  {
    "id": "update-dashboard",
    "type": "function",
    "name": "Update Dashboard",
    "func": `
      // Update appropriate dashboard components based on event
      const event = msg.payload;
      
      switch(event.event_type) {
        case 'audio_uploaded':
          flow.set('current_audio', event.data);
          break;
        case 'transcription_complete':
          flow.set('transcription', event.data.text);
          break;
        case 'image_generated':
          flow.set('generated_images', event.data.images);
          break;
      }
      
      // Trigger dashboard refresh
      return { payload: { refresh: true, data: event.data }};
    `
  }
]
```

### **2. Collaborative Dashboard Features**
```javascript
// collaborative-dashboard-flow.json
[
  {
    "id": "user-session",
    "type": "function",
    "name": "Track User Sessions",
    "func": `
      // Track multiple users using the dashboard
      const sessions = flow.get('active_sessions') || {};
      const userId = msg.req.session.id;
      
      sessions[userId] = {
        timestamp: Date.now(),
        ip: msg.req.ip,
        userAgent: msg.req.headers['user-agent']
      };
      
      flow.set('active_sessions', sessions);
      
      // Broadcast session update to all clients
      return { payload: { active_users: Object.keys(sessions).length }};
    `
  },
  {
    "id": "shared-workspace",
    "type": "ui_template",
    "name": "Shared Workspace",
    "template": `
      <div class="shared-workspace">
        <div class="active-users">
          <span class="user-count">{{msg.payload.active_users}}</span> users online
        </div>
        <div class="shared-content">
          <!-- Shared AI results and collaborative features -->
        </div>
      </div>
    `
  }
]
```

## 📊 Dashboard Performance Optimization

### **1. Efficient Data Updates**
```javascript
// Optimized data flow for high-frequency updates
const optimizeDataFlow = {
  "throttle_updates": {
    "type": "delay",
    "pauseType": "timed", 
    "timeout": "1000", // Limit updates to 1 per second
    "timeoutUnits": "milliseconds"
  },
  
  "batch_processing": {
    "type": "function",
    "name": "Batch AI Requests",
    "func": `
      // Batch multiple AI requests for efficiency
      const batch = context.get('pending_requests') || [];
      batch.push(msg.payload);
      
      if (batch.length >= 5) {
        context.set('pending_requests', []);
        return { payload: { batch: batch }};
      } else {
        context.set('pending_requests', batch);
        return null; // Wait for more requests
      }
    `
  },
  
  "cache_results": {
    "type": "function", 
    "name": "Cache AI Results",
    "func": `
      // Cache AI results to avoid repeated processing
      const cache = flow.get('ai_cache') || {};
      const cacheKey = Buffer.from(msg.payload.prompt).toString('base64');
      
      if (cache[cacheKey] && (Date.now() - cache[cacheKey].timestamp < 300000)) {
        // Return cached result if less than 5 minutes old
        return { payload: cache[cacheKey].result };
      }
      
      // Process and cache new result
      // ... AI processing logic ...
    `
  }
}
```

### **2. Memory Management**
```javascript
// Memory-efficient dashboard operations
const memoryManagement = {
  "cleanup_old_data": {
    "type": "inject",
    "name": "Cleanup Timer",
    "repeat": "300", // Every 5 minutes
    "payload": "",
    "topic": "cleanup"
  },
  
  "memory_cleanup": {
    "type": "function",
    "name": "Clean Old Data",
    "func": `
      // Remove old logs and temporary data
      const maxAge = 3600000; // 1 hour
      const now = Date.now();
      
      const logs = flow.get('activity_logs') || [];
      const cleanLogs = logs.filter(log => (now - log.timestamp) < maxAge);
      flow.set('activity_logs', cleanLogs);
      
      // Clear old cache entries
      const cache = flow.get('ai_cache') || {};
      Object.keys(cache).forEach(key => {
        if ((now - cache[key].timestamp) > maxAge) {
          delete cache[key];
        }
      });
      flow.set('ai_cache', cache);
    `
  }
}
```

## 🔗 Integration with Windmill

### **Triggering Windmill Workflows from Node-RED**
```javascript
// windmill-integration-flow.json
[
  {
    "id": "trigger-windmill",
    "type": "http request",
    "name": "Execute Windmill Workflow",
    "method": "POST",
    "url": "http://localhost:5681/api/jobs/run",
    "headers": {
      "Authorization": "Bearer {{windmill_token}}",
      "Content-Type": "application/json"
    }
  },
  {
    "id": "format-windmill-request",
    "type": "function", 
    "name": "Format Request",
    "func": `
      // Format request for Windmill API
      const request = {
        script: 'multi_modal_assistant/process_workflow',
        args: {
          audio_data: msg.payload.audio,
          user_preferences: msg.payload.preferences,
          session_id: msg.payload.session_id
        }
      };
      
      return { payload: request };
    `
  },
  {
    "id": "handle-windmill-response",
    "type": "function",
    "name": "Process Response", 
    "func": `
      // Handle Windmill workflow results
      const result = msg.payload;
      
      if (result.success) {
        // Update dashboard with results
        flow.set('last_windmill_result', result.data);
        
        // Send to dashboard display
        return { 
          payload: {
            type: 'windmill_success',
            data: result.data,
            timestamp: new Date()
          }
        };
      } else {
        // Handle error
        return {
          payload: {
            type: 'windmill_error', 
            error: result.error,
            timestamp: new Date()
          }
        };
      }
    `
  }
]
```

## 🎯 Business Dashboard Applications

### **AI Service Monitoring ($2K-5K)**
- Resource health dashboards
- Performance monitoring
- Usage analytics
- Alert management

### **Real-time AI Workflows ($3K-8K)**
- Live processing dashboards
- Interactive AI controls
- Multi-user collaboration
- Event-driven automation

### **IoT + AI Integration ($5K-12K)**
- Sensor data + AI analysis
- Real-time decision making
- Automated responses
- Predictive maintenance

## 📚 Dashboard Templates

### **Quick Start Templates**
```bash
# Import dashboard templates
cd /path/to/node-red/flows
wget https://flows.nodered.org/flow/ai-monitoring-dashboard
wget https://flows.nodered.org/flow/multi-modal-interface
wget https://flows.nodered.org/flow/resource-health-monitor

# Load into Node-RED
curl -X POST http://localhost:1880/flows -d @ai-monitoring-dashboard.json
```

### **Custom Dashboard Generator**
```javascript
// Generate dashboard from AI service configuration
function generateDashboard(services) {
  const dashboardFlow = [];
  
  services.forEach(service => {
    // Add monitoring components for each service
    dashboardFlow.push(createMonitoringComponent(service));
    dashboardFlow.push(createControlComponent(service));
    dashboardFlow.push(createStatusDisplay(service));
  });
  
  return dashboardFlow;
}
```

## 📖 Additional Resources

- **Node-RED Dashboard Docs**: [nodered.org/docs/user-guide/dashboard](https://nodered.org/docs/user-guide/dashboard)
- **Flow Examples**: [flows/ai-dashboards/](../flows/ai-dashboards/)
- **Custom Nodes**: [lib/dashboard-nodes/](../lib/dashboard-nodes/)
- **Integration Guide**: [API.md](API.md)
- **Performance Tuning**: [CONFIGURATION.md](CONFIGURATION.md)

---

**🎛️ Node-RED's dashboard capabilities provide the perfect foundation for real-time AI monitoring and interactive control interfaces that complement Windmill's professional application development approach.**