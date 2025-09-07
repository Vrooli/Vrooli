// App Debugger Interactive Terminal
// API base URL - configurable via global variables or environment
const API_HOST = window.SCENARIO_IMPROVER_API_HOST || window.location.hostname;
const API_PORT = window.SCENARIO_IMPROVER_API_PORT || window.API_PORT;
const API_BASE = window.SCENARIO_IMPROVER_API_URL || 
    (API_PORT ? `${window.location.protocol}//${API_HOST}:${API_PORT}` : 
                `${window.location.protocol}//${API_HOST}:${window.location.port}`);

let selectedApp = null;
let debugHistory = [];
let historyIndex = -1;
let stats = {
    errors: 0,
    fixed: 0,
    pending: 0,
    critical: 0
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initializeTerminal();
    loadApps();
    setupEventListeners();
    startClock();
    updateSystemStats(); // Fetch real stats from API
});

function initializeTerminal() {
    addTerminalLine('Initializing Vrooli App Debugger v2.0...', 'success');
    setTimeout(() => {
        addTerminalLine('Loading debugging modules...', 'success');
        setTimeout(() => {
            addTerminalLine('Connecting to scenario improvement queue...', 'success');
            setTimeout(() => {
                addTerminalLine('Debug terminal ready. Type "help" for commands.', 'success');
            }, 300);
        }, 200);
    }, 100);
}

function setupEventListeners() {
    // Refresh apps button
    document.getElementById('refresh-apps').addEventListener('click', loadApps);
    
    // Debug input
    const debugInput = document.getElementById('debug-input');
    debugInput.addEventListener('keydown', handleDebugInput);
    
    // Debug mode selector
    document.getElementById('debug-mode').addEventListener('change', (e) => {
        addTerminalLine(`Switched to ${e.target.value.toUpperCase()} mode`, 'success');
        document.getElementById('current-mode').textContent = e.target.value.toUpperCase();
    });
}

function handleDebugInput(e) {
    if (e.key === 'Enter') {
        const input = e.target.value.trim();
        if (!input) return;
        
        // Add to history
        debugHistory.push(input);
        historyIndex = debugHistory.length;
        
        // Process command
        processDebugCommand(input);
        
        // Clear input
        e.target.value = '';
    } else if (e.key === 'ArrowUp') {
        // Navigate history up
        if (historyIndex > 0) {
            historyIndex--;
            e.target.value = debugHistory[historyIndex];
        }
    } else if (e.key === 'ArrowDown') {
        // Navigate history down
        if (historyIndex < debugHistory.length - 1) {
            historyIndex++;
            e.target.value = debugHistory[historyIndex];
        } else {
            historyIndex = debugHistory.length;
            e.target.value = '';
        }
    }
}

function processDebugCommand(command) {
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
        case 'apps':
            loadApps();
            break;
        case 'select':
            if (parts[1]) {
                selectApp(parts.slice(1).join(' '));
            } else {
                addTerminalLine('Usage: select <app-name>', 'error');
            }
            break;
        case 'debug':
            if (selectedApp) {
                startDebugSession();
            } else {
                addTerminalLine('No app selected. Use "select <app-name>" first.', 'error');
            }
            break;
        case 'analyze':
            if (parts.length > 1) {
                analyzeError(parts.slice(1).join(' '));
            } else {
                openErrorModal();
            }
            break;
        case 'logs':
            if (selectedApp) {
                fetchLogs(selectedApp);
            } else {
                addTerminalLine('No app selected.', 'error');
            }
            break;
        case 'perf':
        case 'performance':
            if (selectedApp) {
                checkPerformance(selectedApp);
            } else {
                addTerminalLine('No app selected.', 'error');
            }
            break;
        case 'fix':
            if (selectedApp) {
                suggestFixes(selectedApp);
            } else {
                addTerminalLine('No app selected.', 'error');
            }
            break;
        case 'stats':
            showStats();
            break;
        case 'export':
            exportDebugLog();
            break;
        default:
            // Try to interpret as error message
            if (command.includes('Error') || command.includes('error')) {
                analyzeError(command);
            } else {
                addTerminalLine(`Unknown command: ${cmd}. Type "help" for available commands.`, 'error');
            }
    }
}

