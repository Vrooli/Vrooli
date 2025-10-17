import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child'

if (typeof window !== 'undefined' && window.parent !== window) {
    initIframeBridgeChild({ appId: 'text-tools-ui' })
}

const APIClient = window.APIClient
const DiffModule = window.DiffModule
const SearchModule = window.SearchModule
const TransformModule = window.TransformModule
const ExtractModule = window.ExtractModule
const AnalyzeModule = window.AnalyzeModule
const PipelineModule = window.PipelineModule

// Main Application Controller
class TextToolsApp {
    constructor() {
        this.apiClient = null;
        this.modules = {};
        this.currentTool = 'diff';
        this.initialize();
    }

    async initialize() {
        // Get port from environment or use default
        await this.detectAPIPort();
        
        // Initialize API client
        this.apiClient = new APIClient();
        
        // Check API health
        await this.checkAPIHealth();
        
        // Initialize modules
        this.initializeModules();
        
        // Setup navigation
        this.setupNavigation();
        
        // Load initial tool
        this.switchTool('diff');
    }

    async detectAPIPort() {
        // Try to get port from environment variable
        try {
            const response = await fetch('/api/config');
            const config = await response.json();
            if (!config.api_port) {
                throw new Error('API port not configured in server response');
            }
            window.API_PORT = config.api_port;
        } catch (error) {
            console.error('Failed to detect API port:', error);
            throw new Error('Unable to detect API port. Please ensure the Text Tools API is running.');
        }
    }

    async checkAPIHealth() {
        const statusIndicator = document.getElementById('status-indicator');
        const statusText = statusIndicator?.querySelector('.status-text');
        const statusDot = statusIndicator?.querySelector('.status-dot');

        try {
            const health = await this.apiClient.checkHealth();
            
            if (statusText) {
                statusText.textContent = 'Connected';
            }
            if (statusDot) {
                statusDot.className = 'status-dot connected';
            }
            
            console.log('API Health:', health);
        } catch (error) {
            if (statusText) {
                statusText.textContent = 'Disconnected';
            }
            if (statusDot) {
                statusDot.className = 'status-dot disconnected';
            }
            
            console.error('API health check failed:', error);
            this.showGlobalError('API is not available. Please ensure the Text Tools API is running.');
        }
    }

    initializeModules() {
        // Wait for components to be loaded, then initialize modules
        document.addEventListener('componentsLoaded', () => {
            this.modules.diff = new DiffModule(this.apiClient);
            this.modules.search = new SearchModule(this.apiClient);
            this.modules.transform = new TransformModule(this.apiClient);
            this.modules.extract = new ExtractModule(this.apiClient);
            this.modules.analyze = new AnalyzeModule(this.apiClient);
            this.modules.pipeline = new PipelineModule(this.apiClient);
            
            console.log('All tool modules initialized');
        });
    }

