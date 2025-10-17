import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

/**
 * Bookmark Intelligence Hub - Frontend Application
 * Professional dashboard for managing intelligent bookmark processing
 */

const BRIDGE_FLAG = '__bookmarkIntelligenceBridgeInitialized';

function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window[BRIDGE_FLAG]) {
        return;
    }

    if (window.parent !== window) {
        let parentOrigin;
        try {
            if (document.referrer) {
                parentOrigin = new URL(document.referrer).origin;
            }
        } catch (error) {
            console.warn('[BookmarkIntelligenceHub] Unable to parse parent origin for iframe bridge', error);
        }

        initIframeBridgeChild({ parentOrigin, appId: 'bookmark-intelligence-hub' });
        window[BRIDGE_FLAG] = true;
    }
}

bootstrapIframeBridge();

class BookmarkIntelligenceHub {
    constructor() {
        this.apiBaseUrl = '/api/v1';
        this.currentProfile = null;
        this.connectionStatus = 'disconnected';
        this.lastSyncTime = null;
        
        this.init();
    }

    /**
     * Initialize the application
     */
    init() {
        this.bindEvents();
        this.checkApiConnection();
        this.loadProfiles();
        this.startPeriodicUpdates();
        
        console.log('🔖 Bookmark Intelligence Hub initialized');
    }

