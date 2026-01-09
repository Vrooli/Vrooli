import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_FLAG = '__documentManagerBridgeInitialized';

function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window.parent === window || window[BRIDGE_FLAG]) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[DocumentManager] Unable to parse parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'document-manager' });
    window[BRIDGE_FLAG] = true;
}

bootstrapIframeBridge();

function ensureIconsReady(attempt = 0) {
    if (typeof window === 'undefined') {
        return;
    }

    const lucide = window.lucide;
    if (lucide?.createIcons) {
        const iconElements = document.querySelectorAll('[data-icon]');
        iconElements.forEach((element) => {
            const iconName = element.getAttribute('data-icon');
            if (!iconName) {
                return;
            }

            if (!element.hasAttribute('data-lucide')) {
                element.setAttribute('data-lucide', iconName);
            }
        });

        lucide.createIcons({
            attrs: {
                'aria-hidden': 'true',
                focusable: 'false'
            }
        });
        return;
    }

    if (attempt < 10) {
        setTimeout(() => ensureIconsReady(attempt + 1), 100 * (attempt + 1));
    } else {
        console.warn('[DocumentManager] Lucide icon library unavailable; icons may not render.');
    }
}

class DocumentManagerApp {
    constructor() {
        this.apiBaseUrl = this.getApiBaseUrl();
        this.currentView = 'dashboard';
        this.data = {
            applications: [],
            agents: [],
            queue: [],
            systemStatus: {}
        };
        
        this.init();
    }

    getApiBaseUrl() {
        if (typeof window === 'undefined') {
            return '';
        }

        if (window.__DOCUMENT_MANAGER_API_BASE_URL) {
            return window.__DOCUMENT_MANAGER_API_BASE_URL;
        }

        return window.location.origin;
    }

