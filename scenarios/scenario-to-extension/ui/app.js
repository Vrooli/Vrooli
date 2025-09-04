/**
 * Browser Extension Generator UI
 * Main application logic for the scenario-to-extension web interface
 */

// Configuration
const CONFIG = {
    API_BASE: '/api/v1',
    POLL_INTERVAL: 5000, // 5 seconds
    NOTIFICATION_TIMEOUT: 5000,
    DEBUG: false
};

// Application state
const state = {
    currentTab: 'generator',
    selectedTemplate: 'full',
    builds: [],
    templates: [],
    systemStatus: null,
    isLoading: false
};

// ============================================
// Utility Functions
// ============================================

function log(...args) {
    if (CONFIG.DEBUG) {
        console.log('[ExtensionGenerator]', ...args);
    }
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function formatDate(dateString) {
    return new Date(dateString).toLocaleString();
}

function formatDuration(startTime, endTime) {
    const start = new Date(startTime);
    const end = endTime ? new Date(endTime) : new Date();
    const diff = end - start;
    
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (hours > 0) return `${hours}h ${minutes % 60}m`;
    if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
    return `${seconds}s`;
}

// ============================================
// API Client
// ============================================

class APIClient {
    static async request(method, endpoint, data = null) {
        const url = `${CONFIG.API_BASE}${endpoint}`;
        
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
        };
        
        if (data) {
            options.body = JSON.stringify(data);
        }
        
        log(`API ${method} ${url}`, data);
        
        try {
            const response = await fetch(url, options);
            
            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.message || `HTTP ${response.status}: ${response.statusText}`);
            }
            
            const result = await response.json();
            log(`API Response:`, result);
            return result;
        } catch (error) {
            log('API Error:', error);
            throw error;
        }
    }
    
    static async get(endpoint) {
        return this.request('GET', endpoint);
    }
    
    static async post(endpoint, data) {
        return this.request('POST', endpoint, data);
    }
    
    static async put(endpoint, data) {
        return this.request('PUT', endpoint, data);
    }
    
    static async delete(endpoint) {
        return this.request('DELETE', endpoint);
    }
}

// ============================================
// UI Components
// ============================================

class NotificationManager {
    static show(type, title, message, timeout = CONFIG.NOTIFICATION_TIMEOUT) {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.innerHTML = `
            <div class="notification-header">
                <span class="notification-title">${title}</span>
                <button class="notification-close">&times;</button>
            </div>
            <div class="notification-message">${message}</div>
        `;
        
        const container = document.getElementById('notifications');
        container.appendChild(notification);
        
        // Close button handler
        notification.querySelector('.notification-close').addEventListener('click', () => {
            notification.remove();
        });
        
        // Auto-remove after timeout
        if (timeout > 0) {
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.remove();
                }
            }, timeout);
        }
        
        return notification;
    }
    
    static success(title, message) {
        return this.show('success', title, message);
    }
    
    static warning(title, message) {
        return this.show('warning', title, message);
    }
    
    static error(title, message) {
        return this.show('error', title, message, 8000);
    }
    
    static info(title, message) {
        return this.show('info', title, message);
    }
}

class LoadingManager {
    static show(title = 'Processing...', message = 'This may take a moment...') {
        const overlay = document.getElementById('loading-overlay');
        document.getElementById('loading-title').textContent = title;
        document.getElementById('loading-message').textContent = message;
        overlay.classList.remove('hidden');
    }
    
    static hide() {
        document.getElementById('loading-overlay').classList.add('hidden');
    }
    
    static setButtonLoading(buttonId, loading = true) {
        const button = document.getElementById(buttonId);
        const spinner = button.querySelector('.spinner');
        
        if (loading) {
            button.disabled = true;
            if (spinner) spinner.classList.remove('hidden');
        } else {
            button.disabled = false;
            if (spinner) spinner.classList.add('hidden');
        }
    }
}

// ============================================
// Tab Management
// ============================================

