import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

(function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window.parent === window || window.__competitorMonitorBridgeInitialized) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[CompetitorChangeMonitor] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'competitor-change-monitor' });
    window.__competitorMonitorBridgeInitialized = true;
})();

// Competitor Intelligence Monitor - Main JavaScript

// Dynamic API URL configuration
let API_URL = 'http://localhost:8140';

// Load configuration
fetch('/config')
    .then(res => res.json())
    .then(config => {
        API_URL = config.apiUrl;
    })
    .catch(() => {
        // Fallback to default
        API_URL = 'http://localhost:8140';
    });

// State management
let state = {
    currentView: 'dashboard',
    competitors: [],
    alerts: [],
    analyses: [],
    selectedCompetitor: null,
    loading: false
};

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
});

function initializeApp() {
    loadDashboard();
    setupEventListeners();
    startAutoRefresh();
}

// Setup event listeners
function setupEventListeners() {
    // Add competitor button
    const addBtn = document.getElementById('addCompetitor');
    if (addBtn) {
        addBtn.addEventListener('click', showAddCompetitorModal);
    }
    
    // Scan now button
    const scanBtn = document.getElementById('scanNow');
    if (scanBtn) {
        scanBtn.addEventListener('click', triggerManualScan);
    }
    
    // Navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', (e) => {
            const view = e.currentTarget.dataset.view;
            switchView(view);
        });
    });
}

// Load dashboard data
async function loadDashboard() {
    showLoading(true);
    
    try {
        // Load all data in parallel
        const [competitors, alerts, analyses] = await Promise.all([
            fetchCompetitors(),
            fetchAlerts(),
            fetchAnalyses()
        ]);
        
        state.competitors = competitors;
        state.alerts = alerts;
        state.analyses = analyses;
        
        renderDashboard();
    } catch (error) {
        console.error('Failed to load dashboard:', error);
        showError('Failed to load dashboard data');
    } finally {
        showLoading(false);
    }
}

// Fetch competitors
async function fetchCompetitors() {
    const response = await fetch(`${API_URL}/api/competitors`);
    if (!response.ok) throw new Error('Failed to fetch competitors');
    return response.json();
}

// Fetch alerts
async function fetchAlerts(status = null) {
    let url = `${API_URL}/api/alerts`;
    if (status) url += `?status=${status}`;
    
    const response = await fetch(url);
    if (!response.ok) throw new Error('Failed to fetch alerts');
    return response.json();
}

// Fetch analyses
async function fetchAnalyses(competitorId = null) {
    let url = `${API_URL}/api/analyses`;
    if (competitorId) url += `?competitor_id=${competitorId}`;
    
    const response = await fetch(url);
    if (!response.ok) throw new Error('Failed to fetch analyses');
    return response.json();
}

// Render dashboard
function renderDashboard() {
    const container = document.querySelector('.content');
    if (!container) return;
    
    container.innerHTML = `
        <div class="dashboard-grid">
            <!-- Stats Cards -->
            <div class="stats-row">
                <div class="stat-card">
                    <div class="stat-icon">🎯</div>
                    <div class="stat-content">
                        <div class="stat-value">${state.competitors.length}</div>
                        <div class="stat-label">Active Competitors</div>
                    </div>
                </div>
                <div class="stat-card alert-stat">
                    <div class="stat-icon">🚨</div>
                    <div class="stat-content">
                        <div class="stat-value">${state.alerts.filter(a => a.status === 'unread').length}</div>
                        <div class="stat-label">Unread Alerts</div>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-icon">📊</div>
                    <div class="stat-content">
                        <div class="stat-value">${state.analyses.length}</div>
                        <div class="stat-label">Changes Today</div>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="stat-icon">✅</div>
                    <div class="stat-content">
                        <div class="stat-value">${calculateUptime()}%</div>
                        <div class="stat-label">Monitor Uptime</div>
                    </div>
                </div>
            </div>
            
            <!-- Recent Alerts -->
            <div class="alerts-section">
                <h2>🚨 Recent Alerts</h2>
                <div class="alerts-list">
                    ${renderAlertsList()}
                </div>
            </div>
            
            <!-- Competitor Activity -->
            <div class="activity-section">
                <h2>📈 Competitor Activity</h2>
                <div class="activity-timeline">
                    ${renderActivityTimeline()}
                </div>
            </div>
        </div>
    `;
}

// Render alerts list
function renderAlertsList() {
    if (state.alerts.length === 0) {
        return '<div class="empty-state">No alerts to display</div>';
    }
    
    return state.alerts.slice(0, 5).map(alert => `
        <div class="alert-item ${alert.priority}" data-id="${alert.id}">
            <div class="alert-priority">${getPriorityIcon(alert.priority)}</div>
            <div class="alert-content">
                <div class="alert-title">${alert.title}</div>
                <div class="alert-summary">${alert.summary}</div>
                <div class="alert-meta">
                    <span class="alert-time">${formatTime(alert.created_at)}</span>
                    <span class="alert-score">Relevance: ${alert.relevance_score}/100</span>
                </div>
            </div>
            <div class="alert-actions">
                <button onclick="markAlertRead('${alert.id}')" class="btn-small">
                    ${alert.status === 'unread' ? '✓ Mark Read' : '👁 Read'}
                </button>
            </div>
        </div>
    `).join('');
}

