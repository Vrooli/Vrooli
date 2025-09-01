// App Issue Tracker Dashboard v2 - File-Based Storage
// Updated for file-based YAML storage system

class IssueTracker {
    constructor() {
        // Use same-origin URLs for tunnel compatibility
        this.apiBase = `${window.location.protocol}//${window.location.host}/api`;
        this.mode = 'unknown';
        this.currentPage = 'dashboard';
        this.statusChart = null;
        this.priorityChart = null;
        this.issues = [];
        this.agents = [];
        this.apps = [];
        this.lastUpdate = null;
        
        this.init();
    }

    async init() {
        await this.detectMode();
        this.setupEventListeners();
        this.loadDashboard();
        this.setupCharts();
        this.startAutoRefresh();
        this.showModeIndicator();
    }

    async detectMode() {
        try {
            const response = await fetch(`${this.apiBase}/../health`);
            const data = await response.json();
            
            if (data.storage === 'file-based-yaml') {
                this.mode = 'file-based';
            } else {
                this.mode = 'database';
            }
        } catch (error) {
            this.mode = 'offline';
        }
    }

    showModeIndicator() {
        // Add mode indicator to navbar
        const navbar = document.querySelector('.navbar');
        const existingIndicator = navbar.querySelector('.mode-indicator');
        if (existingIndicator) {
            existingIndicator.remove();
        }
        
        const indicator = document.createElement('div');
        indicator.className = 'mode-indicator';
        
        switch (this.mode) {
            case 'file-based':
                indicator.innerHTML = '<i class="fas fa-folder"></i> File Mode';
                indicator.className += ' mode-file';
                break;
            case 'database':
                indicator.innerHTML = '<i class="fas fa-database"></i> DB Mode';
                indicator.className += ' mode-db';
                break;
            case 'offline':
                indicator.innerHTML = '<i class="fas fa-wifi"></i> Offline';
                indicator.className += ' mode-offline';
                break;
        }
        
        const navActions = navbar.querySelector('.nav-actions');
        navActions.insertBefore(indicator, navActions.firstChild);
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

        // Fix generation button
        document.getElementById('generateFixBtn')?.addEventListener('click', () => {
            this.generateFix();
        });

        // Close modals on background click
        document.querySelectorAll('.modal').forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.closeModal(modal);
                }
            });
        });

        // File mode specific: Direct file operations
        if (this.mode === 'file-based') {
            this.setupFileOperations();
        }
    }

    setupFileOperations() {
        // Add file operation buttons if in file mode
        const toolbar = document.querySelector('.page-header');
        if (toolbar && !toolbar.querySelector('.file-operations')) {
            const fileOps = document.createElement('div');
            fileOps.className = 'file-operations';
            fileOps.innerHTML = `
                <button class="btn btn-secondary" onclick="app.showFileOperations()">
                    <i class="fas fa-folder-open"></i> File Operations
                </button>
            `;
            toolbar.appendChild(fileOps);
        }
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
            
            if (!response.ok) {
                throw new Error(`API Error: ${response.status}`);
            }
            
            const data = await response.json();
            return data;
        } catch (error) {
            console.error('API call failed:', error);
            
            // In file mode, show helpful error message
            if (this.mode === 'file-based') {
                this.showNotification(`File-based API error: ${error.message}. Try starting the file-based API server.`, 'error');
            } else {
                this.showNotification(error.message, 'error');
            }
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
        try {
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
                    // Search page doesn't need preloading
                    break;
            }
        } catch (error) {
            this.showNotification(`Failed to load ${page}: ${error.message}`, 'error');
        }
    }

    async loadDashboard() {
        try {
            // Load stats
            const statsResponse = await this.apiCall('/stats');
            const stats = statsResponse.data.stats;
            
            // Update stat cards
            document.getElementById('totalIssues').textContent = stats.total_issues || 0;
            document.getElementById('openIssues').textContent = stats.open_issues || 0;
            document.getElementById('inProgress').textContent = stats.in_progress || 0;
            document.getElementById('fixedToday').textContent = stats.fixed_today || 0;
            
            // Load recent issues
            const issuesResponse = await this.apiCall('/issues?limit=10');
            this.issues = issuesResponse.data.issues || [];
            this.renderRecentIssues();
            
            // Update charts
            this.updateStatusChart();
            this.updatePriorityChart();
            
            // Update timestamp
            this.lastUpdate = new Date();
            document.getElementById('lastUpdate').textContent = this.formatTime(this.lastUpdate);
            
        } catch (error) {
            console.error('Failed to load dashboard:', error);
            
            // In file mode, show offline stats
            if (this.mode === 'file-based') {
                this.showOfflineStats();
            }
        }
    }

    showOfflineStats() {
        // Show file-based statistics when API is not available
        const offlineNote = document.createElement('div');
        offlineNote.className = 'offline-notice';
        offlineNote.innerHTML = `
            <div class="alert alert-info">
                <i class="fas fa-info-circle"></i>
                <strong>File Mode:</strong> API server not available. 
                <a href="#" onclick="app.openFileManager()">Manage issues directly</a> or 
                <a href="#" onclick="app.startFileBasedAPI()">start the API server</a>.
            </div>
        `;
        
        const mainContent = document.querySelector('.main-content');
        const existingNotice = mainContent.querySelector('.offline-notice');
        if (existingNotice) {
            existingNotice.remove();
        }
        mainContent.insertBefore(offlineNote, mainContent.firstChild);
    }

    async createIssue() {
        const formData = {
            title: document.getElementById('issueTitle').value,
            description: document.getElementById('issueDescription').value || '',
            type: document.getElementById('issueType').value,
            priority: document.getElementById('issuePriority').value,
            error_message: document.getElementById('errorMessage').value || '',
            stack_trace: document.getElementById('stackTrace').value || '',
            tags: document.getElementById('issueTags').value.split(',').map(t => t.trim()).filter(t => t),
            reporter_name: 'Dashboard User',
            reporter_email: 'user@dashboard.local'
        };

        try {
            const response = await this.apiCall('/issues', {
                method: 'POST',
                body: JSON.stringify(formData)
            });

            if (response.success) {
                this.showNotification('Issue created successfully!', 'success');
                this.closeModal(document.getElementById('newIssueModal'));
                
                // Show created file info in file mode
                if (response.data.filename) {
                    this.showNotification(`File created: ${response.data.filename}`, 'info');
                }
                
                // Reset form
                document.getElementById('newIssueForm').reset();
                
                // Refresh current page
                this.refreshCurrentPage();
            }
        } catch (error) {
            this.showNotification('Failed to create issue', 'error');
        }
    }

    async renderRecentIssues() {
        const tbody = document.querySelector('#recentIssuesTable tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (this.issues.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="text-center">No issues found</td></tr>';
            return;
        }

        this.issues.slice(0, 10).forEach(issue => {
            const row = document.createElement('tr');
            row.className = `priority-${issue.priority}`;
            
            const shortId = issue.id.length > 12 ? issue.id.substring(0, 12) + '...' : issue.id;
            const createdAt = new Date(issue.metadata?.created_at || issue.created_at).toLocaleDateString();
            
            row.innerHTML = `
                <td class="issue-id" title="${issue.id}">${shortId}</td>
                <td class="issue-title">
                    <a href="#" onclick="app.showIssueDetails('${issue.id}')">${this.escapeHtml(issue.title)}</a>
                </td>
                <td><span class="status-badge status-${issue.status}">${this.formatStatus(issue.status)}</span></td>
                <td><span class="priority-badge priority-${issue.priority}">${this.formatPriority(issue.priority)}</span></td>
                <td><span class="type-badge type-${issue.type}">${this.formatType(issue.type)}</span></td>
                <td>${createdAt}</td>
                <td class="actions">
                    <button class="btn btn-sm btn-primary" onclick="app.startInvestigation('${issue.id}')">
                        <i class="fas fa-search"></i>
                    </button>
                    ${this.mode === 'file-based' ? `
                    <button class="btn btn-sm btn-secondary" onclick="app.openIssueFile('${issue.id}')" title="Open YAML file">
                        <i class="fas fa-file-alt"></i>
                    </button>
                    ` : ''}
                </td>
            `;
            
            tbody.appendChild(row);
        });
    }

    // File-mode specific functions
    openIssueFile(issueId) {
        // In a real implementation, this could open the file in an editor
        // For now, show the file path
        this.showNotification(`Issue file location: issues/*/[0-9][0-9][0-9]-*.yaml (ID: ${issueId})`, 'info');
    }

    openFileManager() {
        // Show file operations panel
        const panel = document.createElement('div');
        panel.className = 'file-manager-panel';
        panel.innerHTML = `
            <div class="panel-header">
                <h3><i class="fas fa-folder-open"></i> File Operations</h3>
                <button onclick="this.parentElement.parentElement.remove()">&times;</button>
            </div>
            <div class="panel-content">
                <p>Direct file operations for issue management:</p>
                <div class="file-commands">
                    <div class="command-group">
                        <h4>View Issues</h4>
                        <code>ls issues/open/*.yaml</code>
                        <code>cat issues/open/001-critical-bug.yaml</code>
                    </div>
                    <div class="command-group">
                        <h4>Move Issues</h4>
                        <code>mv issues/open/001-bug.yaml issues/investigating/</code>
                        <code>./issues/manage.sh move 001-bug.yaml fixed</code>
                    </div>
                    <div class="command-group">
                        <h4>Create Issues</h4>
                        <code>cp issues/templates/bug-template.yaml issues/open/123-new.yaml</code>
                        <code>./issues/manage.sh add</code>
                    </div>
                    <div class="command-group">
                        <h4>Search</h4>
                        <code>grep -r "authentication" issues/</code>
                        <code>./issues/manage.sh search "timeout"</code>
                    </div>
                </div>
                <div class="file-info">
                    <p><strong>Issues Directory:</strong> <code>./issues/</code></p>
                    <p><strong>Management CLI:</strong> <code>./issues/manage.sh</code></p>
                    <p><strong>Templates:</strong> <code>./issues/templates/</code></p>
                </div>
            </div>
        `;
        
        document.body.appendChild(panel);
    }

    startFileBasedAPI() {
        this.showNotification('Start the file-based API server with: cd api && ./app-issue-tracker-file-api', 'info');
    }

    async performSearch() {
        const query = document.getElementById('searchInput').value.trim();
        const searchType = document.querySelector('input[name="searchType"]:checked').value;

        if (!query) {
            this.showNotification('Please enter a search query', 'warning');
            return;
        }

        try {
            let endpoint;
            if (searchType === 'vector' && this.mode !== 'offline') {
                endpoint = `/search/vector?q=${encodeURIComponent(query)}&limit=20`;
            } else {
                endpoint = `/issues/search?q=${encodeURIComponent(query)}&limit=20`;
            }

            const response = await this.apiCall(endpoint);
            const results = response.data.results || response.data.issues || [];

            this.renderSearchResults(results, query, searchType);

        } catch (error) {
            this.showNotification('Search failed', 'error');
            
            // In file mode, suggest alternative
            if (this.mode === 'file-based') {
                this.showNotification('Try: grep -r "' + query + '" issues/', 'info');
            }
        }
    }

    renderSearchResults(results, query, searchType) {
        const container = document.getElementById('searchResults');
        container.innerHTML = '';

        if (results.length === 0) {
            container.innerHTML = `
                <div class="no-results">
                    <i class="fas fa-search"></i>
                    <h3>No results found</h3>
                    <p>No issues found for "${this.escapeHtml(query)}"</p>
                    ${this.mode === 'file-based' ? `
                    <p class="file-hint">
                        <i class="fas fa-terminal"></i>
                        Try: <code>./issues/manage.sh search "${this.escapeHtml(query)}"</code>
                    </p>
                    ` : ''}
                </div>
            `;
            return;
        }

        const resultsHtml = results.map(issue => {
            const similarity = issue.similarity ? Math.round(issue.similarity * 100) : null;
            return `
                <div class="search-result" onclick="app.showIssueDetails('${issue.id}')">
                    <div class="result-header">
                        <h4>${this.escapeHtml(issue.title)}</h4>
                        <div class="result-meta">
                            <span class="priority-badge priority-${issue.priority}">${this.formatPriority(issue.priority)}</span>
                            <span class="status-badge status-${issue.status}">${this.formatStatus(issue.status)}</span>
                            ${similarity ? `<span class="similarity-score">${similarity}% match</span>` : ''}
                        </div>
                    </div>
                    <p class="result-description">${this.escapeHtml(issue.description || '')}</p>
                    <div class="result-footer">
                        <span class="app-name">${issue.app_id || 'Unknown App'}</span>
                        <span class="created-date">${new Date(issue.metadata?.created_at || issue.created_at).toLocaleDateString()}</span>
                    </div>
                </div>
            `;
        }).join('');

        container.innerHTML = `
            <div class="search-summary">
                <h3>Search Results</h3>
                <p>Found ${results.length} issues for "${this.escapeHtml(query)}" using ${searchType} search</p>
                ${this.mode === 'file-based' ? '<p class="mode-note"><i class="fas fa-folder"></i> Results from file-based storage</p>' : ''}
            </div>
            <div class="results-grid">
                ${resultsHtml}
            </div>
        `;
    }

    async startInvestigation(issueId) {
        if (!issueId && this.selectedIssueId) {
            issueId = this.selectedIssueId;
        }
        
        if (!issueId) {
            this.showNotification('Please select an issue first', 'warning');
            return;
        }

        try {
            const response = await this.apiCall('/investigate', {
                method: 'POST',
                body: JSON.stringify({
                    issue_id: issueId,
                    agent_id: 'deep-investigator',
                    priority: 'normal'
                })
            });

            if (response.success) {
                this.showNotification('Investigation started successfully!', 'success');
                
                if (this.mode === 'file-based') {
                    this.showNotification('Check issues/investigating/ folder for updates', 'info');
                }
                
                // Close modal if open
                this.closeModal(document.getElementById('issueModal'));
                
                // Refresh page
                setTimeout(() => this.refreshCurrentPage(), 2000);
            }
        } catch (error) {
            this.showNotification('Failed to start investigation', 'error');
        }
    }

    updateStatusChart() {
        if (!this.statusChart) return;
        
        const statusCounts = this.getStatusCounts();
        
        this.statusChart.data.datasets[0].data = [
            statusCounts.open || 0,
            statusCounts.investigating || 0,
            statusCounts['in-progress'] || 0,
            statusCounts.fixed || 0,
            statusCounts.closed || 0,
            statusCounts.failed || 0
        ];
        
        this.statusChart.update();
    }

    getStatusCounts() {
        const counts = {};
        this.issues.forEach(issue => {
            counts[issue.status] = (counts[issue.status] || 0) + 1;
        });
        return counts;
    }

    setupCharts() {
        // Status Chart
        const statusCtx = document.getElementById('statusChart');
        if (statusCtx) {
            this.statusChart = new Chart(statusCtx, {
                type: 'doughnut',
                data: {
                    labels: ['Open', 'Investigating', 'In Progress', 'Fixed', 'Closed', 'Failed'],
                    datasets: [{
                        data: [0, 0, 0, 0, 0, 0],
                        backgroundColor: [
                            '#ff6b6b', // Open - red
                            '#4ecdc4', // Investigating - teal
                            '#45b7d1', // In Progress - blue
                            '#96ceb4', // Fixed - green
                            '#feca57', // Closed - yellow
                            '#ff9ff3'  // Failed - pink
                        ]
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: {
                            position: 'bottom'
                        }
                    }
                }
            });
        }

        // Priority Chart
        const priorityCtx = document.getElementById('priorityChart');
        if (priorityCtx) {
            this.priorityChart = new Chart(priorityCtx, {
                type: 'bar',
                data: {
                    labels: ['Critical', 'High', 'Medium', 'Low'],
                    datasets: [{
                        label: 'Issues',
                        data: [0, 0, 0, 0],
                        backgroundColor: ['#ff6b6b', '#ffa726', '#42a5f5', '#66bb6a']
                    }]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true
                        }
                    }
                }
            });
        }
    }

    formatStatus(status) {
        return status.charAt(0).toUpperCase() + status.slice(1).replace('-', ' ');
    }

    formatPriority(priority) {
        return priority.charAt(0).toUpperCase() + priority.slice(1);
    }

    formatType(type) {
        return type.charAt(0).toUpperCase() + type.slice(1);
    }

    formatTime(date) {
        return date.toLocaleTimeString();
    }

    escapeHtml(unsafe) {
        return unsafe
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#039;");
    }

    showLoading() {
        document.getElementById('loadingOverlay').style.display = 'flex';
    }

    hideLoading() {
        document.getElementById('loadingOverlay').style.display = 'none';
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.innerHTML = `
            <i class="fas fa-${this.getNotificationIcon(type)}"></i>
            <span>${message}</span>
            <button onclick="this.parentElement.remove()">&times;</button>
        `;

        document.getElementById('notifications').appendChild(notification);

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (notification.parentElement) {
                notification.remove();
            }
        }, 5000);
    }

    getNotificationIcon(type) {
        switch (type) {
            case 'success': return 'check-circle';
            case 'error': return 'exclamation-triangle';
            case 'warning': return 'exclamation-circle';
            default: return 'info-circle';
        }
    }

    showNewIssueModal() {
        document.getElementById('newIssueModal').style.display = 'flex';
    }

    closeModal(modal) {
        if (modal) {
            modal.style.display = 'none';
        }
    }

    refreshCurrentPage() {
        this.loadPageContent(this.currentPage);
    }

    startAutoRefresh() {
        // Auto refresh every 30 seconds if API is available
        if (this.mode !== 'offline') {
            setInterval(() => {
                this.refreshCurrentPage();
            }, 30000);
        }
    }

    // Additional methods for full functionality would be added here...
    // Including loadAllIssues(), loadAgents(), loadApps(), etc.
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new IssueTracker();
});

