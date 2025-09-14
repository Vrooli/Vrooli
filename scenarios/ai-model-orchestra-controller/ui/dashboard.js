// Global state
let currentSection = 'overview';
let refreshInterval;

// Initialize dashboard
document.addEventListener('DOMContentLoaded', function() {
    initializeNavigation();
    startAutoRefresh();
    loadInitialData();
});

// Navigation handling
function initializeNavigation() {
    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const section = this.getAttribute('data-section');
            switchSection(section);
        });
    });
}

function switchSection(section) {
    // Update active nav link
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
    });
    document.querySelector(`[data-section="${section}"]`).classList.add('active');

    // Show/hide sections
    document.querySelectorAll('.content-section').forEach(sec => {
        sec.classList.add('hidden');
    });
    document.getElementById(`${section}-section`).classList.remove('hidden');

    currentSection = section;
}

// Data loading functions
async function loadInitialData() {
    await Promise.all([
        loadOverviewData(),
        loadModelsData(),
        loadResourcesData(),
        loadAlertsData()
    ]);
}

async function loadOverviewData() {
    try {
        const response = await fetch('/api/health');
        const data = await response.json();
        
        updateSystemStatus(data);
        
        // Load additional metrics
        const metricsResponse = await fetch('/api/ai/resources/metrics');
        const metricsData = await metricsResponse.json();
        
        updateOverviewMetrics(metricsData);
        
    } catch (error) {
        showError('Failed to load overview data: ' + error.message);
    }
}

async function loadModelsData() {
    try {
        const response = await fetch('/api/ai/models/status');
        const data = await response.json();
        
        updateModelsDisplay(data);
        
    } catch (error) {
        console.error('Failed to load models data:', error);
    }
}

async function loadResourcesData() {
    try {
        const response = await fetch('/api/ai/resources/metrics');
        const data = await response.json();
        
        updateResourcesDisplay(data);
        
    } catch (error) {
        console.error('Failed to load resources data:', error);
    }
}

async function loadAlertsData() {
    // Mock alerts data - in real implementation, this would fetch from an alerts API
    const mockAlerts = [
        {
            type: 'info',
            message: 'System initialized successfully',
            time: new Date().toISOString()
        }
    ];
    
    updateAlertsDisplay(mockAlerts);
}

// Update functions
function updateSystemStatus(data) {
    const statusElement = document.getElementById('system-status');
    const isHealthy = data.status === 'healthy';
    
    statusElement.textContent = isHealthy ? 'System Healthy' : 'System Issues';
    statusElement.className = `status-badge ${isHealthy ? 'status-healthy' : 'status-error'}`;
}

function updateOverviewMetrics(data) {
    // Update basic metrics
    if (data.requests && data.requests.length > 0) {
        const totalRequests = data.requests.reduce((sum, r) => sum + r.request_count, 0);
        const successCount = data.requests.reduce((sum, r) => sum + r.success_count, 0);
        const avgResponseTime = data.requests.reduce((sum, r) => sum + r.avg_response_time, 0) / data.requests.length;
        
        document.getElementById('total-requests').textContent = totalRequests.toLocaleString();
        document.getElementById('success-rate').textContent = totalRequests > 0 ? Math.round((successCount / totalRequests) * 100) + '%' : '100%';
        document.getElementById('avg-response-time').textContent = Math.round(avgResponseTime) + 'ms';
    }

    // Update system metrics
    if (data.current) {
        const memoryPressure = data.current.memoryPressure || 0;
        const cpuUsage = data.current.cpu?.usage || 0;
        
        document.getElementById('memory-pressure').textContent = Math.round(memoryPressure * 100) + '%';
        document.getElementById('cpu-usage').textContent = Math.round(cpuUsage) + '%';
        
        // Update progress bars
        const memoryProgress = document.getElementById('memory-progress');
        const cpuProgress = document.getElementById('cpu-progress');
        
        memoryProgress.style.width = (memoryPressure * 100) + '%';
        memoryProgress.className = `progress-fill ${memoryPressure > 0.8 ? 'danger' : memoryPressure > 0.6 ? 'warning' : ''}`;
        
        cpuProgress.style.width = cpuUsage + '%';
        cpuProgress.className = `progress-fill ${cpuUsage > 80 ? 'danger' : cpuUsage > 60 ? 'warning' : ''}`;
    }
}