    setupNavigation() {
        // Setup tab navigation
        const navTabs = document.querySelectorAll('.nav-tab');
        navTabs.forEach(tab => {
            tab.addEventListener('click', (e) => {
                const tool = e.target.dataset.tool;
                this.switchTool(tool);
            });
        });

        // Setup keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Alt+1 through Alt+6 for quick tool switching
            if (e.altKey && e.key >= '1' && e.key <= '6') {
                const tools = ['diff', 'search', 'transform', 'extract', 'analyze', 'pipeline'];
                const index = parseInt(e.key) - 1;
                if (tools[index]) {
                    this.switchTool(tools[index]);
                }
            }

            // Ctrl+K for quick search
            if (e.ctrlKey && e.key === 'k') {
                e.preventDefault();
                this.showQuickSearch();
            }
        });

        // Setup API docs button
        const apiDocsBtn = document.getElementById('api-docs-btn');
        if (apiDocsBtn) {
            apiDocsBtn.addEventListener('click', () => this.showAPIDocs());
        }
    }

    switchTool(toolName) {
        // Update navigation
        document.querySelectorAll('.nav-tab').forEach(tab => {
            if (tab.dataset.tool === toolName) {
                tab.classList.add('active');
            } else {
                tab.classList.remove('active');
            }
        });

        // Show/hide panels
        document.querySelectorAll('.tool-panel').forEach(panel => {
            if (panel.id === `${toolName}-panel`) {
                panel.classList.add('active');
            } else {
                panel.classList.remove('active');
            }
        });

        this.currentTool = toolName;

        // Tool-specific initialization
        if (this.modules[toolName] && this.modules[toolName].onActivate) {
            this.modules[toolName].onActivate();
        }
    }

    showQuickSearch() {
        // Create quick search modal
        const modal = document.createElement('div');
        modal.className = 'quick-search-modal';
        modal.innerHTML = `
            <div class="quick-search-content">
                <input type="text" id="quick-search-input" placeholder="Search for a tool or feature..." />
                <div id="quick-search-results"></div>
            </div>
        `;
        document.body.appendChild(modal);

        // Focus input
        const input = document.getElementById('quick-search-input');
        input?.focus();

        // Setup search
        input?.addEventListener('input', (e) => {
            this.performQuickSearch(e.target.value);
        });

        // Close on escape
        document.addEventListener('keydown', function closeModal(e) {
            if (e.key === 'Escape') {
                modal.remove();
                document.removeEventListener('keydown', closeModal);
            }
        });

        // Close on click outside
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                modal.remove();
            }
        });
    }

    performQuickSearch(query) {
        const results = document.getElementById('quick-search-results');
        if (!results) return;

        const tools = [
            { name: 'Diff', tool: 'diff', description: 'Compare two texts' },
            { name: 'Search', tool: 'search', description: 'Search for patterns' },
            { name: 'Transform', tool: 'transform', description: 'Transform text format' },
            { name: 'Extract', tool: 'extract', description: 'Extract text from files' },
            { name: 'Analyze', tool: 'analyze', description: 'Analyze text with NLP' },
            { name: 'Pipeline', tool: 'pipeline', description: 'Chain operations' },
        ];

        const filtered = tools.filter(t => 
            t.name.toLowerCase().includes(query.toLowerCase()) ||
            t.description.toLowerCase().includes(query.toLowerCase())
        );

        results.innerHTML = filtered.map(t => `
            <div class="quick-search-item" data-tool="${t.tool}">
                <strong>${t.name}</strong>
                <span>${t.description}</span>
            </div>
        `).join('');

        // Add click handlers
        results.querySelectorAll('.quick-search-item').forEach(item => {
            item.addEventListener('click', () => {
                const tool = item.dataset.tool;
                this.switchTool(tool);
                document.querySelector('.quick-search-modal')?.remove();
            });
        });
    }

    showAPIDocs() {
        // Open API documentation in modal or new tab
        const docsURL = `http://localhost:${window.API_PORT}/docs`;
        window.open(docsURL, '_blank');
    }

    showGlobalError(message) {
        // Create global error notification
        const notification = document.createElement('div');
        notification.className = 'global-notification error';
        notification.innerHTML = `
            <span class="notification-icon">⚠️</span>
            <span class="notification-message">${message}</span>
            <button class="notification-close">×</button>
        `;
        
        document.body.appendChild(notification);

        // Auto-hide after 10 seconds
        setTimeout(() => notification.remove(), 10000);

        // Manual close
        notification.querySelector('.notification-close')?.addEventListener('click', () => {
            notification.remove();
        });
    }

    // Utility methods
    debounce(func, wait) {
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

    throttle(func, limit) {
        let inThrottle;
        return function(...args) {
            if (!inThrottle) {
                func.apply(this, args);
                inThrottle = true;
                setTimeout(() => inThrottle = false, limit);
            }
        };
    }
}


// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.textToolsApp = new TextToolsApp();
});

// Export for use
window.TextToolsApp = TextToolsApp;