    /**
     * Bind event listeners
     */
    bindEvents() {
        // Header actions
        document.getElementById('sync-btn')?.addEventListener('click', () => this.syncBookmarks());
        document.getElementById('profile-select')?.addEventListener('change', (e) => this.switchProfile(e.target.value));
        
        // Footer actions
        document.getElementById('settings-btn')?.addEventListener('click', () => this.openSettings());
        document.getElementById('help-btn')?.addEventListener('click', () => this.openHelp());
        
        // Modal close
        document.querySelectorAll('.modal-close').forEach(btn => {
            btn.addEventListener('click', (e) => this.closeModal(e.target.closest('.modal')));
        });
        
        // Click outside modal to close
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.closeModal(e.target);
            }
        });

        // Example category clicks
        document.addEventListener('click', (e) => {
            if (e.target.closest('.category-item')) {
                const categoryName = e.target.closest('.category-item').querySelector('.category-name').textContent;
                this.viewCategory(categoryName);
            }
        });
    }

    /**
     * Check API connection status
     */
    async checkApiConnection() {
        try {
            const response = await this.apiCall('/health');
            if (response.status === 'healthy') {
                this.updateConnectionStatus('connected');
            } else {
                this.updateConnectionStatus('error');
            }
        } catch (error) {
            console.error('API connection failed:', error);
            this.updateConnectionStatus('disconnected');
            this.showDemoData(); // Show demo data when API is unavailable
        }
    }

    /**
     * Update connection status UI
     */
    updateConnectionStatus(status) {
        this.connectionStatus = status;
        const statusElement = document.getElementById('connection-status');
        const statusIcon = statusElement?.querySelector('i');
        
        if (!statusElement || !statusIcon) return;
        
        statusIcon.className = 'fas fa-circle';
        
        switch (status) {
            case 'connected':
                statusIcon.classList.add('status-connected');
                statusElement.innerHTML = '<i class="fas fa-circle status-connected"></i> API: Connected';
                break;
            case 'disconnected':
                statusIcon.classList.add('status-disconnected');
                statusElement.innerHTML = '<i class="fas fa-circle status-disconnected"></i> API: Disconnected';
                break;
            case 'error':
                statusIcon.classList.add('status-disconnected');
                statusElement.innerHTML = '<i class="fas fa-circle status-disconnected"></i> API: Error';
                break;
        }
    }

    /**
     * Load user profiles
     */
    async loadProfiles() {
        try {
            const profiles = await this.apiCall('/profiles');
            this.populateProfileSelector(profiles);
        } catch (error) {
            console.error('Failed to load profiles:', error);
            this.populateProfileSelector([
                { id: 'demo-profile', name: 'Demo Profile' }
            ]);
        }
    }

    /**
     * Populate profile selector dropdown
     */
    populateProfileSelector(profiles) {
        const select = document.getElementById('profile-select');
        if (!select) return;
        
        select.innerHTML = '<option value="">Select Profile...</option>';
        
        profiles.forEach(profile => {
            const option = document.createElement('option');
            option.value = profile.id;
            option.textContent = profile.name;
            select.appendChild(option);
        });
        
        // Auto-select first profile if available
        if (profiles.length > 0) {
            select.value = profiles[0].id;
            this.switchProfile(profiles[0].id);
        }
    }

    /**
     * Switch to different profile
     */
    async switchProfile(profileId) {
        if (!profileId) return;
        
        this.currentProfile = profileId;
        this.showLoading();
        
        try {
            await this.loadDashboardData();
        } catch (error) {
            console.error('Failed to switch profile:', error);
            this.showDemoData();
        } finally {
            this.hideLoading();
        }
    }

    /**
     * Load dashboard data for current profile
     */
    async loadDashboardData() {
        if (!this.currentProfile) return;
        
        try {
            const [stats, bookmarks, categories, actions, platformStatus] = await Promise.all([
                this.apiCall(`/profiles/${this.currentProfile}/stats`),
                this.apiCall(`/bookmarks/query?profile_id=${this.currentProfile}&limit=10`),
                this.apiCall(`/categories?profile_id=${this.currentProfile}`),
                this.apiCall(`/actions?profile_id=${this.currentProfile}&status=pending`),
                this.apiCall(`/platforms/status?profile_id=${this.currentProfile}`)
            ]);
            
            this.updateStats(stats);
            this.updateRecentBookmarks(bookmarks);
            this.updateCategories(categories);
            this.updatePendingActions(actions);
            this.updatePlatformStatus(platformStatus);
            
        } catch (error) {
            console.error('Failed to load dashboard data:', error);
            this.showDemoData();
        }
    }

    /**
     * Update stats display
     */
    updateStats(stats) {
        document.getElementById('total-bookmarks').textContent = stats.totalBookmarks || 0;
        document.getElementById('categories-count').textContent = stats.categoriesCount || 0;
        document.getElementById('pending-actions').textContent = stats.pendingActions || 0;
        document.getElementById('accuracy-rate').textContent = `${stats.accuracyRate || 0}%`;
    }

    /**
     * Update recent bookmarks list
     */
    updateRecentBookmarks(bookmarks) {
        const container = document.getElementById('recent-bookmarks');
        if (!container) return;
        
        if (!bookmarks || bookmarks.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-bookmark-o"></i>
                    <p>No recent bookmarks. Connect your social media accounts to get started!</p>
                    <button class="btn btn-primary">Add Platform</button>
                </div>
            `;
            return;
        }
        
        container.innerHTML = bookmarks.map(bookmark => `
            <div class="bookmark-item">
                <div class="bookmark-icon ${bookmark.platform}">
                    <i class="fab fa-${bookmark.platform}"></i>
                </div>
                <div class="bookmark-content">
                    <div class="bookmark-title">${bookmark.title || 'Untitled'}</div>
                    <div class="bookmark-meta">${bookmark.platform} • ${this.formatDate(bookmark.created_at)}</div>
                </div>
                <div class="bookmark-category">${bookmark.category || 'Uncategorized'}</div>
            </div>
        `).join('');
    }

    /**
     * Update categories grid
     */
    updateCategories(categories) {
        const container = document.getElementById('categories-grid');
        if (!container) return;
        
        if (!categories || categories.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-folder-open"></i>
                    <p>No categories yet. Bookmarks will be automatically categorized as they're processed.</p>
                </div>
            `;
            return;
        }
        
        container.innerHTML = categories.map(category => `
            <div class="category-item" data-category="${category.name}">
                <div class="category-count">${category.count}</div>
                <div class="category-name">${category.name}</div>
            </div>
        `).join('');
    }

    /**
     * Update pending actions list
     */
    updatePendingActions(actions) {
        const container = document.getElementById('pending-actions-list');
        if (!container) return;
        
        if (!actions || actions.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-check-circle"></i>
                    <p>No pending actions. Great job staying on top of your bookmarks!</p>
                </div>
            `;
            return;
        }
        
        container.innerHTML = actions.map(action => `
            <div class="action-item" data-action-id="${action.id}">
                <div class="action-content">
                    <div class="action-title">${action.title}</div>
                    <div class="action-description">${action.description}</div>
                </div>
                <div class="action-buttons">
                    <button class="btn btn-success btn-sm" onclick="app.approveAction('${action.id}')">
                        <i class="fas fa-check"></i> Approve
                    </button>
                    <button class="btn btn-danger btn-sm" onclick="app.rejectAction('${action.id}')">
                        <i class="fas fa-times"></i> Reject
                    </button>
                </div>
            </div>
        `).join('');
    }

    /**
     * Update platform status
     */
    updatePlatformStatus(status) {
        const container = document.getElementById('platform-status');
        if (!container) return;
        
        const platforms = status || [
            { name: 'Reddit', id: 'reddit', connected: false },
            { name: 'X (Twitter)', id: 'twitter', connected: false },
            { name: 'TikTok', id: 'tiktok', connected: false }
        ];
        
        container.innerHTML = platforms.map(platform => {
            const statusClass = platform.connected ? 'status-connected' : 'status-disconnected';
            const statusText = platform.connected ? 'Connected' : 'Disconnected';
            const iconClass = platform.id === 'twitter' ? 'fa-twitter' : `fa-${platform.id}`;
            
            return `
                <div class="platform-item">
                    <div class="platform-info">
                        <i class="fab ${iconClass}"></i>
                        <span>${platform.name}</span>
                    </div>
                    <div class="status-badge ${statusClass}">
                        <i class="fas fa-circle"></i> ${statusText}
                    </div>
                </div>
            `;
        }).join('');
    }

    /**
     * Show demo data when API is unavailable
     */
    showDemoData() {
        this.updateStats({
            totalBookmarks: 1247,
            categoriesCount: 8,
            pendingActions: 3,
            accuracyRate: 92
        });
        
        this.updateRecentBookmarks([
            {
                id: '1',
                title: 'How to Build Better APIs',
                platform: 'reddit',
                category: 'Programming',
                created_at: new Date().toISOString()
            },
            {
                id: '2',
                title: 'Amazing Cookie Recipe Thread',
                platform: 'twitter',
                category: 'Recipes',
                created_at: new Date(Date.now() - 1000 * 60 * 30).toISOString()
            }
        ]);
        
        this.updateCategories([
            { name: 'Programming', count: 342 },
            { name: 'Recipes', count: 156 },
            { name: 'Fitness', count: 89 },
            { name: 'Travel', count: 67 }
        ]);
        
        this.updatePendingActions([
            {
                id: '1',
                title: 'Add Recipe to Recipe Book',
                description: 'Chocolate chip cookies from @BakeWithJoy'
            }
        ]);
        
        this.updatePlatformStatus();
    }

    /**
     * Sync bookmarks
     */
    async syncBookmarks() {
        if (!this.currentProfile) {
            this.showNotification('Please select a profile first', 'warning');
            return;
        }
        
        this.showLoading();
        
        try {
            const result = await this.apiCall('/bookmarks/sync', 'POST', {
                profile_id: this.currentProfile
            });
            
            this.showNotification(`Synced ${result.processed_count} bookmarks`, 'success');
            this.loadDashboardData(); // Refresh data
            this.lastSyncTime = new Date();
            this.updateLastSyncTime();
            
        } catch (error) {
            console.error('Sync failed:', error);
            this.showNotification('Sync failed. Please try again.', 'error');
        } finally {
            this.hideLoading();
        }
    }

    /**
     * Approve action
     */
    async approveAction(actionId) {
        try {
            await this.apiCall('/actions/approve', 'POST', {
                profile_id: this.currentProfile,
                action_ids: [actionId]
            });
            
            this.showNotification('Action approved successfully', 'success');
            this.loadDashboardData(); // Refresh data
            
        } catch (error) {
            console.error('Failed to approve action:', error);
            this.showNotification('Failed to approve action', 'error');
        }
    }

    /**
     * Reject action
     */
    async rejectAction(actionId) {
        try {
            await this.apiCall('/actions/reject', 'POST', {
                profile_id: this.currentProfile,
                action_ids: [actionId]
            });
            
            this.showNotification('Action rejected', 'info');
            this.loadDashboardData(); // Refresh data
            
        } catch (error) {
            console.error('Failed to reject action:', error);
            this.showNotification('Failed to reject action', 'error');
        }
    }

    /**
     * View category details
     */
    viewCategory(categoryName) {
        console.log(`Viewing category: ${categoryName}`);
        // TODO: Implement category detail view
        this.showNotification(`Category view for "${categoryName}" coming soon!`, 'info');
    }

    /**
     * Open settings modal
     */
    openSettings() {
        console.log('Opening settings...');
        // TODO: Implement settings modal
        this.showNotification('Settings panel coming soon!', 'info');
    }

    /**
     * Open help modal
     */
    openHelp() {
        console.log('Opening help...');
        // TODO: Implement help modal
        this.showNotification('Help documentation coming soon!', 'info');
    }

    /**
     * Show/hide modal
     */
    showModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('hidden');
        }
    }

    closeModal(modal) {
        if (modal) {
            modal.classList.add('hidden');
        }
    }

    /**
     * Show/hide loading overlay
     */
    showLoading() {
        const overlay = document.getElementById('loading-overlay');
        if (overlay) {
            overlay.classList.remove('hidden');
        }
    }

    hideLoading() {
        const overlay = document.getElementById('loading-overlay');
        if (overlay) {
            overlay.classList.add('hidden');
        }
    }

    /**
     * Show notification (simple implementation)
     */
    showNotification(message, type = 'info') {
        console.log(`[${type.toUpperCase()}] ${message}`);
        
        // Create simple toast notification
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.textContent = message;
        toast.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            background: ${type === 'success' ? '#4caf50' : type === 'error' ? '#f44336' : type === 'warning' ? '#ff9800' : '#2196f3'};
            color: white;
            padding: 1rem 1.5rem;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            z-index: 3000;
            animation: slideIn 0.3s ease;
        `;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => {
                document.body.removeChild(toast);
            }, 300);
        }, 3000);
    }

    /**
     * Update last sync time display
     */
    updateLastSyncTime() {
        const element = document.getElementById('last-sync');
        if (element && this.lastSyncTime) {
            element.textContent = `Last Sync: ${this.formatDate(this.lastSyncTime)}`;
        }
    }

    /**
     * Start periodic updates
     */
    startPeriodicUpdates() {
        // Check connection every 30 seconds
        setInterval(() => {
            this.checkApiConnection();
        }, 30000);
        
        // Update dashboard data every 5 minutes
        setInterval(() => {
            if (this.currentProfile && this.connectionStatus === 'connected') {
                this.loadDashboardData();
            }
        }, 5 * 60 * 1000);
    }

    /**
     * Make API call
     */
    async apiCall(endpoint, method = 'GET', data = null) {
        const response = await fetch(`${this.apiBaseUrl}${endpoint}`, {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
            body: data ? JSON.stringify(data) : null
        });
        
        if (!response.ok) {
            throw new Error(`API call failed: ${response.status} ${response.statusText}`);
        }
        
        return await response.json();
    }

    /**
     * Format date for display
     */
    formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / (1000 * 60));
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);
        
        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        if (diffDays < 7) return `${diffDays}d ago`;
        
        return date.toLocaleDateString();
    }
}

// Add toast animation styles
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
`;
document.head.appendChild(style);

// Initialize application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new BookmarkIntelligenceHub();
});
