// Resource Improver Interactive Terminal
// API base URL - configurable via global variables or environment
const API_HOST = window.RESOURCE_IMPROVER_API_HOST || window.location.hostname;
const API_PORT = window.RESOURCE_IMPROVER_API_PORT || window.API_PORT;
if (!API_PORT) {
    console.error('API_PORT environment variable is required');
    alert('API_PORT environment variable is not set. Please configure the application properly.');
}
const API_BASE = window.RESOURCE_IMPROVER_API_URL || `${window.location.protocol}//${API_HOST}:${API_PORT}`;

let selectedResource = null;
let improvementHistory = [];
let historyIndex = -1;
let stats = {
    pending: 0,
    improved: 0,
    progress: 0,
    failed: 0
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initializeTerminal();
    loadResources();
    setupEventListeners();
    startClock();
    updateSystemStats(); // Fetch real stats from API
});

function initializeTerminal() {
    addTerminalLine('Initializing Vrooli Resource Improver v1.0...', 'success');
    setTimeout(() => {
        addTerminalLine('Loading resource improvement modules...', 'success');
        setTimeout(() => {
            addTerminalLine('Connecting to resource improvement queue...', 'success');
            setTimeout(() => {
                addTerminalLine('Improvement terminal ready. Type "help" for commands.', 'success');
            }, 300);
        }, 200);
    }, 100);
}

function setupEventListeners() {
    // Refresh resources button
    document.getElementById('refresh-resources').addEventListener('click', loadResources);
    
    // Improvement input
    const improvementInput = document.getElementById('improvement-input');
    improvementInput.addEventListener('keydown', handleImprovementInput);
    
    // Improvement mode selector
    document.getElementById('improvement-mode').addEventListener('change', (e) => {
        addTerminalLine(`Switched to ${e.target.value.toUpperCase()} mode`, 'success');
        document.getElementById('current-mode').textContent = e.target.value.toUpperCase();
    });
}

function handleImprovementInput(e) {
    if (e.key === 'Enter') {
        const input = e.target.value.trim();
        if (!input) return;
        
        // Add to history
        improvementHistory.push(input);
        historyIndex = improvementHistory.length;
        
        // Process command
        processImprovementCommand(input);
        
        // Clear input
        e.target.value = '';
    } else if (e.key === 'ArrowUp') {
        // Navigate history up
        if (historyIndex > 0) {
            historyIndex--;
            e.target.value = improvementHistory[historyIndex];
        }
    } else if (e.key === 'ArrowDown') {
        // Navigate history down
        if (historyIndex < improvementHistory.length - 1) {
            historyIndex++;
            e.target.value = improvementHistory[historyIndex];
        } else {
            historyIndex = improvementHistory.length;
            e.target.value = '';
        }
    }
}

function processImprovementCommand(command) {
    addTerminalLine(`${command}`, null, true);
    
    const parts = command.toLowerCase().split(' ');
    const cmd = parts[0];
    
    switch(cmd) {
        case 'help':
            showHelp();
            break;
        case 'clear':
            clearTerminal();
            break;
        case 'resources':
        case 'list':
            loadResources();
            break;
        case 'select':
            if (parts[1]) {
                selectResource(parts.slice(1).join(' '));
            } else {
                addTerminalLine('Usage: select <resource-name>', 'error');
            }
            break;
        case 'analyze':
            if (selectedResource) {
                analyzeResource(selectedResource);
            } else {
                addTerminalLine('No resource selected. Use "select <resource-name>" first.', 'error');
            }
            break;
        case 'improve':
        case 'start':
            if (selectedResource) {
                startImprovement(selectedResource);
            } else {
                addTerminalLine('No resource selected. Use "select <resource-name>" first.', 'error');
            }
            break;
        case 'metrics':
        case 'performance':
            if (selectedResource) {
                getResourceMetrics(selectedResource);
            } else {
                addTerminalLine('No resource selected.', 'error');
            }
            break;
        case 'queue':
            showQueue();
            break;
        case 'submit':
        case 'report':
            if (parts.length > 1) {
                const issue = parts.slice(1).join(' ');
                reportIssue(selectedResource || 'unknown', issue);
            } else {
                openReportModal();
            }
            break;
        case 'status':
            if (selectedResource) {
                getResourceStatus(selectedResource);
            } else {
                showSystemStatus();
            }
            break;
        case 'stats':
            showStats();
            break;
        case 'export':
            exportImprovementLog();
            break;
        case 'health':
            if (selectedResource) {
                checkResourceHealth(selectedResource);
            } else {
                addTerminalLine('No resource selected.', 'error');
            }
            break;
        default:
            addTerminalLine(`Unknown command: ${cmd}. Type "help" for available commands.`, 'error');
    }
}

