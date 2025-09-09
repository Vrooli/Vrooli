/**
 * Comment System Admin Dashboard
 * JavaScript functionality for managing comment configurations
 */

class CommentSystemDashboard {
    constructor() {
        this.apiUrl = this.getApiUrl();
        this.currentTab = 'dashboard';
        this.scenarios = new Map();
        this.stats = {};
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.checkApiConnection();
        this.loadDashboard();
        
        // Auto-refresh every 30 seconds
        setInterval(() => {
            this.refreshCurrentView();
        }, 30000);
    }

    getApiUrl() {
        // Try to get API URL from environment or use default
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get('api') || `http://localhost:${window.API_PORT || 8080}`;
    }

    setupEventListeners() {
        // Tab navigation
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tab = e.target.dataset.tab;
                this.switchTab(tab);
            });
        });

        // Refresh button
        document.getElementById('refresh-btn')?.addEventListener('click', () => {
            this.refreshCurrentView();
        });

        // Add scenario button
        document.getElementById('add-scenario-btn')?.addEventListener('click', () => {
            this.showScenarioModal();
        });

        // Modal handling
        document.getElementById('modal-close')?.addEventListener('click', () => {
            this.hideModal();
        });

        document.getElementById('cancel-btn')?.addEventListener('click', () => {
            this.hideModal();
        });

        // Form submission
        document.getElementById('scenario-form')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveScenarioConfig();
        });

        // Modal backdrop click
        document.getElementById('scenario-modal')?.addEventListener('click', (e) => {
            if (e.target.id === 'scenario-modal') {
                this.hideModal();
            }
        });
    }

    async checkApiConnection() {
        const statusEl = document.getElementById('connection-status');
        const statusDot = statusEl?.querySelector('.status-dot');
        const statusText = statusEl?.querySelector('.status-text');

        try {
            const response = await fetch(`${this.apiUrl}/health`);
            const data = await response.json();

            if (response.ok && data.status === 'healthy') {
                statusDot?.classList.remove('disconnected');
                statusDot?.classList.add('connected');
                statusText.textContent = 'Connected';
            } else {
                throw new Error('API unhealthy');
            }
        } catch (error) {
            statusDot?.classList.remove('connected');
            statusDot?.classList.add('disconnected');
            statusText.textContent = 'Disconnected';
            console.error('API connection failed:', error);
        }
    }

    switchTab(tabName) {
        // Update navigation
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.classList.remove('active');
        });
        document.querySelector(`[data-tab="${tabName}"]`)?.classList.add('active');

        // Update content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(`${tabName}-tab`)?.classList.add('active');

        this.currentTab = tabName;

        // Load tab-specific data
        switch (tabName) {
            case 'dashboard':
                this.loadDashboard();
                break;
            case 'scenarios':
                this.loadScenarios();
                break;
            case 'analytics':
                this.loadAnalytics();
                break;
        }
    }

    async loadDashboard() {
        try {
            await Promise.all([
                this.loadStats(),
                this.loadRecentActivity()
            ]);
        } catch (error) {
            console.error('Failed to load dashboard:', error);
        }
    }

    async loadStats() {
        try {
            // For now, use mock data until admin endpoints are implemented
            const stats = {
                totalScenarios: 12,
                totalComments: 1547,
                activeUsers: 89,
                commentsToday: 23
            };

            document.getElementById('total-scenarios').textContent = stats.totalScenarios;
            document.getElementById('total-comments').textContent = stats.totalComments.toLocaleString();
            document.getElementById('active-users').textContent = stats.activeUsers;
            document.getElementById('comments-today').textContent = stats.commentsToday;

            this.stats = stats;
        } catch (error) {
            console.error('Failed to load stats:', error);
            // Show loading placeholders
            document.querySelectorAll('.stat-number').forEach(el => {
                el.textContent = '-';
            });
        }
    }

    async loadRecentActivity() {
        const activityList = document.getElementById('recent-activity-list');
        
        try {
            // Mock activity data for now
            const activities = [
                {
                    icon: '💬',
                    text: 'New comment in picker-wheel scenario',
                    time: '2 minutes ago',
                    user: 'john_doe'
                },
                {
                    icon: '⚙️',
                    text: 'Updated configuration for study-buddy',
                    time: '15 minutes ago',
                    user: 'admin'
                },
                {
                    icon: '🚫',
                    text: 'Comment moderated in retro-game-launcher',
                    time: '1 hour ago',
                    user: 'moderator'
                }
            ];

            activityList.innerHTML = activities.map(activity => `
                <div class="activity-item">
                    <div class="activity-icon">${activity.icon}</div>
                    <div class="activity-content">
                        <p><strong>${activity.text}</strong></p>
                        <small class="text-muted">${activity.time} by ${activity.user}</small>
                    </div>
                </div>
            `).join('');

        } catch (error) {
            console.error('Failed to load recent activity:', error);
            activityList.innerHTML = `
                <div class="activity-item">
                    <div class="activity-icon">❌</div>
                    <div class="activity-content">
                        <p>Failed to load recent activity</p>
                    </div>
                </div>
            `;
        }
    }

    async loadScenarios() {
        const scenariosGrid = document.getElementById('scenarios-grid');
        
        try {
            // For demonstration, we'll create some example scenarios
            // In a real implementation, this would fetch from the API
            const scenarios = [
                {
                    name: 'picker-wheel',
                    authRequired: true,
                    allowAnonymous: false,
                    allowRichMedia: true,
                    moderationLevel: 'manual',
                    commentCount: 45,
                    lastActivity: '2 hours ago'
                },
                {
                    name: 'study-buddy',
                    authRequired: false,
                    allowAnonymous: true,
                    allowRichMedia: false,
                    moderationLevel: 'none',
                    commentCount: 128,
                    lastActivity: '15 minutes ago'
                },
                {
                    name: 'retro-game-launcher',
                    authRequired: true,
                    allowAnonymous: false,
                    allowRichMedia: true,
                    moderationLevel: 'ai_assisted',
                    commentCount: 67,
                    lastActivity: '1 hour ago'
                }
            ];

            scenariosGrid.innerHTML = scenarios.map(scenario => `
                <div class="scenario-card">
                    <h3>${scenario.name}</h3>
                    <p><strong>${scenario.commentCount}</strong> comments • Last activity: ${scenario.lastActivity}</p>
                    
                    <div class="scenario-meta">
                        ${scenario.authRequired ? '<span class="meta-badge auth-required">Auth Required</span>' : ''}
                        ${scenario.allowAnonymous ? '<span class="meta-badge anonymous">Anonymous OK</span>' : ''}
                        ${scenario.allowRichMedia ? '<span class="meta-badge rich-media">Rich Media</span>' : ''}
                    </div>
                    
                    <div class="scenario-actions">
                        <button class="btn btn-primary" onclick="dashboard.editScenario('${scenario.name}')">
                            Configure
                        </button>
                        <button class="btn btn-secondary" onclick="dashboard.viewComments('${scenario.name}')">
                            View Comments
                        </button>
                    </div>
                </div>
            `).join('');

            // Cache scenarios
            scenarios.forEach(scenario => {
                this.scenarios.set(scenario.name, scenario);
            });

        } catch (error) {
            console.error('Failed to load scenarios:', error);
            scenariosGrid.innerHTML = `
                <div class="scenario-card">
                    <h3>Error loading scenarios</h3>
                    <p>Please check your connection and try again.</p>
                </div>
            `;
        }
    }

    async loadAnalytics() {
        // Analytics functionality would be implemented here
        // For now, we show a placeholder
        console.log('Analytics view loaded');
    }

    showScenarioModal(scenarioName = null) {
        const modal = document.getElementById('scenario-modal');
        const form = document.getElementById('scenario-form');
        const title = document.getElementById('modal-title');

        if (scenarioName) {
            // Edit mode
            const scenario = this.scenarios.get(scenarioName);
            if (scenario) {
                title.textContent = `Configure ${scenarioName}`;
                document.getElementById('scenario-name').value = scenarioName;
                document.getElementById('auth-required').checked = scenario.authRequired;
                document.getElementById('allow-anonymous').checked = scenario.allowAnonymous;
                document.getElementById('allow-rich-media').checked = scenario.allowRichMedia;
                document.getElementById('moderation-level').value = scenario.moderationLevel;
                
                // Disable scenario name field for existing scenarios
                document.getElementById('scenario-name').disabled = true;
            }
        } else {
            // Create mode
            title.textContent = 'Add New Scenario';
            form.reset();
            document.getElementById('scenario-name').disabled = false;
        }

        modal.classList.add('active');
    }

    hideModal() {
        const modal = document.getElementById('scenario-modal');
        modal.classList.remove('active');
    }

    async saveScenarioConfig() {
        const form = document.getElementById('scenario-form');
        const formData = new FormData(form);
        
        const config = {
            scenario_name: formData.get('scenario_name'),
            auth_required: formData.has('auth_required'),
            allow_anonymous: formData.has('allow_anonymous'),
            allow_rich_media: formData.has('allow_rich_media'),
            moderation_level: formData.get('moderation_level')
        };

        try {
            const response = await fetch(`${this.apiUrl}/api/v1/config/${config.scenario_name}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(config)
            });

            if (response.ok) {
                // Success - reload scenarios and hide modal
                this.hideModal();
                this.loadScenarios();
                this.showNotification('Configuration saved successfully', 'success');
            } else {
                throw new Error('Failed to save configuration');
            }
        } catch (error) {
            console.error('Failed to save scenario config:', error);
            this.showNotification('Failed to save configuration', 'error');
        }
    }

    editScenario(scenarioName) {
        this.showScenarioModal(scenarioName);
    }

    viewComments(scenarioName) {
        // Open comments view in new tab/window
        window.open(`${this.apiUrl}/api/v1/comments/${scenarioName}`, '_blank');
    }

    refreshCurrentView() {
        switch (this.currentTab) {
            case 'dashboard':
                this.loadDashboard();
                break;
            case 'scenarios':
                this.loadScenarios();
                break;
            case 'analytics':
                this.loadAnalytics();
                break;
        }

        // Always refresh connection status
        this.checkApiConnection();
    }

    showNotification(message, type = 'info') {
        // Simple notification system
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 1rem 1.5rem;
            background: ${type === 'success' ? 'var(--success-color)' : 'var(--danger-color)'};
            color: white;
            border-radius: var(--border-radius);
            box-shadow: var(--box-shadow-hover);
            z-index: 1100;
            animation: slideInRight 0.3s ease;
        `;

        document.body.appendChild(notification);

        // Auto remove after 3 seconds
        setTimeout(() => {
            notification.style.animation = 'slideOutRight 0.3s ease';
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.parentNode.removeChild(notification);
                }
            }, 300);
        }, 3000);
    }

    // Utility methods
    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleString();
    }

    formatNumber(number) {
        return number.toLocaleString();
    }
}

// CSS for notifications
const notificationCSS = `
@keyframes slideInRight {
    from {
        transform: translateX(100%);
        opacity: 0;
    }
    to {
        transform: translateX(0);
        opacity: 1;
    }
}

@keyframes slideOutRight {
    from {
        transform: translateX(0);
        opacity: 1;
    }
    to {
        transform: translateX(100%);
        opacity: 0;
    }
}
`;

// Add notification CSS to document
const style = document.createElement('style');
style.textContent = notificationCSS;
document.head.appendChild(style);

// Initialize dashboard when DOM is loaded
let dashboard;
document.addEventListener('DOMContentLoaded', () => {
    dashboard = new CommentSystemDashboard();
    
    // Make dashboard globally available for onclick handlers
    window.dashboard = dashboard;
});

// Handle escape key to close modal
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        const modal = document.getElementById('scenario-modal');
        if (modal && modal.classList.contains('active')) {
            dashboard.hideModal();
        }
    }
});

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = CommentSystemDashboard;
}