function initializeTabs() {
    const tabButtons = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');
    
    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const targetTab = button.getAttribute('data-tab');
            switchTab(targetTab);
        });
    });
}

function switchTab(tabName) {
    state.currentTab = tabName;
    
    // Update buttons
    document.querySelectorAll('.tab-button').forEach(btn => {
        btn.classList.toggle('active', btn.getAttribute('data-tab') === tabName);
    });
    
    // Update content
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === `tab-${tabName}`);
    });
    
    // Load tab-specific data
    switch (tabName) {
        case 'builds':
            loadBuilds();
            break;
        case 'templates':
            loadTemplates();
            break;
        case 'testing':
            // Testing tab is ready by default
            break;
    }
}

// ============================================
// System Status
// ============================================

async function refreshStatus() {
    try {
        const status = await APIClient.get('/health');
        state.systemStatus = status;
        updateStatusIndicator(status);
    } catch (error) {
        log('Status check failed:', error);
        updateStatusIndicator({ status: 'error' });
    }
}

function updateStatusIndicator(status) {
    const indicator = document.getElementById('status-indicator');
    const dot = document.getElementById('status-dot');
    const text = document.getElementById('status-text');
    
    // Remove all status classes
    indicator.classList.remove('healthy', 'warning', 'error');
    
    if (status.status === 'healthy') {
        indicator.classList.add('healthy');
        text.textContent = 'System Healthy';
    } else if (status.status === 'warning') {
        indicator.classList.add('warning');
        text.textContent = 'System Warning';
    } else {
        indicator.classList.add('error');
        text.textContent = 'System Error';
    }
}

// ============================================
// Extension Generation
// ============================================

function initializeGenerator() {
    // Template selection
    const templateCards = document.querySelectorAll('.template-card');
    templateCards.forEach(card => {
        card.addEventListener('click', () => {
            templateCards.forEach(c => c.classList.remove('active'));
            card.classList.add('active');
            state.selectedTemplate = card.getAttribute('data-template');
        });
    });
    
    // Form submission
    const form = document.getElementById('extension-form');
    form.addEventListener('submit', handleExtensionGeneration);
    
    // Auto-populate app name from scenario name
    const scenarioNameInput = document.getElementById('scenario-name');
    const appNameInput = document.getElementById('app-name');
    
    scenarioNameInput.addEventListener('input', debounce(() => {
        if (!appNameInput.value || appNameInput.dataset.autoGenerated) {
            const scenarioName = scenarioNameInput.value;
            const appName = scenarioName
                .split('-')
                .map(word => word.charAt(0).toUpperCase() + word.slice(1))
                .join(' ') + ' Extension';
            appNameInput.value = appName;
            appNameInput.dataset.autoGenerated = 'true';
        }
    }, 300));
    
    appNameInput.addEventListener('input', () => {
        delete appNameInput.dataset.autoGenerated;
    });
}

async function handleExtensionGeneration(event) {
    event.preventDefault();
    
    LoadingManager.setButtonLoading('generate-btn', true);
    
    try {
        // Collect form data
        const formData = new FormData(event.target);
        const permissions = Array.from(document.getElementById('permissions').selectedOptions)
            .map(option => option.value);
        const hostPermissions = document.getElementById('host-permissions').value
            .split('\n')
            .map(line => line.trim())
            .filter(line => line.length > 0);
        
        const requestData = {
            scenario_name: formData.get('scenario_name'),
            template_type: state.selectedTemplate,
            config: {
                app_name: formData.get('app_name'),
                app_description: formData.get('app_description'),
                api_endpoint: formData.get('api_endpoint'),
                permissions: permissions,
                host_permissions: hostPermissions,
                version: formData.get('version') || '1.0.0',
                author_name: formData.get('author') || 'Vrooli Scenario Generator',
                license: 'MIT',
                debug_mode: formData.get('debug_mode') === 'on'
            }
        };
        
        log('Generating extension with config:', requestData);
        
        // Make API request
        const result = await APIClient.post('/extension/generate', requestData);
        
        NotificationManager.success(
            'Extension Generation Started',
            `Build ${result.build_id} is now processing. Check the Builds tab for progress.`
        );
        
        // Switch to builds tab to show progress
        setTimeout(() => {
            switchTab('builds');
            startBuildPolling(result.build_id);
        }, 1000);
        
    } catch (error) {
        log('Extension generation failed:', error);
        NotificationManager.error(
            'Generation Failed',
            error.message || 'Unknown error occurred during extension generation'
        );
    } finally {
        LoadingManager.setButtonLoading('generate-btn', false);
    }
}