// Add CSS for file mode indicators
const style = document.createElement('style');
style.textContent = `
.mode-indicator {
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 12px;
    margin-right: 10px;
}

.mode-file {
    background: #4ecdc4;
    color: white;
}

.mode-db {
    background: #45b7d1; 
    color: white;
}

.mode-offline {
    background: #ff6b6b;
    color: white;
}

.file-manager-panel {
    position: fixed;
    top: 20px;
    right: 20px;
    width: 400px;
    background: white;
    border: 1px solid #ddd;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    z-index: 1000;
}

.panel-header {
    background: #f8f9fa;
    padding: 15px;
    border-bottom: 1px solid #dee2e6;
    border-radius: 8px 8px 0 0;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.panel-content {
    padding: 15px;
    max-height: 400px;
    overflow-y: auto;
}

.command-group {
    margin-bottom: 15px;
}

.command-group h4 {
    margin: 0 0 8px 0;
    color: #495057;
    font-size: 14px;
}

.command-group code {
    display: block;
    background: #f8f9fa;
    padding: 4px 8px;
    border-radius: 4px;
    margin: 4px 0;
    font-family: 'Courier New', monospace;
    font-size: 12px;
    border-left: 3px solid #007bff;
}

.file-info {
    background: #e7f3ff;
    padding: 10px;
    border-radius: 4px;
    margin-top: 15px;
}

.file-info p {
    margin: 4px 0;
    font-size: 12px;
}

.offline-notice {
    margin: 10px;
}

.alert {
    padding: 15px;
    border-radius: 4px;
    border-left: 4px solid;
}

.alert-info {
    background: #e7f3ff;
    border-color: #007bff;
    color: #004085;
}

.file-hint {
    background: #f8f9fa;
    padding: 8px;
    border-radius: 4px;
    margin-top: 10px;
    font-size: 12px;
}
`;
document.head.appendChild(style);