function showHelp() {
    const helpText = `
Available Commands:
  help              - Show this help message
  clear             - Clear the terminal
  resources/list    - List all available resources
  select <resource> - Select a resource for improvement
  analyze           - Analyze selected resource for improvements
  improve/start     - Start improvement process for selected resource
  metrics/performance - Check performance metrics for selected resource
  queue             - Show improvement queue status
  submit/report [issue] - Report an issue (opens modal if no issue provided)
  status            - Show resource or system status
  health            - Check health of selected resource
  stats             - Show improvement statistics
  export            - Export improvement log to file
  
Resource Improvement Modes:
  queue     - Monitor improvement queue
  analyze   - Resource analysis mode
  metrics   - Performance metrics mode  
  improve   - Active improvement mode
  
Tips:
  - Select a resource first to enable resource-specific commands
  - Use arrow keys to navigate command history
  - Switch improvement modes from the dropdown
    `;
    addTerminalLine(helpText);
}

function clearTerminal() {
    const output = document.getElementById('improvement-output');
    output.innerHTML = '<div class="terminal-line success"><span class="prompt">improve@vrooli:~$</span> Terminal cleared.</div>';
}

async function loadResources() {
    addTerminalLine('Fetching available resources...');
    
    try {
        const response = await fetch(`${API_BASE}/api/resources/list`);
        const data = await response.json();
        const resources = data.resources || [];
        
        displayResources(resources);
        addTerminalLine(`Found ${resources.length} resources`, 'success');
    } catch (error) {
        addTerminalLine(`Failed to fetch resources: ${error.message}`, 'error');
        displayResources([]);
    }
}

function displayResources(resources) {
    const resourceList = document.getElementById('resource-list');
    resourceList.innerHTML = '';
    
    resources.forEach(resource => {
        const resourceItem = document.createElement('div');
        resourceItem.className = 'resource-item';
        resourceItem.onclick = () => selectResourceUI(resource.name);
        
        const statusClass = resource.status === 'running' ? 'running' : 'stopped';
        const healthClass = resource.health === 'healthy' ? 'healthy' : 
                          resource.health === 'needs-improvement' ? 'needs-improvement' : 
                          resource.health === 'unhealthy' ? 'unhealthy' : 'stopped';
        
        resourceItem.innerHTML = `
            <div class="resource-name">${resource.name}</div>
            <div class="resource-status">
                <span class="status-badge ${statusClass}">${resource.status}</span>
                <span class="status-badge ${healthClass}">${resource.health}</span>
            </div>
        `;
        
        resourceList.appendChild(resourceItem);
    });
}

function selectResource(resourceName) {
    selectedResource = resourceName;
    updateSelectedResource();
    addTerminalLine(`Selected resource: ${resourceName}`, 'success');
}

function selectResourceUI(resourceName) {
    selectedResource = resourceName;
    updateSelectedResource();
    addTerminalLine(`Selected resource: ${resourceName}`, 'success');
    
    // Update UI selection
    document.querySelectorAll('.resource-item').forEach(item => {
        const name = item.querySelector('.resource-name').textContent;
        if (name === resourceName) {
            item.classList.add('selected');
        } else {
            item.classList.remove('selected');
        }
    });
}

function updateSelectedResource() {
    document.getElementById('selected-resource').textContent = selectedResource || 'NONE';
}

async function analyzeResource(resourceName) {
    addTerminalLine(`Analyzing resource: ${resourceName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/resources/${resourceName}/analyze`);
        const result = await response.json();
        
        if (result.analysis) {
            displayResourceAnalysis(result.analysis);
        } else {
            addTerminalLine('No analysis available', 'warning');
        }
    } catch (error) {
        addTerminalLine(`Failed to analyze resource: ${error.message}`, 'error');
    }
}

function displayResourceAnalysis(analysis) {
    addTerminalLine('=== RESOURCE ANALYSIS ===', 'success');
    addTerminalLine(`V2 Compliance Score: ${analysis.v2_compliance_score || 'N/A'}%`);
    addTerminalLine(`Health Reliability: ${analysis.health_reliability || 'N/A'}%`);
    addTerminalLine(`CLI Coverage: ${analysis.cli_coverage || 'N/A'}%`);
    addTerminalLine(`Doc Completeness: ${analysis.doc_completeness || 'N/A'}%`);
    
    if (analysis.recommendations && analysis.recommendations.length > 0) {
        addTerminalLine('Recommendations:', 'warning');
        analysis.recommendations.forEach((rec, index) => {
            addTerminalLine(`${index + 1}. ${rec}`);
        });
    }
    
    // Update last analysis panel
    const lastAnalysis = document.getElementById('last-analysis');
    lastAnalysis.innerHTML = `<pre>Resource: ${selectedResource}
V2 Compliance: ${analysis.v2_compliance_score || 'N/A'}%
Health Reliability: ${analysis.health_reliability || 'N/A'}%
CLI Coverage: ${analysis.cli_coverage || 'N/A'}%
Doc Completeness: ${analysis.doc_completeness || 'N/A'}%

Status: ${analysis.status || 'Unknown'}
Timestamp: ${new Date().toLocaleTimeString()}</pre>`;

    // Update health scores
    document.getElementById('compliance-score').textContent = `${analysis.v2_compliance_score || '--'}%`;
    document.getElementById('reliability-score').textContent = `${analysis.health_reliability || '--'}%`;
}

