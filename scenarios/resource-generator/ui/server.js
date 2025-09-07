const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.UI_PORT;
const API_PORT = process.env.API_PORT;
const API_HOST = process.env.API_HOST || 'localhost';

if (!PORT) {
    console.error('Error: UI_PORT environment variable is not set');
    process.exit(1);
}

if (!API_PORT) {
    console.error('Error: API_PORT environment variable is not set');
    process.exit(1);
}

// Serve static files
app.use(express.static(path.join(__dirname, 'public')));

// Main page
app.get('/', (req, res) => {
    res.send(`
<!DOCTYPE html>
<html>
<head>
    <title>Resource Generator</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #4CAF50;
            padding-bottom: 10px;
        }
        .container {
            background: white;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        .form-group {
            margin-bottom: 15px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
            color: #555;
        }
        input, select, textarea {
            width: 100%;
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
        }
        textarea {
            min-height: 100px;
            resize: vertical;
        }
        button {
            background: #4CAF50;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }
        button:hover {
            background: #45a049;
        }
        .status {
            margin-top: 20px;
            padding: 10px;
            border-radius: 4px;
            background: #e3f2fd;
        }
        .queue-section {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 15px;
            margin-top: 15px;
        }
        .queue-box {
            background: #f9f9f9;
            padding: 15px;
            border-radius: 4px;
            border-left: 4px solid #2196F3;
        }
        .queue-box h3 {
            margin-top: 0;
            color: #333;
        }
        .queue-item {
            background: white;
            padding: 5px;
            margin: 5px 0;
            border-radius: 3px;
            font-size: 12px;
        }
        .template-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 15px;
            margin-top: 15px;
        }
        .template-card {
            background: white;
            padding: 15px;
            border-radius: 8px;
            border: 1px solid #e0e0e0;
            cursor: pointer;
            transition: all 0.3s;
        }
        .template-card:hover {
            box-shadow: 0 4px 8px rgba(0,0,0,0.1);
            transform: translateY(-2px);
        }
        .template-card.selected {
            border: 2px solid #4CAF50;
            background: #f1f8e9;
        }
        .template-card h4 {
            margin: 0 0 8px 0;
            color: #2196F3;
        }
        .template-card p {
            margin: 0;
            color: #666;
            font-size: 14px;
        }
        .category-badge {
            display: inline-block;
            padding: 2px 8px;
            background: #e3f2fd;
            color: #1976d2;
            border-radius: 12px;
            font-size: 11px;
            margin-top: 8px;
        }
        .processing-status {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            font-weight: bold;
            margin-left: 10px;
        }
        .processing-status.active {
            background: #c8e6c9;
            color: #2e7d32;
        }
        .processing-status.inactive {
            background: #ffccbc;
            color: #d84315;
        }
    </style>
</head>
<body>
    <h1>🚀 Resource Generator</h1>
    
    <div class="container">
        <h2>Create New Resource</h2>
        <form id="createForm">
            <div class="form-group">
                <label for="name">Resource Name *</label>
                <input type="text" id="name" name="name" required placeholder="e.g., matrix-synapse">
            </div>
            
            <div class="form-group">
                <label>Template *</label>
                <div class="template-grid" id="templateGrid">
                    <!-- Templates will be loaded here -->
                </div>
                <input type="hidden" id="template" name="template" required>
            </div>
            
            <div class="form-group">
                <label for="type">Resource Type</label>
                <select id="type" name="type">
                    <option value="resource">Resource</option>
                    <option value="service">Service</option>
                    <option value="tool">Tool</option>
                    <option value="integration">Integration</option>
                </select>
            </div>
            
            <div class="form-group">
                <label for="priority">Priority</label>
                <select id="priority" name="priority">
                    <option value="low">Low</option>
                    <option value="medium" selected>Medium</option>
                    <option value="high">High</option>
                    <option value="critical">Critical</option>
                </select>
            </div>
            
            <div class="form-group">
                <label for="description">Description</label>
                <textarea id="description" name="description" placeholder="Describe the resource requirements and features..."></textarea>
            </div>
            
            <button type="submit">Generate Resource</button>
        </form>
        
        <div id="createStatus" class="status" style="display: none;"></div>
    </div>
    
    <div class="container">
        <h2>Queue Status 
            <span id="processingStatus" class="processing-status">Loading...</span>
        </h2>
        <div class="queue-section" id="queueStatus">
            <div class="queue-box">
                <h3>📋 Pending</h3>
                <div id="pendingQueue"></div>
            </div>
            <div class="queue-box">
                <h3>⚙️ In Progress</h3>
                <div id="inProgressQueue"></div>
            </div>
            <div class="queue-box">
                <h3>✅ Completed</h3>
                <div id="completedQueue"></div>
            </div>
            <div class="queue-box">
                <h3>❌ Failed</h3>
                <div id="failedQueue"></div>
            </div>
        </div>
    </div>

    <script>
        const API_BASE = 'http://${API_HOST}:${API_PORT}';
        
        // Load templates on page load
        async function loadTemplates() {
            try {
                const response = await fetch(\`\${API_BASE}/api/templates\`);
                const data = await response.json();
                
                if (data.success && data.data) {
                    const grid = document.getElementById('templateGrid');
                    grid.innerHTML = data.data.map(template => \`
                        <div class="template-card" data-id="\${template.id}" onclick="selectTemplate('\${template.id}')">
                            <h4>\${template.name}</h4>
                            <p>\${template.description}</p>
                            <span class="category-badge">\${template.category}</span>
                        </div>
                    \`).join('');
                }
            } catch (error) {
                console.error('Failed to load templates:', error);
            }
        }
        
        // Select template
        function selectTemplate(templateId) {
            document.querySelectorAll('.template-card').forEach(card => {
                card.classList.remove('selected');
            });
            document.querySelector(\`[data-id="\${templateId}"]\`).classList.add('selected');
            document.getElementById('template').value = templateId;
        }
        
        // Load queue status
        async function loadQueueStatus() {
            try {
                const response = await fetch(\`\${API_BASE}/api/queue\`);
                const data = await response.json();
                
                if (data.success && data.data) {
                    updateQueueDisplay('pendingQueue', data.data.pending || []);
                    updateQueueDisplay('inProgressQueue', data.data.in_progress || []);
                    updateQueueDisplay('completedQueue', data.data.completed || []);
                    updateQueueDisplay('failedQueue', data.data.failed || []);
                }
                
                // Also check processing status
                const healthResponse = await fetch(\`\${API_BASE}/health\`);
                const healthData = await healthResponse.json();
                
                if (healthData.success && healthData.data) {
                    const statusEl = document.getElementById('processingStatus');
                    if (healthData.data.processing_active) {
                        statusEl.textContent = 'ACTIVE';
                        statusEl.className = 'processing-status active';
                    } else {
                        statusEl.textContent = 'PAUSED';
                        statusEl.className = 'processing-status inactive';
                    }
                }
            } catch (error) {
                console.error('Failed to load queue status:', error);
            }
        }
        
        function updateQueueDisplay(elementId, items) {
            const element = document.getElementById(elementId);
            if (items.length === 0) {
                element.innerHTML = '<div style="color: #999;">Empty</div>';
            } else {
                element.innerHTML = items.slice(0, 5).map(item => 
                    \`<div class="queue-item">\${item}</div>\`
                ).join('');
                if (items.length > 5) {
                    element.innerHTML += \`<div style="color: #666; font-style: italic;">+\${items.length - 5} more</div>\`;
                }
            }
        }
        
        // Handle form submission
        document.getElementById('createForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const formData = new FormData(e.target);
            const data = Object.fromEntries(formData);
            
            const statusDiv = document.getElementById('createStatus');
            statusDiv.style.display = 'block';
            statusDiv.innerHTML = '⏳ Queueing resource generation...';
            
            try {
                const response = await fetch(\`\${API_BASE}/api/resources/generate\`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data),
                });
                
                const result = await response.json();
                
                if (result.success) {
                    statusDiv.innerHTML = \`✅ Resource generation queued successfully!<br>
                        Queue ID: \${result.data.id}<br>
                        Resource: \${result.data.resourceName}\`;
                    e.target.reset();
                    document.querySelectorAll('.template-card').forEach(card => {
                        card.classList.remove('selected');
                    });
                    loadQueueStatus();
                } else {
                    statusDiv.innerHTML = \`❌ Failed to queue generation: \${result.error || 'Unknown error'}\`;
                }
            } catch (error) {
                statusDiv.innerHTML = \`❌ Error: \${error.message}\`;
            }
        });
        
        // Load data on page load
        loadTemplates();
        loadQueueStatus();
        
        // Refresh queue status every 5 seconds
        setInterval(loadQueueStatus, 5000);
    </script>
</body>
</html>
    `);
});

app.listen(PORT, () => {
    console.log(`Resource Generator UI running on port ${PORT}`);
    console.log(`Connected to API at ${API_HOST}:${API_PORT}`);
});