function showHelp() {
    const helpText = `
Available Commands:
  help              - Show this help message
  clear             - Clear the terminal
  apps              - List all running apps
  select <app>      - Select an app for debugging
  debug             - Start debug session for selected app
  analyze [error]   - Analyze an error (opens modal if no error provided)
  logs              - Show logs for selected app
  perf/performance  - Check performance metrics
  fix               - Suggest fixes for recent errors
  stats             - Show debugging statistics
  export            - Export debug log to file
  
Tips:
  - Paste error messages directly to get instant analysis
  - Use arrow keys to navigate command history
  - Select different debug modes from the dropdown
    `;
    addTerminalLine(helpText);
}

function clearTerminal() {
    const output = document.getElementById('debug-output');
    output.innerHTML = '<div class="terminal-line success"><span class="prompt">debug@vrooli:~$</span> Terminal cleared.</div>';
}

async function loadApps() {
    addTerminalLine('Fetching running scenarios...');
    
    try {
        const response = await fetch(`${API_BASE}/api/scenarios/list`);
        const data = await response.json();
        const scenarios = data.scenarios || [];
        
        displayApps(scenarios);
        addTerminalLine(`Found ${scenarios.length} scenarios`, 'success');
    } catch (error) {
        addTerminalLine(`Failed to fetch scenarios: ${error.message}`, 'error');
        displayApps([]);
    }
}

function displayApps(apps) {
    const appList = document.getElementById('app-list');
    appList.innerHTML = '';
    
    apps.forEach(app => {
        const appItem = document.createElement('div');
        appItem.className = 'app-item';
        appItem.onclick = () => selectAppUI(app.name);
        
        const statusClass = app.status === 'running' ? 'running' : 'stopped';
        const healthClass = app.health === 'healthy' ? 'success' : 
                          app.health === 'warning' ? 'warning' : 
                          app.health === 'error' ? 'error' : 'stopped';
        
        appItem.innerHTML = `
            <div class="app-name">${app.name}</div>
            <div class="app-status">
                <span class="status-badge ${statusClass}">${app.status}</span>
                <span class="status-badge ${healthClass}">${app.health}</span>
            </div>
        `;
        
        appList.appendChild(appItem);
    });
}

function selectApp(appName) {
    selectedApp = appName;
    updateSelectedApp();
    addTerminalLine(`Selected app: ${appName}`, 'success');
}

function selectAppUI(appName) {
    selectedApp = appName;
    updateSelectedApp();
    addTerminalLine(`Selected app: ${appName}`, 'success');
    
    // Update UI selection
    document.querySelectorAll('.app-item').forEach(item => {
        const name = item.querySelector('.app-name').textContent;
        if (name === appName) {
            item.classList.add('selected');
        } else {
            item.classList.remove('selected');
        }
    });
}

function updateSelectedApp() {
    document.getElementById('selected-app').textContent = selectedApp || 'NONE';
}

async function startDebugSession() {
    addTerminalLine(`Starting debug session for ${selectedApp}...`);
    
    const debugMode = document.getElementById('debug-mode').value;
    
    try {
        const response = await fetch(`${API_BASE}/api/debug`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                app_name: selectedApp,
                debug_type: debugMode
            })
        });
        
        const result = await response.json();
        addTerminalLine(`Debug session ${result.status}: ${result.message}`, 'success');
        
        // Display actual debug results if available
        if (result.data) {
            displayDebugResults(result.data, debugMode);
        }
    } catch (error) {
        addTerminalLine(`Failed to start debug session: ${error.message}`, 'error');
    }
}