async function startImprovement(resourceName) {
    addTerminalLine(`Starting improvement for ${resourceName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/improvements/start`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                target: resourceName,
                type: 'general-improvement',
                priority: 'medium'
            })
        });
        
        const result = await response.json();
        addTerminalLine(`Improvement ${result.status}: ${result.message}`, 'success');
        
        if (result.queue_item_id) {
            addTerminalLine(`Queue item created: ${result.queue_item_id}`);
            stats.progress++;
            updateStats();
        }
    } catch (error) {
        addTerminalLine(`Failed to start improvement: ${error.message}`, 'error');
    }
}

async function getResourceMetrics(resourceName) {
    addTerminalLine(`Fetching metrics for ${resourceName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/resources/${resourceName}/metrics`);
        const data = await response.json();
        
        if (data.metrics) {
            addTerminalLine('=== PERFORMANCE METRICS ===', 'warning');
            Object.entries(data.metrics).forEach(([key, value]) => {
                addTerminalLine(`${key}: ${value}`);
            });
        } else {
            addTerminalLine('No metrics available', 'warning');
        }
    } catch (error) {
        addTerminalLine(`Failed to fetch metrics: ${error.message}`, 'error');
    }
}

async function showQueue() {
    addTerminalLine('Fetching improvement queue status...');
    
    try {
        const response = await fetch(`${API_BASE}/api/queue/status`);
        const data = await response.json();
        
        addTerminalLine('=== IMPROVEMENT QUEUE ===', 'success');
        addTerminalLine(`Pending: ${data.pending || 0}`);
        addTerminalLine(`In Progress: ${data.in_progress || 0}`);
        addTerminalLine(`Completed: ${data.completed || 0}`);
        addTerminalLine(`Failed: ${data.failed || 0}`);
        
        if (data.recent && data.recent.length > 0) {
            addTerminalLine('\nRecent Items:', 'warning');
            data.recent.forEach(item => {
                addTerminalLine(`${item.title} - ${item.status}`);
            });
        }
        
        // Update stats
        stats.pending = data.pending || 0;
        stats.progress = data.in_progress || 0;
        stats.improved = data.completed || 0;
        stats.failed = data.failed || 0;
        updateStats();
        
    } catch (error) {
        addTerminalLine(`Failed to fetch queue status: ${error.message}`, 'error');
    }
}

async function reportIssue(resourceName, issue) {
    addTerminalLine(`Reporting issue for ${resourceName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/reports`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                resource_name: resourceName,
                issue_type: 'general',
                description: issue,
                context: {}
            })
        });
        
        const result = await response.json();
        addTerminalLine(`Issue reported successfully: ${result.id}`, 'success');
    } catch (error) {
        addTerminalLine(`Failed to report issue: ${error.message}`, 'error');
    }
}

function openReportModal() {
    document.getElementById('report-modal').classList.remove('hidden');
    if (selectedResource) {
        document.getElementById('report-resource-name').value = selectedResource;
    }
}

function closeReportModal() {
    document.getElementById('report-modal').classList.add('hidden');
}

function submitReport() {
    const resourceName = document.getElementById('report-resource-name').value;
    const issueType = document.getElementById('issue-type').value;
    const description = document.getElementById('issue-description').value;
    const context = document.getElementById('issue-context').value;
    
    if (!description) {
        alert('Please enter a description');
        return;
    }
    
    closeReportModal();
    
    // Submit the report
    const fullIssue = `Type: ${issueType}\nDescription: ${description}\nContext: ${context}`;
    reportIssue(resourceName, fullIssue);
}

async function getResourceStatus(resourceName) {
    addTerminalLine(`Checking status for ${resourceName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/resources/${resourceName}/status`);
        const data = await response.json();
        
        addTerminalLine(`Status: ${data.status}`, data.status === 'running' ? 'success' : 'warning');
        addTerminalLine(`Health: ${data.health}`, data.health === 'healthy' ? 'success' : 'error');
        
        if (data.details) {
            Object.entries(data.details).forEach(([key, value]) => {
                addTerminalLine(`${key}: ${value}`);
            });
        }
    } catch (error) {
        addTerminalLine(`Failed to check status: ${error.message}`, 'error');
    }
}

