// App Issue Tracker Dashboard - JavaScript

class IssueTracker {
    constructor() {
        this.apiBase = 'http://localhost:8090/api';
        this.currentPage = 'dashboard';
        this.statusChart = null;
        this.priorityChart = null;
        this.issues = [];
        this.agents = [];
        this.apps = [];
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadDashboard();
        this.setupCharts();
        this.startAutoRefresh();
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const page = e.currentTarget.dataset.page;
                this.navigateTo(page);
            });
        });

        // Refresh button
        document.getElementById('refreshBtn')?.addEventListener('click', () => {
            this.refreshCurrentPage();
        });

        // New issue modal
        document.getElementById('newIssueBtn')?.addEventListener('click', () => {
            this.showNewIssueModal();
        });

        // Modal close buttons
        document.querySelectorAll('.modal-close').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.closeModal(e.target.closest('.modal'));
            });
        });

        // New issue form
        document.getElementById('newIssueForm')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.createIssue();
        });

        // Cancel button
        document.getElementById('cancelBtn')?.addEventListener('click', () => {
            this.closeModal(document.getElementById('newIssueModal'));
        });

        // Filters
        document.getElementById('statusFilter')?.addEventListener('change', () => {
            this.filterIssues();
        });
        document.getElementById('priorityFilter')?.addEventListener('change', () => {
            this.filterIssues();
        });
        document.getElementById('typeFilter')?.addEventListener('change', () => {
            this.filterIssues();
        });

        // Search
        document.getElementById('searchBtn')?.addEventListener('click', () => {
            this.performSearch();
        });
        document.getElementById('searchInput')?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.performSearch();
            }
        });

        // Investigation button
        document.getElementById('investigateBtn')?.addEventListener('click', () => {
            this.startInvestigation();
        });

        // Close modals on background click
        document.querySelectorAll('.modal').forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.closeModal(modal);
                }
            });
        });
    }

    async apiCall(endpoint, options = {}) {
        this.showLoading();
        try {
            const response = await fetch(`${this.apiBase}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });
            
            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.message || `API Error: ${response.status}`);
            }
            
            return data;
        } catch (error) {
            console.error('API call failed:', error);
            this.showNotification(error.message, 'error');
            throw error;
        } finally {
            this.hideLoading();
        }
    }

    navigateTo(page) {
        // Update active nav link
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.remove('active');
        });
        document.querySelector(`[data-page="${page}"]`)?.classList.add('active');

        // Show page
        document.querySelectorAll('.page').forEach(p => {
            p.classList.remove('active');
        });
        document.getElementById(page)?.classList.add('active');

        this.currentPage = page;
        this.loadPageContent(page);
    }

    async loadPageContent(page) {
        switch (page) {
            case 'dashboard':
                await this.loadDashboard();
                break;
            case 'issues':
                await this.loadAllIssues();
                break;
            case 'agents':
                await this.loadAgents();
                break;
            case 'apps':
                await this.loadApps();
                break;
            case 'search':
                // Search page is interactive, no initial load needed
                break;
        }
    }

    async loadDashboard() {
        try {
            // Load stats
            const statsResponse = await this.apiCall('/stats');
            if (statsResponse.success) {
                this.updateStats(statsResponse.data.stats);
            }

            // Load recent issues
            const issuesResponse = await this.apiCall('/issues?limit=10');
            if (issuesResponse.success) {
                this.displayRecentIssues(issuesResponse.data.issues);
                this.updateCharts(issuesResponse.data.issues);
            }

            this.updateLastUpdate();
        } catch (error) {
            console.error('Failed to load dashboard:', error);
        }
    }

    updateStats(stats) {
        document.getElementById('totalIssues').textContent = stats.total_issues || 0;
        document.getElementById('openIssues').textContent = stats.open_issues || 0;
        document.getElementById('inProgress').textContent = stats.in_progress || 0;
        document.getElementById('fixedToday').textContent = stats.fixed_today || 0;
    }

    displayRecentIssues(issues) {
        const tbody = document.querySelector('#recentIssuesTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';
        
        issues.forEach(issue => {
            const row = this.createIssueRow(issue);
            tbody.appendChild(row);
        });
    }

    createIssueRow(issue) {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td><code>${this.truncateId(issue.id)}</code></td>
            <td><strong>${this.escapeHtml(issue.title)}</strong></td>
            <td><span class="status-badge status-${issue.status}">${issue.status}</span></td>
            <td><span class="status-badge priority-${issue.priority}">${issue.priority}</span></td>
            <td>${issue.type}</td>
            <td>${this.formatDate(issue.created_at)}</td>
            <td>
                <button class="btn btn-primary" onclick="tracker.viewIssue('${issue.id}')">
                    <i class="fas fa-eye"></i>
                </button>
                ${issue.status === 'open' ? `
                    <button class="btn btn-success" onclick="tracker.investigateIssue('${issue.id}')">
                        <i class="fas fa-search"></i>
                    </button>
                ` : ''}
            </td>
        `;
        return row;
    }

    async loadAllIssues() {
        try {
            const response = await this.apiCall('/issues?limit=100');
            if (response.success) {
                this.issues = response.data.issues;
                this.displayAllIssues(this.issues);
            }
        } catch (error) {
            console.error('Failed to load issues:', error);
        }
    }

    displayAllIssues(issues) {
        const tbody = document.querySelector('#allIssuesTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';
        
        issues.forEach(issue => {
            const row = this.createDetailedIssueRow(issue);
            tbody.appendChild(row);
        });
    }

    createDetailedIssueRow(issue) {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td><code>${this.truncateId(issue.id)}</code></td>
            <td><strong>${this.escapeHtml(issue.title)}</strong></td>
            <td><span class="status-badge status-${issue.status}">${issue.status}</span></td>
            <td><span class="status-badge priority-${issue.priority}">${issue.priority}</span></td>
            <td>${issue.type}</td>
            <td>${issue.app_name || 'Unknown'}</td>
            <td>${this.formatDate(issue.created_at)}</td>
            <td>
                <button class="btn btn-primary" onclick="tracker.viewIssue('${issue.id}')">
                    <i class="fas fa-eye"></i>
                </button>
                ${issue.status === 'open' ? `
                    <button class="btn btn-success" onclick="tracker.investigateIssue('${issue.id}')">
                        <i class="fas fa-search"></i>
                    </button>
                ` : ''}
            </td>
        `;
        return row;
    }

    filterIssues() {
        const statusFilter = document.getElementById('statusFilter')?.value;
        const priorityFilter = document.getElementById('priorityFilter')?.value;
        const typeFilter = document.getElementById('typeFilter')?.value;

        let filteredIssues = this.issues;

        if (statusFilter) {
            filteredIssues = filteredIssues.filter(issue => issue.status === statusFilter);
        }
        if (priorityFilter) {
            filteredIssues = filteredIssues.filter(issue => issue.priority === priorityFilter);
        }
        if (typeFilter) {
            filteredIssues = filteredIssues.filter(issue => issue.type === typeFilter);
        }

        this.displayAllIssues(filteredIssues);
    }

    async loadAgents() {
        try {
            const response = await this.apiCall('/agents');
            if (response.success) {
                this.agents = response.data.agents;
                this.displayAgents(this.agents);
            }
        } catch (error) {
            console.error('Failed to load agents:', error);
        }
    }

    displayAgents(agents) {
        const grid = document.getElementById('agentsGrid');
        if (!grid) return;

        grid.innerHTML = '';

        agents.forEach(agent => {
            const card = document.createElement('div');
            card.className = 'agent-card';
            card.innerHTML = `
                <h4>${agent.display_name}</h4>
                <p>${agent.description || 'No description available'}</p>
                <div class="agent-stats">
                    <div class="stat">
                        <div class="stat-value">${agent.total_runs}</div>
                        <div class="stat-label">Total Runs</div>
                    </div>
                    <div class="stat">
                        <div class="stat-value">${Math.round(agent.success_rate)}%</div>
                        <div class="stat-label">Success Rate</div>
                    </div>
                    <div class="stat">
                        <div class="stat-value">${agent.is_active ? 'Active' : 'Inactive'}</div>
                        <div class="stat-label">Status</div>
                    </div>
                </div>
                <div class="agent-capabilities">
                    <strong>Capabilities:</strong>
                    <div style="margin-top: 8px;">
                        ${agent.capabilities?.map(cap => 
                            `<span class="status-badge">${cap}</span>`
                        ).join(' ') || 'No capabilities listed'}
                    </div>
                </div>
            `;
            grid.appendChild(card);
        });
    }

    async loadApps() {
        try {
            const response = await this.apiCall('/apps');
            if (response.success) {
                this.apps = response.data.apps;
                this.displayApps(this.apps);
            }
        } catch (error) {
            console.error('Failed to load apps:', error);
        }
    }

    displayApps(apps) {
        const grid = document.getElementById('appsGrid');
        if (!grid) return;

        grid.innerHTML = '';

        apps.forEach(app => {
            const card = document.createElement('div');
            card.className = 'app-card';
            card.innerHTML = `
                <h4>${app.display_name}</h4>
                <p>Type: ${app.type} | Status: <span class="status-badge status-${app.status}">${app.status}</span></p>
                <div class="app-stats">
                    <div class="stat">
                        <div class="stat-value">${app.total_issues}</div>
                        <div class="stat-label">Total Issues</div>
                    </div>
                    <div class="stat">
                        <div class="stat-value">${app.open_issues}</div>
                        <div class="stat-label">Open Issues</div>
                    </div>
                </div>
                ${app.scenario_name ? `<p><strong>Scenario:</strong> ${app.scenario_name}</p>` : ''}
            `;
            grid.appendChild(card);
        });
    }

    async performSearch() {
        const query = document.getElementById('searchInput')?.value;
        const searchType = document.querySelector('input[name="searchType"]:checked')?.value || 'text';
        
        if (!query?.trim()) {
            this.showNotification('Please enter a search query', 'warning');
            return;
        }

        try {
            let endpoint = '/issues/search';
            if (searchType === 'vector') {
                endpoint = '/search/vector';
            }
            
            const response = await this.apiCall(`${endpoint}?q=${encodeURIComponent(query)}&limit=20`);
            
            if (response.success) {
                this.displaySearchResults(response.data.results, searchType);
            }
        } catch (error) {
            console.error('Search failed:', error);
        }
    }

    displaySearchResults(results, searchType) {
        const container = document.getElementById('searchResults');
        if (!container) return;

        container.innerHTML = '';

        if (!results || results.length === 0) {
            container.innerHTML = '<p style="padding: 24px; text-align: center; color: var(--text-secondary);">No results found</p>';
            return;
        }

        results.forEach(result => {
            const resultDiv = document.createElement('div');
            resultDiv.className = 'search-result';
            resultDiv.innerHTML = `
                <div class="result-title">${this.escapeHtml(result.title)}</div>
                <div class="result-description">${this.escapeHtml(result.description || '')}</div>
                <div class="result-meta">
                    <span class="status-badge status-${result.status}">${result.status}</span>
                    <span class="status-badge priority-${result.priority}">${result.priority}</span>
                    <span>${result.type}</span>
                    ${searchType === 'vector' && result.similarity ? 
                        `<span class="similarity-score">Similarity: ${Math.round(result.similarity * 100)}%</span>` 
                        : ''
                    }
                </div>
            `;
            
            resultDiv.addEventListener('click', () => {
                this.viewIssue(result.id || result.issue_id);
            });
            
            container.appendChild(resultDiv);
        });
    }

    async viewIssue(issueId) {
        try {
            // For now, we'll show a simple modal with issue details
            // In a real implementation, you'd fetch full issue details
            const issue = this.issues.find(i => i.id === issueId);
            if (issue) {
                this.showIssueModal(issue);
            } else {
                this.showNotification('Issue not found', 'error');
            }
        } catch (error) {
            console.error('Failed to view issue:', error);
        }
    }

    showIssueModal(issue) {
        const modal = document.getElementById('issueModal');
        const title = document.getElementById('issueModalTitle');
        const body = document.getElementById('issueModalBody');
        
        title.textContent = `Issue: ${issue.title}`;
        body.innerHTML = `
            <div style="margin-bottom: 16px;">
                <strong>ID:</strong> <code>${issue.id}</code>
            </div>
            <div style="margin-bottom: 16px;">
                <strong>Status:</strong> <span class="status-badge status-${issue.status}">${issue.status}</span>
                <strong style="margin-left: 16px;">Priority:</strong> <span class="status-badge priority-${issue.priority}">${issue.priority}</span>
                <strong style="margin-left: 16px;">Type:</strong> ${issue.type}
            </div>
            <div style="margin-bottom: 16px;">
                <strong>Description:</strong><br>
                <p style="margin-top: 8px; padding: 12px; background: var(--bg-secondary); border-radius: var(--border-radius);">
                    ${this.escapeHtml(issue.description || 'No description provided')}
                </p>
            </div>
            ${issue.error_message ? `
                <div style="margin-bottom: 16px;">
                    <strong>Error Message:</strong><br>
                    <pre style="margin-top: 8px; padding: 12px; background: var(--bg-secondary); border-radius: var(--border-radius); font-size: 12px; overflow-x: auto;">${this.escapeHtml(issue.error_message)}</pre>
                </div>
            ` : ''}
            ${issue.stack_trace ? `
                <div style="margin-bottom: 16px;">
                    <strong>Stack Trace:</strong><br>
                    <pre style="margin-top: 8px; padding: 12px; background: var(--bg-secondary); border-radius: var(--border-radius); font-size: 11px; overflow-x: auto; max-height: 200px; overflow-y: auto;">${this.escapeHtml(issue.stack_trace)}</pre>
                </div>
            ` : ''}
            ${issue.investigation_report ? `
                <div style="margin-bottom: 16px;">
                    <strong>Investigation Report:</strong><br>
                    <div style="margin-top: 8px; padding: 12px; background: var(--bg-secondary); border-radius: var(--border-radius);">
                        ${this.escapeHtml(issue.investigation_report)}
                    </div>
                </div>
            ` : ''}
            <div style="margin-bottom: 16px;">
                <strong>Created:</strong> ${this.formatDate(issue.created_at)}
            </div>
        `;
        
        // Store the current issue for investigation
        this.currentIssue = issue;
        
        modal.classList.add('active');
    }

    async investigateIssue(issueId) {
        try {
            const response = await this.apiCall('/investigate', {
                method: 'POST',
                body: JSON.stringify({
                    issue_id: issueId,
                    priority: 'high'
                })
            });
            
            if (response.success) {
                this.showNotification('Investigation started successfully!', 'success');
                // Refresh the current page to show updated status
                setTimeout(() => {
                    this.refreshCurrentPage();
                }, 2000);
            }
        } catch (error) {
            console.error('Failed to start investigation:', error);
        }
    }

    async startInvestigation() {
        if (!this.currentIssue) return;
        
        await this.investigateIssue(this.currentIssue.id);
        this.closeModal(document.getElementById('issueModal'));
    }

    showNewIssueModal() {
        document.getElementById('newIssueModal').classList.add('active');
    }

    async createIssue() {
        const form = document.getElementById('newIssueForm');
        const formData = new FormData(form);
        
        const issue = {
            title: document.getElementById('issueTitle').value,
            description: document.getElementById('issueDescription').value,
            type: document.getElementById('issueType').value,
            priority: document.getElementById('issuePriority').value,
            error_message: document.getElementById('errorMessage').value,
            stack_trace: document.getElementById('stackTrace').value,
            tags: document.getElementById('issueTags').value.split(',').map(t => t.trim()).filter(t => t),
            app_token: 'dashboard-ui'
        };
        
        try {
            const response = await this.apiCall('/issues', {
                method: 'POST',
                body: JSON.stringify(issue)
            });
            
            if (response.success) {
                this.showNotification('Issue created successfully!', 'success');
                this.closeModal(document.getElementById('newIssueModal'));
                form.reset();
                this.refreshCurrentPage();
            }
        } catch (error) {
            console.error('Failed to create issue:', error);
        }
    }

    closeModal(modal) {
        modal?.classList.remove('active');
    }

    setupCharts() {
        // Status chart
        const statusCtx = document.getElementById('statusChart');
        if (statusCtx) {
            this.statusChart = new Chart(statusCtx, {
                type: 'doughnut',
                data: {
                    labels: ['Open', 'Investigating', 'In Progress', 'Fixed', 'Closed'],
                    datasets: [{
                        data: [0, 0, 0, 0, 0],
                        backgroundColor: [
                            '#ef4444',
                            '#f59e0b',
                            '#2563eb',
                            '#10b981',
                            '#64748b'
                        ]
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'bottom'
                        }
                    }
                }
            });
        }

        // Priority chart
        const priorityCtx = document.getElementById('priorityChart');
        if (priorityCtx) {
            this.priorityChart = new Chart(priorityCtx, {
                type: 'bar',
                data: {
                    labels: ['Critical', 'High', 'Medium', 'Low'],
                    datasets: [{
                        data: [0, 0, 0, 0],
                        backgroundColor: [
                            '#dc2626',
                            '#f59e0b',
                            '#2563eb',
                            '#10b981'
                        ]
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            display: false
                        }
                    },
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });
        }
    }

    updateCharts(issues) {
        if (!issues || issues.length === 0) return;

        // Update status chart
        if (this.statusChart) {
            const statusCounts = {
                open: 0,
                investigating: 0,
                in_progress: 0,
                fixed: 0,
                closed: 0
            };
            
            issues.forEach(issue => {
                statusCounts[issue.status] = (statusCounts[issue.status] || 0) + 1;
            });
            
            this.statusChart.data.datasets[0].data = [
                statusCounts.open,
                statusCounts.investigating,
                statusCounts.in_progress,
                statusCounts.fixed,
                statusCounts.closed
            ];
            this.statusChart.update();
        }

        // Update priority chart
        if (this.priorityChart) {
            const priorityCounts = {
                critical: 0,
                high: 0,
                medium: 0,
                low: 0
            };
            
            issues.forEach(issue => {
                priorityCounts[issue.priority] = (priorityCounts[issue.priority] || 0) + 1;
            });
            
            this.priorityChart.data.datasets[0].data = [
                priorityCounts.critical,
                priorityCounts.high,
                priorityCounts.medium,
                priorityCounts.low
            ];
            this.priorityChart.update();
        }
    }

    refreshCurrentPage() {
        this.loadPageContent(this.currentPage);
    }

    startAutoRefresh() {
        // Refresh dashboard every 30 seconds if on dashboard page
        setInterval(() => {
            if (this.currentPage === 'dashboard') {
                this.loadDashboard();
            }
        }, 30000);
    }

    updateLastUpdate() {
        const now = new Date().toLocaleString();
        const element = document.getElementById('lastUpdate');
        if (element) {
            element.textContent = now;
        }
    }

    showLoading() {
        document.getElementById('loadingOverlay')?.classList.add('active');
    }

    hideLoading() {
        document.getElementById('loadingOverlay')?.classList.remove('active');
    }

    showNotification(message, type = 'info') {
        const container = document.getElementById('notifications');
        if (!container) return;

        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.innerHTML = `
            <div>${this.escapeHtml(message)}</div>
        `;

        container.appendChild(notification);

        // Auto remove after 5 seconds
        setTimeout(() => {
            notification.remove();
        }, 5000);
    }

    // Utility methods
    truncateId(id) {
        return id.length > 8 ? id.substring(0, 8) + '...' : id;
    }

    formatDate(dateString) {
        try {
            return new Date(dateString).toLocaleDateString();
        } catch {
            return 'Invalid date';
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize the app when DOM is loaded
let tracker;
document.addEventListener('DOMContentLoaded', () => {
    tracker = new IssueTracker();
});

// Make tracker available globally for onclick handlers
window.tracker = tracker;