function resetForm() {
    document.getElementById('extension-form').reset();
    document.getElementById('api-endpoint').value = 'http://localhost:3000';
    document.getElementById('version').value = '1.0.0';
    document.getElementById('author').value = 'Vrooli Scenario Generator';
    
    // Reset template selection
    document.querySelectorAll('.template-card').forEach(card => {
        card.classList.remove('active');
    });
    document.querySelector('[data-template="full"]').classList.add('active');
    state.selectedTemplate = 'full';
    
    // Reset permissions
    const permissionsSelect = document.getElementById('permissions');
    Array.from(permissionsSelect.options).forEach(option => {
        option.selected = ['storage', 'activeTab'].includes(option.value);
    });
}

// ============================================
// Builds Management
// ============================================

async function loadBuilds() {
    const container = document.getElementById('builds-container');
    const loading = document.getElementById('builds-loading');
    const empty = document.getElementById('builds-empty');
    const list = document.getElementById('builds-list');
    
    try {
        loading.classList.remove('hidden');
        empty.classList.add('hidden');
        list.innerHTML = '';
        
        const response = await APIClient.get('/extension/builds');
        state.builds = response.builds || [];
        
        if (state.builds.length === 0) {
            loading.classList.add('hidden');
            empty.classList.remove('hidden');
        } else {
            loading.classList.add('hidden');
            renderBuilds();
        }
        
    } catch (error) {
        log('Failed to load builds:', error);
        loading.classList.add('hidden');
        NotificationManager.error('Load Failed', 'Failed to load extension builds');
    }
}

function renderBuilds() {
    const list = document.getElementById('builds-list');
    list.innerHTML = '';
    
    // Sort builds by creation date (newest first)
    const sortedBuilds = [...state.builds].sort((a, b) => 
        new Date(b.created_at) - new Date(a.created_at)
    );
    
    sortedBuilds.forEach(build => {
        const buildElement = createBuildElement(build);
        list.appendChild(buildElement);
    });
}

function createBuildElement(build) {
    const div = document.createElement('div');
    div.className = 'build-item';
    
    const statusClass = build.status;
    const statusIcon = {
        building: '⚙️',
        ready: '✅',
        failed: '❌'
    }[build.status] || '❓';
    
    div.innerHTML = `
        <div class="build-header">
            <div>
                <h4>${build.scenario_name}</h4>
                <div class="build-id">${build.build_id}</div>
            </div>
            <div class="build-status ${statusClass}">
                ${statusIcon} ${build.status}
            </div>
        </div>
        
        <div class="build-info">
            <div class="build-info-item">
                <div class="build-info-label">Template</div>
                <div>${build.template_type}</div>
            </div>
            <div class="build-info-item">
                <div class="build-info-label">Created</div>
                <div>${formatDate(build.created_at)}</div>
            </div>
            <div class="build-info-item">
                <div class="build-info-label">Duration</div>
                <div>${formatDuration(build.created_at, build.completed_at)}</div>
            </div>
            <div class="build-info-item">
                <div class="build-info-label">Path</div>
                <div class="text-muted">${build.extension_path}</div>
            </div>
        </div>
        
        ${build.error_log && build.error_log.length > 0 ? `
            <div class="build-errors">
                <h5>Errors:</h5>
                <ul>
                    ${build.error_log.map(error => `<li>${error}</li>`).join('')}
                </ul>
            </div>
        ` : ''}
        
        <div class="build-actions">
            ${build.status === 'ready' ? `
                <button class="btn btn-outline" onclick="testBuild('${build.build_id}')">
                    <span class="icon">🧪</span>
                    Test Extension
                </button>
                <button class="btn btn-outline" onclick="downloadBuild('${build.build_id}')">
                    <span class="icon">📦</span>
                    Download
                </button>
            ` : ''}
            <button class="btn btn-outline" onclick="viewBuildDetails('${build.build_id}')">
                <span class="icon">👁️</span>
                View Details
            </button>
        </div>
    `;
    
    return div;
}