function updateModelsDisplay(data) {
    const modelsGrid = document.getElementById('models-grid');
    modelsGrid.innerHTML = '';
    
    document.getElementById('active-models').textContent = data.healthyModels || 0;
    
    if (data.models && data.models.length > 0) {
        data.models.forEach(model => {
            const modelCard = createModelCard(model);
            modelsGrid.appendChild(modelCard);
        });
    } else {
        modelsGrid.innerHTML = '<div style="color: #a0aec0; text-align: center; grid-column: 1 / -1;">No models available</div>';
    }
}

function createModelCard(model) {
    const card = document.createElement('div');
    card.className = 'model-card';
    
    const successRate = model.request_count > 0 ? Math.round((model.success_count / model.request_count) * 100) : 100;
    
    card.innerHTML = `
        <div class="model-name">${model.model_name}</div>
        <div class="model-stats">
            <div class="model-stat">
                <div class="model-stat-value">${model.request_count || 0}</div>
                <div class="model-stat-label">Requests</div>
            </div>
            <div class="model-stat">
                <div class="model-stat-value">${successRate}%</div>
                <div class="model-stat-label">Success</div>
            </div>
            <div class="model-stat">
                <div class="model-stat-value">${Math.round(model.avg_response_time_ms || 0)}</div>
                <div class="model-stat-label">Avg MS</div>
            </div>
        </div>
        <div class="progress-bar">
            <div class="progress-fill" style="width: ${Math.min(model.current_load * 100, 100)}%"></div>
        </div>
    `;
    
    return card;
}

function updateResourcesDisplay(data) {
    if (data.current && data.current.memory) {
        document.getElementById('memory-available').textContent = data.current.memory.available_gb.toFixed(1) + 'GB';
        document.getElementById('memory-total').textContent = data.current.memory.total_gb.toFixed(1) + 'GB';
    }
    
    if (data.current && data.current.cpu) {
        document.getElementById('cpu-load').textContent = data.current.cpu.load.toFixed(2);
    }
    
    // Mock swap usage
    document.getElementById('swap-usage').textContent = '5%';
}

function updateAlertsDisplay(alerts) {
    const alertsContainer = document.getElementById('alerts-container');
    alertsContainer.innerHTML = '';
    
    if (alerts.length === 0) {
        alertsContainer.innerHTML = '<div style="color: #a0aec0; text-align: center;">No alerts</div>';
        return;
    }
    
    alerts.forEach(alert => {
        const alertElement = createAlertElement(alert);
        alertsContainer.appendChild(alertElement);
    });
}

function createAlertElement(alert) {
    const element = document.createElement('div');
    element.className = `alert-item alert-${alert.type}`;
    
    const icons = {
        critical: '🚨',
        warning: '⚠️',
        info: 'ℹ️'
    };
    
    element.innerHTML = `
        <div class="alert-icon">${icons[alert.type] || 'ℹ️'}</div>
        <div class="alert-content">
            <div class="alert-message">${alert.message}</div>
            <div class="alert-time">${new Date(alert.time).toLocaleString()}</div>
        </div>
    `;
    
    return element;
}

// Refresh functions
async function refreshData() {
    const refreshIcon = document.getElementById('refresh-icon');
    refreshIcon.innerHTML = '<span class="loading"></span>';
    
    try {
        await loadInitialData();
    } finally {
        refreshIcon.innerHTML = '🔄';
    }
}

async function refreshModels() {
    await loadModelsData();
}

async function refreshResources() {
    await loadResourcesData();
}

async function refreshRequests() {
    // Mock implementation
    console.log('Refreshing requests data...');
}

async function refreshAlerts() {
    await loadAlertsData();
}

// Auto refresh
function startAutoRefresh() {
    refreshInterval = setInterval(async () => {
        if (currentSection === 'overview') {
            await loadOverviewData();
        } else if (currentSection === 'models') {
            await loadModelsData();
        } else if (currentSection === 'resources') {
            await loadResourcesData();
        }
    }, 30000); // Refresh every 30 seconds
}

// Error handling
function showError(message) {
    const errorContainer = document.getElementById('error-container');
    errorContainer.innerHTML = `<div class="error-message">⚠️ ${message}</div>`;
    
    // Auto-hide error after 5 seconds
    setTimeout(() => {
        errorContainer.innerHTML = '';
    }, 5000);
}