function displayDebugResults(data, mode) {
    switch(mode) {
        case 'errors':
            addTerminalLine('=== ERROR ANALYSIS ===', 'warning');
            if (data.errors && data.errors.length > 0) {
                addTerminalLine(`Found ${data.errors.length} errors:`);
                data.errors.forEach((error, index) => {
                    addTerminalLine(`${index + 1}. ${error.message}`);
                    if (error.stack) {
                        addTerminalLine(`   at ${error.stack}`);
                    }
                });
                stats.errors += data.errors.length;
                stats.pending += data.errors.length;
            } else {
                addTerminalLine('No errors found');
            }
            updateStats();
            break;
            
        case 'logs':
            addTerminalLine('=== RECENT LOGS ===', 'warning');
            if (data.logs && data.logs.length > 0) {
                data.logs.forEach(log => {
                    addTerminalLine(log);
                });
            } else {
                addTerminalLine('No recent logs available');
            }
            break;
            
        case 'performance':
            addTerminalLine('=== PERFORMANCE METRICS ===', 'warning');
            if (data.metrics) {
                Object.entries(data.metrics).forEach(([key, value]) => {
                    addTerminalLine(`${key}: ${value}`);
                });
            } else {
                addTerminalLine('No performance data available');
            }
            break;
            
        case 'fix':
            addTerminalLine('=== FIX SUGGESTIONS ===', 'warning');
            if (data.suggestions && data.suggestions.length > 0) {
                data.suggestions.forEach(suggestion => {
                    addTerminalLine(suggestion);
                });
                stats.fixed++;
                stats.pending = Math.max(0, stats.pending - 1);
            } else {
                addTerminalLine('No fix suggestions available');
            }
            updateStats();
            break;
    }
}

async function analyzeError(errorText) {
    addTerminalLine('Analyzing error...');
    
    try {
        const response = await fetch(`${API_BASE}/api/error/analyze`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                app_name: selectedApp || 'unknown',
                error_message: errorText,
                stack_trace: '',
                context: {}
            })
        });
        
        const result = await response.json();
        
        // Display actual analysis from API
        if (result.analysis) {
            const analysis = {
                root_cause: result.analysis.root_cause || 'Unknown',
                severity: result.analysis.priority || result.analysis.severity || 'medium',
                fix_suggestion: (result.analysis.suggestions && result.analysis.suggestions[0]) || 'No suggestions available',
                prevention: result.analysis.prevention || 'Follow best practices'
            };
            displayAnalysis(analysis);
        } else {
            addTerminalLine('No analysis available', 'warning');
        }
    } catch (error) {
        addTerminalLine(`Failed to analyze error: ${error.message}`, 'error');
    }
}

function displayAnalysis(analysis) {
    addTerminalLine('=== ANALYSIS COMPLETE ===', 'success');
    addTerminalLine(`Root Cause: ${analysis.root_cause}`);
    addTerminalLine(`Severity: ${analysis.severity.toUpperCase()}`, 
                    analysis.severity === 'critical' ? 'error' : 'warning');
    addTerminalLine(`Fix: ${analysis.fix_suggestion}`);
    addTerminalLine(`Prevention: ${analysis.prevention}`);
    
    // Update last analysis panel
    const lastAnalysis = document.getElementById('last-analysis');
    lastAnalysis.innerHTML = `<pre>Root Cause: ${analysis.root_cause}
Severity: ${analysis.severity.toUpperCase()}
Fix Suggestion: ${analysis.fix_suggestion}
Prevention: ${analysis.prevention}

Timestamp: ${new Date().toLocaleTimeString()}</pre>`;
    
    if (analysis.severity === 'critical') {
        stats.critical++;
    }
    updateStats();
}

function openErrorModal() {
    document.getElementById('error-modal').classList.remove('hidden');
    if (selectedApp) {
        document.getElementById('error-app-name').value = selectedApp;
    }
}

