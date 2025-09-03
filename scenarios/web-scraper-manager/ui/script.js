// Web Scraper Manager Dashboard JavaScript

class WebScraperDashboard {
    constructor() {
        this.apiBase = this.getApiBaseUrl();
        this.currentView = 'dashboard';
        this.agents = [];
        this.jobs = [];
        this.dataResults = [];
        this.platforms = [];
        this.isConnected = false;
        this.refreshInterval = null;

        this.init();
    }

    getApiBaseUrl() {
        // Try to get API URL from environment or use default
        const apiPort = window.location.hostname === 'localhost' ? '8091' : '31750';
        return `http://${window.location.hostname}:${apiPort}`;
    }

    init() {
        this.setupEventListeners();
        this.checkApiConnection();
        this.loadInitialData();
        this.startAutoRefresh();
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const view = link.getAttribute('data-view');
                this.switchView(view);
            });
        });

        // Refresh buttons
        document.getElementById('refresh-all').addEventListener('click', () => {
            this.loadAllData();
        });

        document.getElementById('refresh-jobs').addEventListener('click', () => {
            this.loadJobs();
        });

        // Filter inputs
        document.getElementById('platform-filter').addEventListener('change', () => {
            this.filterAgents();
        });

        document.getElementById('status-filter').addEventListener('change', () => {
            this.filterAgents();
        });

        document.getElementById('search-agents').addEventListener('input', () => {
            this.filterAgents();
        });

        // Data export
        document.getElementById('export-data').addEventListener('click', () => {
            this.exportData();
        });

        // Create agent button
        document.getElementById('create-agent').addEventListener('click', () => {
            this.showCreateAgentModal();
        });
    }

    async checkApiConnection() {
        try {
            const response = await fetch(`${this.apiBase}/health`);
            if (response.ok) {
                this.updateConnectionStatus(true);
                this.isConnected = true;
            } else {
                throw new Error('API not healthy');
            }
        } catch (error) {
            console.error('API connection failed:', error);
            this.updateConnectionStatus(false);
            this.isConnected = false;
        }
    }

    updateConnectionStatus(connected) {
        const statusIndicator = document.querySelector('.status-indicator');
        const statusText = document.querySelector('.status-text');

        if (connected) {
            statusIndicator.className = 'status-indicator connected';
            statusText.textContent = 'Connected';
        } else {
            statusIndicator.className = 'status-indicator error';
            statusText.textContent = 'Connection Error';
            this.showNotification('Failed to connect to API server', 'error');
        }
    }

    switchView(viewName) {
        // Update navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-view="${viewName}"]`).parentElement.classList.add('active');

        // Update views
        document.querySelectorAll('.view').forEach(view => {
            view.classList.remove('active');
        });
        document.getElementById(`${viewName}-view`).classList.add('active');

        this.currentView = viewName;

        // Load view-specific data
        switch (viewName) {
            case 'dashboard':
                this.loadDashboardData();
                break;
            case 'agents':
                this.loadAgents();
                break;
            case 'jobs':
                this.loadJobs();
                break;
            case 'data':
                this.loadDataResults();
                break;
            case 'config':
                this.loadConfiguration();
                break;
            case 'analytics':
                // Analytics view is placeholder for now
                break;
        }
    }

    async loadInitialData() {
        if (!this.isConnected) return;

        try {
            await Promise.all([
                this.loadDashboardData(),
                this.loadPlatforms()
            ]);
        } catch (error) {
            console.error('Failed to load initial data:', error);
        }
    }

    async loadAllData() {
        if (!this.isConnected) return;

        this.showLoading(true);
        try {
            await Promise.all([
                this.loadDashboardData(),
                this.loadAgents(),
                this.loadJobs(),
                this.loadDataResults(),
                this.loadPlatforms()
            ]);
            this.showNotification('Data refreshed successfully', 'success');
        } catch (error) {
            console.error('Failed to refresh data:', error);
            this.showNotification('Failed to refresh data', 'error');
        } finally {
            this.showLoading(false);
        }
    }

    async loadDashboardData() {
        try {
            // Load metrics
            const metricsResponse = await fetch(`${this.apiBase}/api/metrics`);
            if (metricsResponse.ok) {
                const metrics = await metricsResponse.json();
                this.updateDashboardMetrics(metrics.data);
            }

            // Load recent activity (using recent results as activity)
            const resultsResponse = await fetch(`${this.apiBase}/api/results?limit=5`);
            if (resultsResponse.ok) {
                const results = await resultsResponse.json();
                this.updateRecentActivity(results.data);
            }

            // Load platform status
            await this.loadPlatforms();
            this.updatePlatformStatus();

        } catch (error) {
            console.error('Failed to load dashboard data:', error);
        }
    }

    updateDashboardMetrics(metrics) {
        document.getElementById('total-agents').textContent = metrics.total_agents || '0';
        document.getElementById('active-agents').textContent = metrics.active_agents || '0';
        document.getElementById('total-results').textContent = metrics.total_results || '0';
        document.getElementById('running-jobs').textContent = '0'; // TODO: Implement running jobs count
        document.getElementById('queued-jobs').textContent = '0'; // TODO: Implement queued jobs count
        document.getElementById('today-results').textContent = '0'; // TODO: Implement today's results count
        document.getElementById('success-rate').textContent = '0%'; // TODO: Calculate success rate
    }

    updateRecentActivity(results) {
        const activityList = document.getElementById('recent-activity');
        
        if (!results || results.length === 0) {
            activityList.innerHTML = '<div class="activity-item">No recent activity</div>';
            return;
        }

        const activityHTML = results.map(result => {
            const statusClass = result.status === 'success' ? 'success' : 
                               result.status === 'failed' ? 'error' : 'warning';
            const timeAgo = this.getTimeAgo(result.started_at);
            
            return `
                <div class="activity-item ${statusClass}">
                    <div class="activity-content">
                        <strong>Agent ${result.agent_id.substring(0, 8)}</strong>
                        <span class="status-badge ${result.status}">${result.status}</span>
                    </div>
                    <div class="activity-time">${timeAgo}</div>
                </div>
            `;
        }).join('');

        activityList.innerHTML = activityHTML;
    }

    async loadAgents() {
        try {
            const response = await fetch(`${this.apiBase}/api/agents`);
            if (response.ok) {
                const data = await response.json();
                this.agents = data.data || [];
                this.updateAgentsTable();
                this.populateAgentFilters();
            }
        } catch (error) {
            console.error('Failed to load agents:', error);
            document.getElementById('agents-tbody').innerHTML = 
                '<tr><td colspan="7" class="error">Failed to load agents</td></tr>';
        }
    }

    updateAgentsTable() {
        const tbody = document.getElementById('agents-tbody');
        
        if (this.agents.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="loading">No agents found</td></tr>';
            return;
        }

        const agentsHTML = this.agents.map(agent => {
            const lastRun = agent.last_run ? this.formatDate(agent.last_run) : 'Never';
            const nextRun = agent.next_run ? this.formatDate(agent.next_run) : 'Not scheduled';
            
            return `
                <tr>
                    <td>
                        <strong>${agent.name}</strong>
                        ${agent.tags ? agent.tags.map(tag => `<span class="tag">${tag}</span>`).join('') : ''}
                    </td>
                    <td><span class="platform-badge ${agent.platform}">${agent.platform}</span></td>
                    <td>${agent.agent_type || 'Default'}</td>
                    <td><span class="status-badge ${agent.enabled ? 'enabled' : 'disabled'}">${agent.enabled ? 'Enabled' : 'Disabled'}</span></td>
                    <td>${lastRun}</td>
                    <td>${nextRun}</td>
                    <td>
                        <button class="btn btn-secondary btn-sm" onclick="dashboard.executeAgent('${agent.id}')">Run</button>
                        <button class="btn btn-secondary btn-sm" onclick="dashboard.editAgent('${agent.id}')">Edit</button>
                    </td>
                </tr>
            `;
        }).join('');

        tbody.innerHTML = agentsHTML;
    }

    populateAgentFilters() {
        const dataAgentFilter = document.getElementById('data-agent-filter');
        dataAgentFilter.innerHTML = '<option value="">All Agents</option>';
        
        this.agents.forEach(agent => {
            const option = document.createElement('option');
            option.value = agent.id;
            option.textContent = agent.name;
            dataAgentFilter.appendChild(option);
        });
    }

    filterAgents() {
        const platformFilter = document.getElementById('platform-filter').value;
        const statusFilter = document.getElementById('status-filter').value;
        const searchQuery = document.getElementById('search-agents').value.toLowerCase();

        let filteredAgents = this.agents;

        if (platformFilter) {
            filteredAgents = filteredAgents.filter(agent => agent.platform === platformFilter);
        }

        if (statusFilter) {
            const enabled = statusFilter === 'true';
            filteredAgents = filteredAgents.filter(agent => agent.enabled === enabled);
        }

        if (searchQuery) {
            filteredAgents = filteredAgents.filter(agent => 
                agent.name.toLowerCase().includes(searchQuery) ||
                agent.platform.toLowerCase().includes(searchQuery) ||
                (agent.tags && agent.tags.some(tag => tag.toLowerCase().includes(searchQuery)))
            );
        }

        // Temporarily store original agents and update with filtered
        const originalAgents = this.agents;
        this.agents = filteredAgents;
        this.updateAgentsTable();
        this.agents = originalAgents;
    }

    async loadJobs() {
        try {
            const response = await fetch(`${this.apiBase}/api/results?limit=50`);
            if (response.ok) {
                const data = await response.json();
                this.jobs = data.data || [];
                this.updateJobsTable();
                this.updateJobStatusSummary();
            }
        } catch (error) {
            console.error('Failed to load jobs:', error);
            document.getElementById('jobs-tbody').innerHTML = 
                '<tr><td colspan="6" class="error">Failed to load job history</td></tr>';
        }
    }

    updateJobsTable() {
        const tbody = document.getElementById('jobs-tbody');
        
        if (this.jobs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">No job history found</td></tr>';
            return;
        }

        const jobsHTML = this.jobs.map(job => {
            const duration = job.execution_time_ms ? `${job.execution_time_ms}ms` : 'N/A';
            const dataPoints = job.extracted_count || 'N/A';
            
            return `
                <tr>
                    <td>${job.agent_id.substring(0, 8)}</td>
                    <td><span class="status-badge ${job.status}">${job.status}</span></td>
                    <td>${this.formatDate(job.started_at)}</td>
                    <td>${duration}</td>
                    <td>${dataPoints}</td>
                    <td>
                        <button class="btn btn-secondary btn-sm" onclick="dashboard.viewJobDetails('${job.id}')">Details</button>
                    </td>
                </tr>
            `;
        }).join('');

        tbody.innerHTML = jobsHTML;
    }

    updateJobStatusSummary() {
        const statusCounts = {
            pending: 0,
            running: 0,
            success: 0,
            failed: 0
        };

        this.jobs.forEach(job => {
            if (statusCounts[job.status] !== undefined) {
                statusCounts[job.status]++;
            }
        });

        document.getElementById('pending-count').textContent = statusCounts.pending;
        document.getElementById('running-count').textContent = statusCounts.running;
        document.getElementById('success-count').textContent = statusCounts.success;
        document.getElementById('failed-count').textContent = statusCounts.failed;
    }

    async loadDataResults() {
        try {
            const response = await fetch(`${this.apiBase}/api/results`);
            if (response.ok) {
                const data = await response.json();
                this.dataResults = data.data || [];
                this.updateDataTable();
                this.updateDataStats();
            }
        } catch (error) {
            console.error('Failed to load data results:', error);
            document.getElementById('data-tbody').innerHTML = 
                '<tr><td colspan="6" class="error">Failed to load scraped data</td></tr>';
        }
    }

    updateDataTable() {
        const tbody = document.getElementById('data-tbody');
        
        if (this.dataResults.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="loading">No scraped data found</td></tr>';
            return;
        }

        const dataHTML = this.dataResults.map(result => {
            const duration = result.execution_time_ms ? `${result.execution_time_ms}ms` : 'N/A';
            const dataPoints = result.extracted_count || 'N/A';
            
            return `
                <tr>
                    <td>${result.agent_id.substring(0, 8)}</td>
                    <td><span class="status-badge ${result.status}">${result.status}</span></td>
                    <td>${dataPoints}</td>
                    <td>${this.formatDate(result.started_at)}</td>
                    <td>${duration}</td>
                    <td>
                        <button class="btn btn-secondary btn-sm" onclick="dashboard.viewDataDetails('${result.id}')">View</button>
                        <button class="btn btn-secondary btn-sm" onclick="dashboard.exportResult('${result.id}')">Export</button>
                    </td>
                </tr>
            `;
        }).join('');

        tbody.innerHTML = dataHTML;
    }

    updateDataStats() {
        document.getElementById('total-records').textContent = this.dataResults.length;
        document.getElementById('filtered-records').textContent = this.dataResults.length;
        document.getElementById('last-updated').textContent = this.formatDate(new Date());
    }

    async loadPlatforms() {
        try {
            const response = await fetch(`${this.apiBase}/api/platforms`);
            if (response.ok) {
                const data = await response.json();
                this.platforms = data.data || [];
            }
        } catch (error) {
            console.error('Failed to load platforms:', error);
        }
    }

    updatePlatformStatus() {
        const platformGrid = document.getElementById('platform-status');
        
        if (this.platforms.length === 0) {
            platformGrid.innerHTML = '<div class="loading">Loading platform information...</div>';
            return;
        }

        const platformHTML = this.platforms.map(platform => {
            return `
                <div class="platform-card disabled">
                    <h4>${platform.platform}</h4>
                    <p>${platform.best_for}</p>
                    <div class="platform-features">
                        <strong>Capabilities:</strong>
                        ${platform.capabilities ? platform.capabilities.join(', ') : 
                          platform.agent_types ? platform.agent_types.join(', ') : 'N/A'}
                    </div>
                    <div class="platform-status">Status: <span class="status-badge disabled">Not Connected</span></div>
                </div>
            `;
        }).join('');

        platformGrid.innerHTML = platformHTML;
    }

    async loadConfiguration() {
        // Load platform capabilities
        const platformCapabilities = document.getElementById('platform-capabilities');
        
        if (this.platforms.length > 0) {
            const capabilitiesHTML = this.platforms.map(platform => {
                const features = platform.features ? Object.keys(platform.features).join(', ') : 'N/A';
                return `
                    <div class="config-item">
                        <h4>${platform.platform}</h4>
                        <p><strong>Best for:</strong> ${platform.best_for}</p>
                        <p><strong>Features:</strong> ${features}</p>
                    </div>
                `;
            }).join('');
            platformCapabilities.innerHTML = capabilitiesHTML;
        }

        // Load system status
        try {
            const response = await fetch(`${this.apiBase}/api/status`);
            if (response.ok) {
                const data = await response.json();
                this.updateSystemStatus(data.data);
            }
        } catch (error) {
            console.error('Failed to load system status:', error);
        }
    }

    updateSystemStatus(status) {
        const systemStatus = document.getElementById('system-status');
        const statusHTML = `
            <div class="status-item">
                <strong>API Status:</strong> <span class="status-badge ${status.api === 'healthy' ? 'success' : 'error'}">${status.api}</span>
            </div>
            <div class="status-item">
                <strong>Database:</strong> <span class="status-badge ${status.database === 'connected' ? 'success' : 'error'}">${status.database}</span>
            </div>
            <div class="status-item">
                <strong>Version:</strong> ${status.version}
            </div>
            <div class="status-item">
                <strong>Uptime:</strong> ${status.uptime}
            </div>
        `;
        systemStatus.innerHTML = statusHTML;
    }

    // Action handlers
    async executeAgent(agentId) {
        try {
            const response = await fetch(`${this.apiBase}/api/agents/${agentId}/execute`, {
                method: 'POST'
            });
            
            if (response.ok) {
                const data = await response.json();
                this.showNotification(`Agent execution started: ${data.data.run_id}`, 'success');
                // Refresh jobs after a delay
                setTimeout(() => this.loadJobs(), 2000);
            } else {
                throw new Error('Failed to execute agent');
            }
        } catch (error) {
            console.error('Failed to execute agent:', error);
            this.showNotification('Failed to execute agent', 'error');
        }
    }

    editAgent(agentId) {
        this.showNotification('Agent editing is not yet implemented', 'warning');
        // TODO: Implement agent editing modal
    }

    viewJobDetails(jobId) {
        this.showNotification('Job details view is not yet implemented', 'warning');
        // TODO: Implement job details modal
    }

    viewDataDetails(resultId) {
        this.showNotification('Data details view is not yet implemented', 'warning');
        // TODO: Implement data details modal
    }

    exportResult(resultId) {
        this.showNotification('Individual result export is not yet implemented', 'warning');
        // TODO: Implement individual result export
    }

    exportData() {
        this.showNotification('Data export is not yet implemented', 'warning');
        // TODO: Implement bulk data export
    }

    showCreateAgentModal() {
        this.showNotification('Agent creation is not yet implemented', 'warning');
        // TODO: Implement agent creation modal
    }

    // Utility functions
    startAutoRefresh() {
        this.refreshInterval = setInterval(() => {
            if (this.isConnected && this.currentView === 'dashboard') {
                this.loadDashboardData();
            }
        }, 30000); // Refresh every 30 seconds
    }

    showLoading(show) {
        const overlay = document.getElementById('loading-overlay');
        if (show) {
            overlay.classList.add('active');
        } else {
            overlay.classList.remove('active');
        }
    }

    showNotification(message, type = 'info') {
        const notifications = document.getElementById('notifications');
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        notifications.appendChild(notification);
        
        // Remove after 5 seconds
        setTimeout(() => {
            notification.remove();
        }, 5000);
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
    }

    getTimeAgo(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diffInSeconds = Math.floor((now - date) / 1000);

        if (diffInSeconds < 60) {
            return 'Just now';
        } else if (diffInSeconds < 3600) {
            const minutes = Math.floor(diffInSeconds / 60);
            return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
        } else if (diffInSeconds < 86400) {
            const hours = Math.floor(diffInSeconds / 3600);
            return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
        } else {
            const days = Math.floor(diffInSeconds / 86400);
            return `${days} day${days !== 1 ? 's' : ''} ago`;
        }
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new WebScraperDashboard();
});