function startBuildPolling(buildId) {
    const pollBuild = async () => {
        try {
            const build = await APIClient.get(`/extension/status/${buildId}`);
            
            // Update the build in state
            const index = state.builds.findIndex(b => b.build_id === buildId);
            if (index >= 0) {
                state.builds[index] = build;
            } else {
                state.builds.unshift(build);
            }
            
            // Re-render builds
            if (state.currentTab === 'builds') {
                renderBuilds();
            }
            
            // Continue polling if still building
            if (build.status === 'building') {
                setTimeout(pollBuild, CONFIG.POLL_INTERVAL);
            } else {
                // Show completion notification
                if (build.status === 'ready') {
                    NotificationManager.success(
                        'Extension Ready',
                        `${build.scenario_name} extension has been generated successfully!`
                    );
                } else if (build.status === 'failed') {
                    NotificationManager.error(
                        'Generation Failed',
                        `${build.scenario_name} extension generation failed. Check build details for errors.`
                    );
                }
            }
            
        } catch (error) {
            log('Build polling error:', error);
            // Don't spam error notifications during polling
        }
    };
    
    pollBuild();
}

function testBuild(buildId) {
    const build = state.builds.find(b => b.build_id === buildId);
    if (build) {
        // Switch to testing tab and populate the extension path
        switchTab('testing');
        document.getElementById('test-extension-path').value = build.extension_path;
    }
}

function downloadBuild(buildId) {
    // TODO: Implement build download functionality
    NotificationManager.info('Coming Soon', 'Build download functionality will be available soon.');
}

function viewBuildDetails(buildId) {
    // TODO: Implement build details modal
    NotificationManager.info('Coming Soon', 'Build details view will be available soon.');
}

// ============================================
// Templates Management
// ============================================

async function loadTemplates() {
    const container = document.getElementById('templates-container');
    const loading = document.getElementById('templates-loading');
    const list = document.getElementById('templates-list');
    
    try {
        loading.classList.remove('hidden');
        list.innerHTML = '';
        
        const response = await APIClient.get('/extension/templates');
        state.templates = response.templates || [];
        
        loading.classList.add('hidden');
        renderTemplates();
        
    } catch (error) {
        log('Failed to load templates:', error);
        loading.classList.add('hidden');
        NotificationManager.error('Load Failed', 'Failed to load extension templates');
    }
}

function renderTemplates() {
    const list = document.getElementById('templates-list');
    list.innerHTML = '';
    
    state.templates.forEach(template => {
        const templateElement = createTemplateElement(template);
        list.appendChild(templateElement);
    });
}

function createTemplateElement(template) {
    const div = document.createElement('div');
    div.className = 'template-detail';
    
    div.innerHTML = `
        <h4>${template.display_name || template.name}</h4>
        <p>${template.description}</p>
        
        <div class="template-files">
            <h5>Generated Files:</h5>
            <div class="file-list">
                ${template.files.map(file => `<span class="file-tag">${file}</span>`).join('')}
            </div>
        </div>
    `;
    
    return div;
}

// ============================================
// Extension Testing
// ============================================

function initializeTesting() {
    const form = document.getElementById('testing-form');
    form.addEventListener('submit', handleExtensionTesting);
}