function closeErrorModal() {
    document.getElementById('error-modal').classList.add('hidden');
}

function submitError() {
    const appName = document.getElementById('error-app-name').value;
    const errorMessage = document.getElementById('error-message').value;
    const stackTrace = document.getElementById('stack-trace').value;
    const context = document.getElementById('error-context').value;
    
    if (!errorMessage) {
        alert('Please enter an error message');
        return;
    }
    
    closeErrorModal();
    
    // Analyze the error
    const fullError = `${errorMessage}\n${stackTrace}`;
    analyzeError(fullError);
}

async function fetchLogs(appName) {
    addTerminalLine(`Fetching logs for ${appName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/logs/${appName}`);
        const data = await response.json();
        displayDebugResults({ logs: data.logs || [] }, 'logs');
    } catch (error) {
        addTerminalLine(`Failed to fetch logs: ${error.message}`, 'error');
    }
}

async function checkPerformance(appName) {
    addTerminalLine(`Checking performance for ${appName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/performance/profile?app=${appName}`);
        const data = await response.json();
        if (data.metrics) {
            displayDebugResults({ metrics: data.metrics }, 'performance');
        } else {
            addTerminalLine('No performance data available', 'warning');
        }
    } catch (error) {
        addTerminalLine(`Failed to check performance: ${error.message}`, 'error');
    }
}

async function suggestFixes(appName) {
    addTerminalLine(`Generating fix suggestions for ${appName}...`);
    
    try {
        const response = await fetch(`${API_BASE}/api/fixes/${appName}`);
        const data = await response.json();
        displayDebugResults({ suggestions: data.suggestions || [] }, 'fix');
    } catch (error) {
        addTerminalLine(`Failed to get fix suggestions: ${error.message}`, 'error');
    }
}

function showStats() {
    addTerminalLine('=== DEBUG STATISTICS ===', 'success');
    addTerminalLine(`Total Errors Analyzed: ${stats.errors}`);
    addTerminalLine(`Errors Fixed: ${stats.fixed}`);
    addTerminalLine(`Pending Issues: ${stats.pending}`);
    addTerminalLine(`Critical Issues: ${stats.critical}`);
}

function exportDebugLog() {
    const output = document.getElementById('debug-output').innerText;
    const blob = new Blob([output], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `debug-log-${Date.now()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
    
    addTerminalLine('Debug log exported successfully', 'success');
}

function addTerminalLine(text, className = null, isCommand = false) {
    const output = document.getElementById('debug-output');
    const line = document.createElement('div');
    line.className = 'terminal-line';
    if (className) line.classList.add(className);
    
    if (isCommand) {
        line.innerHTML = `<span class="prompt">debug@vrooli:~$</span> ${text}`;
    } else {
        line.textContent = text;
    }
    
    output.appendChild(line);
    output.scrollTop = output.scrollHeight;
}

function updateStats() {
    document.getElementById('error-count').textContent = stats.errors;
    document.getElementById('fixed-count').textContent = stats.fixed;
    document.getElementById('pending-count').textContent = stats.pending;
    document.getElementById('critical-count').textContent = stats.critical;
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
            if (data.scenarios && data.scenarios.length > 0) {
                // Calculate average CPU and memory across all scenarios
                let totalCpu = 0;
                let totalMem = 0;
                let count = 0;
                
                data.scenarios.forEach(scenario => {
                    if (scenario.cpu_usage) {
                        totalCpu += scenario.cpu_usage;
                        count++;
                    }
                    if (scenario.memory_usage) {
                        totalMem += scenario.memory_usage;
                    }
                });
                
                if (count > 0) {
                    const avgCpu = Math.round(totalCpu / count);
                    document.getElementById('cpu-usage').textContent = `${avgCpu}%`;
                }
                if (data.scenarios.length > 0) {
                    const avgMem = Math.round(totalMem / data.scenarios.length);
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
window.closeErrorModal = closeErrorModal;
window.submitError = submitError;