// Render activity timeline
function renderActivityTimeline() {
    if (state.analyses.length === 0) {
        return '<div class="empty-state">No recent activity</div>';
    }
    
    return state.analyses.slice(0, 10).map(analysis => `
        <div class="timeline-item">
            <div class="timeline-marker ${analysis.impact_level}"></div>
            <div class="timeline-content">
                <div class="timeline-title">${analysis.change_category} change detected</div>
                <div class="timeline-url">${analysis.target_url}</div>
                <div class="timeline-summary">${analysis.summary}</div>
                <div class="timeline-time">${formatTime(analysis.created_at)}</div>
            </div>
        </div>
    `).join('');
}

// Show add competitor modal
function showAddCompetitorModal() {
    const modal = document.createElement('div');
    modal.className = 'modal';
    modal.innerHTML = `
        <div class="modal-content">
            <h2>Add New Competitor</h2>
            <form id="addCompetitorForm">
                <div class="form-group">
                    <label>Competitor Name</label>
                    <input type="text" id="competitorName" required placeholder="e.g., TechCorp">
                </div>
                <div class="form-group">
                    <label>Description</label>
                    <textarea id="competitorDescription" placeholder="Brief description of the competitor"></textarea>
                </div>
                <div class="form-group">
                    <label>Category</label>
                    <select id="competitorCategory">
                        <option value="direct">Direct Competitor</option>
                        <option value="indirect">Indirect Competitor</option>
                        <option value="aspirational">Aspirational</option>
                        <option value="emerging">Emerging</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Importance</label>
                    <select id="competitorImportance">
                        <option value="low">Low</option>
                        <option value="medium" selected>Medium</option>
                        <option value="high">High</option>
                        <option value="critical">Critical</option>
                    </select>
                </div>
                <div class="modal-actions">
                    <button type="button" onclick="closeModal()" class="btn-secondary">Cancel</button>
                    <button type="submit" class="btn-primary">Add Competitor</button>
                </div>
            </form>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    document.getElementById('addCompetitorForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        await addCompetitor();
    });
}

// Add competitor
async function addCompetitor() {
    const name = document.getElementById('competitorName').value;
    const description = document.getElementById('competitorDescription').value;
    const category = document.getElementById('competitorCategory').value;
    const importance = document.getElementById('competitorImportance').value;
    
    try {
        const response = await fetch(`${API_URL}/api/competitors`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                name,
                description,
                category,
                importance
            })
        });
        
        if (!response.ok) throw new Error('Failed to add competitor');
        
        const competitor = await response.json();
        state.competitors.push(competitor);
        
        showSuccess('Competitor added successfully!');
        closeModal();
        loadDashboard();
    } catch (error) {
        showError('Failed to add competitor');
    }
}

// Trigger manual scan
async function triggerManualScan() {
    const btn = document.getElementById('scanNow');
    btn.disabled = true;
    btn.textContent = 'Scanning...';
    
    try {
        const response = await fetch(`${API_URL}/api/scan`, {
            method: 'POST'
        });
        
        if (!response.ok) throw new Error('Failed to trigger scan');
        
        const result = await response.json();
        showSuccess(`Scan completed: ${result.message || 'Success'}`);
        
        // Reload data
        setTimeout(() => loadDashboard(), 2000);
    } catch (error) {
        showError('Failed to trigger scan');
    } finally {
        btn.disabled = false;
        btn.innerHTML = `
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <polyline points="23 4 23 10 17 10"/>
                <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
            </svg>
            Scan Now
        `;
    }
}

// Mark alert as read
async function markAlertRead(alertId) {
    try {
        const response = await fetch(`${API_URL}/api/alerts/${alertId}`, {
            method: 'PATCH',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                status: 'read'
            })
        });
        
        if (!response.ok) throw new Error('Failed to update alert');
        
        // Update local state
        const alert = state.alerts.find(a => a.id === alertId);
        if (alert) alert.status = 'read';
        
        renderDashboard();
    } catch (error) {
        showError('Failed to update alert');
    }
}

// Switch view
function switchView(view) {
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });
    document.querySelector(`[data-view="${view}"]`).classList.add('active');
    
    state.currentView = view;
    
    switch(view) {
        case 'overview':
            loadDashboard();
            break;
        case 'competitors':
            loadCompetitorsView();
            break;
        case 'alerts':
            loadAlertsView();
            break;
        case 'changes':
            loadChangesView();
            break;
        case 'settings':
            loadSettingsView();
            break;
    }
}

// Auto-refresh
function startAutoRefresh() {
    setInterval(() => {
        if (state.currentView === 'overview') {
            loadDashboard();
        }
    }, 30000); // Refresh every 30 seconds
}

// Utility functions
function getPriorityIcon(priority) {
    const icons = {
        critical: '🔴',
        high: '🟠',
        medium: '🟡',
        low: '🟢'
    };
    return icons[priority] || '⚪';
}

function formatTime(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;
    
    if (diff < 3600000) {
        return `${Math.floor(diff / 60000)} minutes ago`;
    } else if (diff < 86400000) {
        return `${Math.floor(diff / 3600000)} hours ago`;
    } else {
        return date.toLocaleDateString();
    }
}

function calculateUptime() {
    return 99.7; // Placeholder
}

function showLoading(show) {
    state.loading = show;
    // Implement loading indicator
}

function showError(message) {
    console.error(message);
    // Implement error toast
}

function showSuccess(message) {
    console.log(message);
    // Implement success toast
}

function closeModal() {
    const modal = document.querySelector('.modal');
    if (modal) modal.remove();
}