async function handleExtensionTesting(event) {
    event.preventDefault();
    
    LoadingManager.setButtonLoading('test-btn', true);
    
    try {
        const formData = new FormData(event.target);
        const testSites = document.getElementById('test-sites').value
            .split('\n')
            .map(line => line.trim())
            .filter(line => line.length > 0);
        
        const requestData = {
            extension_path: formData.get('extension_path'),
            test_sites: testSites.length > 0 ? testSites : ['https://example.com'],
            screenshot: formData.get('screenshot') === 'on',
            headless: formData.get('headless') === 'on'
        };
        
        log('Testing extension with config:', requestData);
        
        // Make API request
        const result = await APIClient.post('/extension/test', requestData);
        
        // Show results
        displayTestResults(result);
        
        if (result.success) {
            NotificationManager.success(
                'Tests Passed',
                `All ${result.summary.passed} tests completed successfully!`
            );
        } else {
            NotificationManager.warning(
                'Some Tests Failed',
                `${result.summary.failed} out of ${result.summary.total_tests} tests failed.`
            );
        }
        
    } catch (error) {
        log('Extension testing failed:', error);
        NotificationManager.error(
            'Testing Failed',
            error.message || 'Unknown error occurred during extension testing'
        );
    } finally {
        LoadingManager.setButtonLoading('test-btn', false);
    }
}

function displayTestResults(results) {
    const resultsContainer = document.getElementById('test-results');
    const resultsContent = document.getElementById('test-results-content');
    
    const { summary, test_results } = results;
    
    resultsContent.innerHTML = `
        <div class="test-summary">
            <div class="test-stat">
                <div class="test-stat-value">${summary.total_tests}</div>
                <div class="test-stat-label">Total Tests</div>
            </div>
            <div class="test-stat">
                <div class="test-stat-value" style="color: var(--success-color)">${summary.passed}</div>
                <div class="test-stat-label">Passed</div>
            </div>
            <div class="test-stat">
                <div class="test-stat-value" style="color: var(--error-color)">${summary.failed}</div>
                <div class="test-stat-label">Failed</div>
            </div>
            <div class="test-stat">
                <div class="test-stat-value">${summary.success_rate.toFixed(1)}%</div>
                <div class="test-stat-label">Success Rate</div>
            </div>
        </div>
        
        <div class="test-details">
            <h4>Test Results by Site</h4>
            ${test_results.map(result => `
                <div class="test-site-result">
                    <div class="test-site-info">
                        <h5>${new URL(result.site).hostname}</h5>
                        <div class="test-site-url">${result.site}</div>
                        ${result.errors && result.errors.length > 0 ? `
                            <div class="test-errors">
                                ${result.errors.map(error => `<div class="error-text">${error}</div>`).join('')}
                            </div>
                        ` : ''}
                    </div>
                    <div class="test-result-status">
                        <span class="test-result-icon">${result.loaded ? '✅' : '❌'}</span>
                        <span>${result.loaded ? 'PASS' : 'FAIL'}</span>
                        ${result.load_time_ms ? `<span class="text-muted">(${result.load_time_ms}ms)</span>` : ''}
                    </div>
                </div>
            `).join('')}
        </div>
    `;
    
    resultsContainer.classList.remove('hidden');
}

function browseExtension() {
    // TODO: Implement file browser or provide common paths
    NotificationManager.info('File Browser', 'Enter the path to your extension directory manually for now.');
}

// ============================================
// Initialization
// ============================================

document.addEventListener('DOMContentLoaded', () => {
    log('Extension Generator UI initializing...');
    
    // Initialize components
    initializeTabs();
    initializeGenerator();
    initializeTesting();
    
    // Initial status check
    refreshStatus();
    
    // Set up periodic status updates
    setInterval(refreshStatus, 30000); // Every 30 seconds
    
    log('Extension Generator UI initialized');
});

// ============================================
// Global Functions (exposed to HTML)
// ============================================

window.refreshStatus = refreshStatus;
window.switchTab = switchTab;
window.resetForm = resetForm;
window.loadBuilds = loadBuilds;
window.loadTemplates = loadTemplates;
window.testBuild = testBuild;
window.downloadBuild = downloadBuild;
window.viewBuildDetails = viewBuildDetails;
window.browseExtension = browseExtension;