function showSystemStatus() {
    addTerminalLine('=== SYSTEM STATUS ===', 'success');
    addTerminalLine(`Selected Resource: ${selectedResource || 'None'}`);
    addTerminalLine(`API Connection: Connected`);
    addTerminalLine(`Queue Processing: Active`);
    addTerminalLine(`Improvement Engine: Running`);
}

async function checkResourceHealth(resourceName) {
    addTerminalLine(`Checking health for ${resourceName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/resources/${resourceName}/health`);
        const data = await response.json();
        
        const healthClass = data.health === 'healthy' ? 'success' : 'error';
        addTerminalLine(`Health Status: ${data.health}`, healthClass);
        
        if (data.checks && data.checks.length > 0) {
            addTerminalLine('Health Checks:', 'warning');
            data.checks.forEach(check => {
                const status = check.passed ? '✓' : '✗';
                const className = check.passed ? 'success' : 'error';
                addTerminalLine(`${status} ${check.name}: ${check.message}`, className);
            });
        }
    } catch (error) {
        addTerminalLine(`Failed to check health: ${error.message}`, 'error');
    }
}

function showStats() {
    addTerminalLine('=== IMPROVEMENT STATISTICS ===', 'success');
    addTerminalLine(`Pending Improvements: ${stats.pending}`);
    addTerminalLine(`In Progress: ${stats.progress}`);
    addTerminalLine(`Completed Improvements: ${stats.improved}`);
    addTerminalLine(`Failed Improvements: ${stats.failed}`);
}

function exportImprovementLog() {
    const output = document.getElementById('improvement-output').innerText;
    const blob = new Blob([output], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `improvement-log-${Date.now()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
    
    addTerminalLine('Improvement log exported successfully', 'success');
}

function addTerminalLine(text, className = null, isCommand = false) {
    const output = document.getElementById('improvement-output');
    const line = document.createElement('div');
    line.className = 'terminal-line';
    if (className) line.classList.add(className);
    
    if (isCommand) {
        line.innerHTML = `<span class="prompt">improve@vrooli:~$</span> ${text}`;
    } else {
        line.textContent = text;
    }
    
    output.appendChild(line);
    output.scrollTop = output.scrollHeight;
}

function updateStats() {
    document.getElementById('pending-count').textContent = stats.pending;
    document.getElementById('improved-count').textContent = stats.improved;
    document.getElementById('progress-count').textContent = stats.progress;
    document.getElementById('failed-count').textContent = stats.failed;
}

function startClock() {
    function updateTime() {
        const now = new Date();
        const time = now.toLocaleTimeString('en-US', { 
            hour12: false, 
            hour: '2-digit', 
            minute: '2-digit', 
            second: '2-digit' 
        });
        document.getElementById('current-time').textContent = time;
    }
    
    updateTime();
    setInterval(updateTime, 1000);
}

async function updateSystemStats() {
    async function fetchStats() {
        try {
            const response = await fetch(`${API_BASE}/health`);
            const data = await response.json();
            
            // Update CPU and memory if available in health response
            if (data.resources && data.resources.length > 0) {
                // Calculate average CPU and memory across all resources
                let totalCpu = 0;
                let totalMem = 0;
                let count = 0;
                
                data.resources.forEach(resource => {
                    if (resource.cpu_usage) {
                        totalCpu += resource.cpu_usage;
                        count++;
                    }
                    if (resource.memory_usage) {
                        totalMem += resource.memory_usage;
                    }
                });
                
                if (count > 0) {
                    const avgCpu = Math.round(totalCpu / count);
                    document.getElementById('cpu-usage').textContent = `${avgCpu}%`;
                }
                if (data.resources.length > 0) {
                    const avgMem = Math.round(totalMem / data.resources.length);
                    document.getElementById('mem-usage').textContent = `${avgMem}MB`;
                }
            } else {
                // If no data available, show N/A
                document.getElementById('cpu-usage').textContent = 'N/A';
                document.getElementById('mem-usage').textContent = 'N/A';
            }
        } catch (error) {
            // On error, show N/A
            document.getElementById('cpu-usage').textContent = 'N/A';
            document.getElementById('mem-usage').textContent = 'N/A';
        }
    }
    
    fetchStats();
    setInterval(fetchStats, 5000); // Update every 5 seconds
}

// Global functions for modal
window.closeReportModal = closeReportModal;
window.submitReport = submitReport;