    async init() {
        this.setupEventListeners();
        this.showLoadingOverlay();
        
        try {
            await this.loadInitialData();
            this.updateUI();
            this.showToast('System initialized successfully', 'success');
        } catch (error) {
            console.error('Initialization error:', error);
            this.showToast('Failed to initialize system. Check API connection.', 'error');
        } finally {
            this.hideLoadingOverlay();
        }

        // Auto-refresh data every 30 seconds
        setInterval(() => {
            this.refreshCurrentView();
        }, 30000);
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const view = e.currentTarget.getAttribute('data-view');
                if (view) this.switchView(view);
            });
        });

        // Refresh buttons
        document.getElementById('refreshBtn').addEventListener('click', () => this.refreshCurrentView());
        document.getElementById('refreshAppsBtn')?.addEventListener('click', () => this.loadApplications());
        document.getElementById('refreshAgentsBtn')?.addEventListener('click', () => this.loadAgents());
        document.getElementById('refreshQueueBtn')?.addEventListener('click', () => this.loadQueue());

        // Add application buttons
        document.getElementById('addApplicationBtn').addEventListener('click', () => this.showAddApplicationModal());
        document.getElementById('addApplicationBtn2')?.addEventListener('click', () => this.showAddApplicationModal());

        // Modal controls
        document.querySelector('.modal-close').addEventListener('click', () => this.hideAddApplicationModal());
        document.getElementById('cancelAddApp').addEventListener('click', () => this.hideAddApplicationModal());
        document.getElementById('addApplicationForm').addEventListener('submit', (e) => this.handleAddApplication(e));

        // System status
        document.getElementById('systemStatus').addEventListener('click', () => this.checkSystemStatus());

        // Close modal when clicking outside
        document.getElementById('addApplicationModal').addEventListener('click', (e) => {
            if (e.target.id === 'addApplicationModal') {
                this.hideAddApplicationModal();
            }
        });
    }

    async loadInitialData() {
        await Promise.all([
            this.loadApplications(),
            this.loadAgents(),
            this.loadQueue(),
            this.checkSystemStatus()
        ]);
    }

    async apiRequest(endpoint, options = {}) {
        const url = `${this.apiBaseUrl}${endpoint}`;
        const defaultOptions = {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
        };

        try {
            const response = await fetch(url, { ...defaultOptions, ...options });
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            return await response.json();
        } catch (error) {
            console.error(`API request failed for ${endpoint}:`, error);
            throw error;
        }
    }

    async loadApplications() {
        try {
            this.data.applications = await this.apiRequest('/api/applications');
            this.updateApplicationsUI();
        } catch (error) {
            console.error('Failed to load applications:', error);
            this.data.applications = [];
        }
    }

    async loadAgents() {
        try {
            this.data.agents = await this.apiRequest('/api/agents');
            this.updateAgentsUI();
        } catch (error) {
            console.error('Failed to load agents:', error);
            this.data.agents = [];
        }
    }

    async loadQueue() {
        try {
            this.data.queue = await this.apiRequest('/api/queue');
            this.updateQueueUI();
        } catch (error) {
            console.error('Failed to load queue:', error);
            this.data.queue = [];
        }
    }

    async checkSystemStatus() {
        const services = ['db-status', 'vector-status', 'ai-status'];
        const checks = [];

        for (const service of services) {
            try {
                const status = await this.apiRequest(`/api/system/${service}`);
                checks.push(status);
            } catch (error) {
                checks.push({
                    service: service.replace('-status', ''),
                    status: 'unhealthy',
                    details: error.message
                });
            }
        }

        this.data.systemStatus = checks;
        this.updateSystemStatusUI();
    }

    updateUI() {
        this.updateDashboardUI();
        this.updateNavigationCounts();
    }

    updateDashboardUI() {
        // Update metrics
        document.getElementById('totalApps').textContent = this.data.applications.length;
        document.getElementById('activeAgents').textContent = this.data.agents.filter(a => a.enabled).length;
        document.getElementById('pendingItems').textContent = this.data.queue.filter(q => q.status === 'pending').length;
        
        // Calculate average health score
        const activeApps = this.data.applications.filter(a => a.active);
        const avgHealth = activeApps.length > 0 
            ? Math.round(activeApps.reduce((sum, app) => sum + app.health_score, 0) / activeApps.length * 10) / 10
            : 0;
        document.getElementById('avgHealth').textContent = avgHealth;

        // Update activity
        this.updateActivityFeed();
    }

    updateActivityFeed() {
        const activityList = document.getElementById('activityList');
        const activities = [];

        // Add recent applications
        this.data.applications.slice(0, 3).forEach(app => {
            activities.push({
                icon: '📁',
                title: `Application "${app.name}" monitored`,
                time: this.formatRelativeTime(app.created_at)
            });
        });

        // Add recent queue items
        this.data.queue.slice(0, 2).forEach(item => {
            activities.push({
                icon: '🔍',
                title: `Improvement suggested: ${item.title}`,
                time: this.formatRelativeTime(item.created_at)
            });
        });

        if (activities.length === 0) {
            activities.push({
                icon: '🚀',
                title: 'System initialized',
                time: 'Just now'
            });
        }

        activityList.innerHTML = activities.map(activity => `
            <div class="activity-item">
                <div class="activity-icon">${activity.icon}</div>
                <div class="activity-content">
                    <div class="activity-title">${activity.title}</div>
                    <div class="activity-time">${activity.time}</div>
                </div>
            </div>
        `).join('');
    }

    updateNavigationCounts() {
        document.getElementById('appCount').textContent = this.data.applications.length;
        document.getElementById('agentCount').textContent = this.data.agents.length;
        document.getElementById('queueCount').textContent = this.data.queue.filter(q => q.status === 'pending').length;
    }

    updateApplicationsUI() {
        const grid = document.getElementById('applicationsGrid');
        
        if (this.data.applications.length === 0) {
            grid.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">📁</div>
                    <h3>No Applications Yet</h3>
                    <p>Add your first application to start monitoring documentation quality.</p>
                    <button class="btn btn-primary" onclick="app.showAddApplicationModal()">
                        <span class="btn-icon">➕</span>
                        Add Application
                    </button>
                </div>
            `;
            return;
        }

        grid.innerHTML = this.data.applications.map(app => `
            <div class="application-card" onclick="app.viewApplication(${app.id})">
                <div class="app-header-section">
                    <div class="app-info">
                        <h3>${this.escapeHtml(app.name)}</h3>
                        <a href="${this.escapeHtml(app.repository_url)}" 
                           class="app-url" 
                           onclick="event.stopPropagation()" 
                           target="_blank" 
                           rel="noopener">
                            ${this.truncateUrl(app.repository_url)}
                        </a>
                    </div>
                    <div class="app-status ${app.active ? 'active' : 'inactive'}">
                        ${app.active ? '●' : '○'} ${app.active ? 'Active' : 'Inactive'}
                    </div>
                </div>
                <div class="app-metrics">
                    <div class="app-metric">
                        <div class="app-metric-value">${app.health_score.toFixed(1)}</div>
                        <div class="app-metric-label">Health Score</div>
                    </div>
                    <div class="app-metric">
                        <div class="app-metric-value">${app.agent_count || 0}</div>
                        <div class="app-metric-label">Agents</div>
                    </div>
                </div>
                <div class="app-actions">
                    <button class="btn btn-secondary" onclick="event.stopPropagation(); app.configureAgents(${app.id})">
                        Configure Agents
                    </button>
                    <button class="btn btn-primary" onclick="event.stopPropagation(); app.viewReports(${app.id})">
                        View Reports
                    </button>
                </div>
            </div>
        `).join('');
    }

    updateAgentsUI() {
        const grid = document.getElementById('agentsGrid');
        
        if (this.data.agents.length === 0) {
            grid.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">🤖</div>
                    <h3>No Agents Configured</h3>
                    <p>Create AI agents to automatically monitor and improve your documentation.</p>
                    <button class="btn btn-primary" onclick="app.createAgent()">
                        <span class="btn-icon">🤖</span>
                        Create Agent
                    </button>
                </div>
            `;
            return;
        }

        grid.innerHTML = this.data.agents.map(agent => `
            <div class="agent-card">
                <div class="agent-header">
                    <div class="agent-info">
                        <h3>${this.escapeHtml(agent.name)}</h3>
                        <div class="agent-type">${this.escapeHtml(agent.type)}</div>
                    </div>
                    <div class="agent-status ${agent.enabled ? 'enabled' : 'disabled'}">
                        ${agent.enabled ? '●' : '○'} ${agent.enabled ? 'Enabled' : 'Disabled'}
                    </div>
                </div>
                <div class="agent-performance">
                    <div class="performance-score">${agent.last_performance_score.toFixed(1)}</div>
                    <div class="performance-label">Performance Score</div>
                </div>
                <div class="agent-details">
                    <div class="agent-detail">
                        <strong>Application:</strong> ${this.escapeHtml(agent.application_name)}
                    </div>
                    <div class="agent-detail">
                        <strong>Schedule:</strong> ${this.escapeHtml(agent.schedule_cron || 'Manual')}
                    </div>
                    <div class="agent-detail">
                        <strong>Auto-apply threshold:</strong> ${agent.auto_apply_threshold}
                    </div>
                </div>
            </div>
        `).join('');
    }

    updateQueueUI() {
        const container = document.getElementById('queueContainer');
        
        if (this.data.queue.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">✅</div>
                    <h3>All Caught Up!</h3>
                    <p>No pending improvements at the moment. Your documentation is in good shape.</p>
                </div>
            `;
            return;
        }

        container.innerHTML = this.data.queue.map(item => `
            <div class="queue-item ${item.severity}">
                <div class="queue-header">
                    <div>
                        <div class="queue-title">${this.escapeHtml(item.title)}</div>
                        <div class="queue-meta">
                            ${this.escapeHtml(item.application_name)} • ${this.escapeHtml(item.agent_name)} • 
                            ${this.formatRelativeTime(item.created_at)}
                        </div>
                    </div>
                    <div class="queue-severity ${item.severity}">
                        ${item.severity}
                    </div>
                </div>
                <div class="queue-description">
                    ${this.escapeHtml(item.description)}
                </div>
                <div class="queue-actions">
                    <button class="btn btn-secondary" onclick="app.viewDetails(${item.id})">
                        View Details
                    </button>
                    <button class="btn btn-success" onclick="app.approveImprovement(${item.id})">
                        ✅ Approve
                    </button>
                    <button class="btn btn-secondary" onclick="app.rejectImprovement(${item.id})">
                        ❌ Reject
                    </button>
                </div>
            </div>
        `).join('');
    }

    updateSystemStatusUI() {
        const statusElement = document.getElementById('systemStatus');
        const checksContainer = document.getElementById('systemChecks');
        
        const healthyCount = this.data.systemStatus.filter(s => s.status === 'healthy').length;
        const totalCount = this.data.systemStatus.length;
        
        if (healthyCount === totalCount) {
            statusElement.textContent = '🟢';
            statusElement.title = 'All systems operational';
        } else if (healthyCount > 0) {
            statusElement.textContent = '🟡';
            statusElement.title = 'Some systems have issues';
        } else {
            statusElement.textContent = '🔴';
            statusElement.title = 'System issues detected';
        }

        if (checksContainer) {
            checksContainer.innerHTML = this.data.systemStatus.map(check => `
                <div class="system-check">
                    <div class="check-icon ${check.status}">
                        ${check.status === 'healthy' ? '✅' : '❌'}
                    </div>
                    <div class="check-info">
                        <div class="check-name">${this.capitalizeFirst(check.service)}</div>
                        <div class="check-status">
                            ${check.status === 'healthy' ? 'Connected' : check.details || 'Connection failed'}
                        </div>
                    </div>
                </div>
            `).join('');
        }
    }

    switchView(viewName) {
        // Update navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-view="${viewName}"]`).classList.add('active');

        // Update views
        document.querySelectorAll('.view').forEach(view => {
            view.classList.add('hidden');
        });
        document.getElementById(`${viewName}View`).classList.remove('hidden');

        // Update breadcrumb
        document.querySelector('.breadcrumb-item').textContent = this.capitalizeFirst(viewName);

        this.currentView = viewName;
        
        // Load data if needed
        this.refreshCurrentView();
    }

    async refreshCurrentView() {
        this.showLoadingOverlay();
        
        try {
            switch (this.currentView) {
                case 'dashboard':
                    await this.loadInitialData();
                    this.updateUI();
                    break;
                case 'applications':
                    await this.loadApplications();
                    break;
                case 'agents':
                    await this.loadAgents();
                    break;
                case 'queue':
                    await this.loadQueue();
                    break;
                case 'settings':
                    await this.checkSystemStatus();
                    break;
            }
        } catch (error) {
            console.error('Refresh error:', error);
            this.showToast('Failed to refresh data', 'error');
        } finally {
            this.hideLoadingOverlay();
        }
    }

    showAddApplicationModal() {
        document.getElementById('addApplicationModal').classList.add('show');
        document.getElementById('appName').focus();
    }

    hideAddApplicationModal() {
        document.getElementById('addApplicationModal').classList.remove('show');
        document.getElementById('addApplicationForm').reset();
    }

    async handleAddApplication(event) {
        event.preventDefault();
        
        const formData = new FormData(event.target);
        const applicationData = {
            name: document.getElementById('appName').value.trim(),
            repository_url: document.getElementById('repositoryUrl').value.trim(),
            documentation_path: document.getElementById('docPath').value.trim() || 'docs/'
        };

        if (!applicationData.name || !applicationData.repository_url) {
            this.showToast('Please fill in all required fields', 'error');
            return;
        }

        try {
            this.showLoadingOverlay();
            
            await this.apiRequest('/api/applications', {
                method: 'POST',
                body: JSON.stringify(applicationData)
            });

            this.hideAddApplicationModal();
            await this.loadApplications();
            this.updateNavigationCounts();
            this.showToast('Application added successfully!', 'success');
            
        } catch (error) {
            console.error('Failed to add application:', error);
            this.showToast('Failed to add application. Please try again.', 'error');
        } finally {
            this.hideLoadingOverlay();
        }
    }

    // Placeholder methods for future functionality
    viewApplication(id) {
        this.showToast(`Opening application ${id} (feature coming soon)`, 'info');
    }

    configureAgents(appId) {
        this.showToast(`Configure agents for app ${appId} (feature coming soon)`, 'info');
    }

    viewReports(appId) {
        this.showToast(`View reports for app ${appId} (feature coming soon)`, 'info');
    }

    createAgent() {
        this.showToast('Agent creation wizard coming soon', 'info');
    }

    viewDetails(queueId) {
        this.showToast(`View details for queue item ${queueId} (feature coming soon)`, 'info');
    }

    approveImprovement(queueId) {
        this.showToast(`Approve improvement ${queueId} (feature coming soon)`, 'info');
    }

    rejectImprovement(queueId) {
        this.showToast(`Reject improvement ${queueId} (feature coming soon)`, 'info');
    }

    // Utility methods
    showLoadingOverlay() {
        document.getElementById('loadingOverlay').classList.remove('hidden');
    }

    hideLoadingOverlay() {
        document.getElementById('loadingOverlay').classList.add('hidden');
    }

    showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        
        document.getElementById('toastContainer').appendChild(toast);
        
        setTimeout(() => {
            toast.remove();
        }, 4000);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    truncateUrl(url, maxLength = 40) {
        if (url.length <= maxLength) return url;
        return url.substring(0, maxLength) + '...';
    }

    capitalizeFirst(str) {
        return str.charAt(0).toUpperCase() + str.slice(1);
    }

    formatRelativeTime(dateString) {
        if (!dateString) return 'Unknown';
        
        const date = new Date(dateString);
        const now = new Date();
        const diffInSeconds = Math.floor((now - date) / 1000);
        
        if (diffInSeconds < 60) return 'Just now';
        if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`;
        if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`;
        return `${Math.floor(diffInSeconds / 86400)}d ago`;
    }
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    ensureIconsReady();
    window.app = new DocumentManagerApp();
});

// Add some CSS for empty states
const style = document.createElement('style');
style.textContent = `
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 80px 40px;
        text-align: center;
        background: #ffffff;
        border-radius: 12px;
        border: 1px solid #e2e8f0;
        grid-column: 1 / -1;
    }
    
    .empty-icon {
        font-size: 64px;
        margin-bottom: 24px;
        opacity: 0.5;
    }
    
    .empty-state h3 {
        font-size: 24px;
        font-weight: 600;
        margin-bottom: 12px;
        color: #1e293b;
    }
    
    .empty-state p {
        color: #64748b;
        max-width: 400px;
        margin-bottom: 24px;
    }
    
    .agent-details {
        display: flex;
        flex-direction: column;
        gap: 8px;
        font-size: 14px;
        color: #64748b;
    }
    
    .agent-detail strong {
        color: #374151;
    }
`;
document.head.appendChild(style);
