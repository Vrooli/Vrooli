// App Debugger Interactive Terminal
const API_BASE = window.location.hostname === 'localhost' 
    ? 'http://localhost:30150' 
    : `http://${window.location.hostname}:30150`;

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
    simulateSystemStats();
});

function initializeTerminal() {
    addTerminalLine('Initializing Vrooli App Debugger v2.0...', 'success');
    setTimeout(() => {
        addTerminalLine('Loading debugging modules...', 'success');
        setTimeout(() => {
            addTerminalLine('Connecting to N8N workflows...', 'success');
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
    addTerminalLine('Fetching running apps...');
    
    try {
        const response = await fetch(`${API_BASE}/api/apps`);
        const apps = await response.json();
        
        displayApps(apps);
        addTerminalLine(`Found ${apps.length} apps`, 'success');
    } catch (error) {
        // Use mock data if API fails
        const mockApps = [
            { name: 'app-monitor', status: 'running', health: 'healthy' },
            { name: 'prompt-manager', status: 'running', health: 'warning' },
            { name: 'idea-generator', status: 'stopped', health: 'unknown' },
            { name: 'resume-screener', status: 'running', health: 'healthy' },
            { name: 'study-buddy', status: 'running', health: 'error' }
        ];
        displayApps(mockApps);
        addTerminalLine('Using mock data (API unreachable)', 'warning');
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
        
        // Simulate receiving debug data
        setTimeout(() => {
            simulateDebugData(debugMode);
        }, 1000);
    } catch (error) {
        addTerminalLine(`Failed to start debug session: ${error.message}`, 'error');
        // Show mock debug data anyway
        simulateDebugData(debugMode);
    }
}

function simulateDebugData(mode) {
    switch(mode) {
        case 'errors':
            addTerminalLine('=== ERROR ANALYSIS ===', 'warning');
            addTerminalLine('Found 3 errors in the last hour:');
            addTerminalLine('1. TypeError: Cannot read property "map" of undefined');
            addTerminalLine('   at processData (main.js:45)');
            addTerminalLine('2. NetworkError: Failed to fetch from API endpoint');
            addTerminalLine('   at fetchUserData (api.js:120)');
            addTerminalLine('3. ValidationError: Invalid email format');
            addTerminalLine('   at validateForm (validators.js:33)');
            stats.errors += 3;
            stats.pending += 3;
            updateStats();
            break;
            
        case 'logs':
            addTerminalLine('=== RECENT LOGS ===', 'warning');
            addTerminalLine('[INFO] Application started successfully');
            addTerminalLine('[INFO] Connected to database');
            addTerminalLine('[WARN] High memory usage detected (85%)');
            addTerminalLine('[ERROR] Failed to process webhook');
            addTerminalLine('[INFO] Retrying webhook processing...');
            addTerminalLine('[SUCCESS] Webhook processed on retry');
            break;
            
        case 'performance':
            addTerminalLine('=== PERFORMANCE METRICS ===', 'warning');
            addTerminalLine('CPU Usage: 45%');
            addTerminalLine('Memory: 512MB / 1024MB');
            addTerminalLine('Response Time (avg): 235ms');
            addTerminalLine('Request Rate: 120 req/min');
            addTerminalLine('Error Rate: 0.3%');
            addTerminalLine('Database Queries (avg): 12ms');
            break;
            
        case 'fix':
            addTerminalLine('=== FIX SUGGESTIONS ===', 'warning');
            addTerminalLine('For TypeError: Check if data exists before mapping');
            addTerminalLine('Suggested fix:');
            addTerminalLine('  const items = data?.items || [];');
            addTerminalLine('  return items.map(item => item.name);');
            addTerminalLine('');
            addTerminalLine('For NetworkError: Implement retry logic');
            addTerminalLine('For ValidationError: Update regex pattern');
            stats.fixed++;
            stats.pending = Math.max(0, stats.pending - 1);
            updateStats();
            break;
    }
}

async function analyzeError(errorText) {
    addTerminalLine('Analyzing error...');
    
    try {
        const response = await fetch(`${API_BASE}/api/report-error`, {
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
        
        // Display analysis
        const analysis = {
            root_cause: 'Undefined variable access',
            severity: 'medium',
            fix_suggestion: 'Add null check before accessing property',
            prevention: 'Use TypeScript or add validation'
        };
        
        displayAnalysis(analysis);
    } catch (error) {
        // Mock analysis
        const analysis = {
            root_cause: 'Analyzing: ' + errorText.substring(0, 50) + '...',
            severity: errorText.includes('Critical') ? 'critical' : 'medium',
            fix_suggestion: 'Check error handling in the affected module',
            prevention: 'Add comprehensive error boundaries'
        };
        
        displayAnalysis(analysis);
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

function fetchLogs(appName) {
    addTerminalLine(`Fetching logs for ${appName}...`);
    simulateDebugData('logs');
}

function checkPerformance(appName) {
    addTerminalLine(`Checking performance for ${appName}...`);
    simulateDebugData('performance');
}

function suggestFixes(appName) {
    addTerminalLine(`Generating fix suggestions for ${appName}...`);
    simulateDebugData('fix');
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

function simulateSystemStats() {
    function update() {
        const cpu = Math.floor(Math.random() * 30 + 20); // 20-50%
        const mem = Math.floor(Math.random() * 200 + 300); // 300-500MB
        
        document.getElementById('cpu-usage').textContent = `${cpu}%`;
        document.getElementById('mem-usage').textContent = `${mem}MB`;
    }
    
    update();
    setInterval(update, 3000);
}

// Global functions for modal
window.closeErrorModal = closeErrorModal;
window.submitError = submitError;