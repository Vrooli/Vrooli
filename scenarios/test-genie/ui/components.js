// Shared UI Components for Test Genie
// This file contains reusable components and utilities

const MODEL_STORAGE_KEY = 'testGenie.selectedModel';
const DEFAULT_MODEL = 'openrouter/x-ai/grok-code-fast-1';

function getSelectedModel() {
    return localStorage.getItem(MODEL_STORAGE_KEY) || DEFAULT_MODEL;
}

// Navigation Component
class Navigation {
    constructor(currentPage) {
        this.currentPage = currentPage;
    }

    render() {
        return `
            <nav class="sidebar">
                <div class="nav-section">
                    <div class="nav-section-title">Navigation</div>
                    <div class="nav-item ${this.currentPage === 'dashboard' ? 'active' : ''}" data-page="dashboard">
                        <i class="nav-icon" data-lucide="bar-chart-3"></i> Dashboard
                    </div>
                    <div class="nav-item ${this.currentPage === 'generate' ? 'active' : ''}" data-page="generate">
                        <i class="nav-icon" data-lucide="zap"></i> Generate Tests
                    </div>
                    <div class="nav-item ${this.currentPage === 'suites' ? 'active' : ''}" data-page="suites">
                        <i class="nav-icon" data-lucide="flask-conical"></i> Test Suites
                    </div>
                    <div class="nav-item ${this.currentPage === 'executions' ? 'active' : ''}" data-page="executions">
                        <i class="nav-icon" data-lucide="play"></i> Executions
                    </div>
                </div>
                
                <div class="nav-section">
                    <div class="nav-section-title">Analysis</div>
                    <div class="nav-item ${this.currentPage === 'coverage' ? 'active' : ''}" data-page="coverage">
                        <i class="nav-icon" data-lucide="trending-up"></i> Coverage
                    </div>
                    <div class="nav-item ${this.currentPage === 'vault' ? 'active' : ''}" data-page="vault">
                        <i class="nav-icon" data-lucide="building-2"></i> Test Vault
                    </div>
                    <div class="nav-item ${this.currentPage === 'reports' ? 'active' : ''}" data-page="reports">
                        <i class="nav-icon" data-lucide="clipboard"></i> Reports
                    </div>
                </div>
                
                <div class="nav-section">
                    <div class="nav-section-title">System</div>
                    <div class="nav-item ${this.currentPage === 'settings' ? 'active' : ''}" data-page="settings">
                        <i class="nav-icon" data-lucide="settings"></i> Settings
                    </div>
                    <div class="nav-item ${this.currentPage === 'help' ? 'active' : ''}" data-page="help">
                        <i class="nav-icon" data-lucide="help-circle"></i> Help
                    </div>
                </div>
            </nav>
        `;
    }

    attachEventListeners() {
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', () => {
                const page = item.dataset.page;
                if (page === this.currentPage) return;
                
                // Navigate to the appropriate page
                switch(page) {
                    case 'dashboard':
                        window.location.href = '/';
                        break;
                    case 'generate':
                        window.location.href = '/generate.html';
                        break;
                    case 'vault':
                        window.location.href = '/vault.html';
                        break;
                    case 'coverage':
                        window.location.href = '/coverage.html';
                        break;
                    case 'executions':
                        window.location.href = '/executions.html';
                        break;
                    case 'settings':
                        window.location.href = '/settings.html';
                        break;
                    case 'help':
                        window.location.href = '/help.html';
                        break;
                    default:
                        window.location.href = `/${page}.html`;
                }
            });
        });
    }
}

// Header Component
class Header {
    constructor() {
        this.statusInterval = null;
    }

    render() {
        return `
            <header class="header">
                <div class="header-left">
                    <div class="logo">
                        <i data-lucide="flask-conical"></i>
                        <span>Test Genie</span>
                    </div>
                    <div class="system-status">
                        <span class="status-indicator status-checking"></span>
                        <span class="status-text">Checking...</span>
                    </div>
                </div>
                <div class="header-actions">
                    <button class="header-button" id="refresh-btn">
                        <i data-lucide="refresh-cw"></i> Refresh
                    </button>
                    <button class="header-button" id="notifications-btn">
                        <i data-lucide="bell"></i>
                        <span class="notification-badge" style="display: none;">0</span>
                    </button>
                </div>
            </header>
        `;
    }

    startStatusCheck() {
        const checkStatus = async () => {
            try {
                const response = await fetch('/api/health');
                const data = await response.json();
                this.updateStatus(data.healthy ? 'healthy' : 'degraded', data.message || 'System operational');
            } catch (error) {
                this.updateStatus('error', 'Connection failed');
            }
        };

        checkStatus();
        this.statusInterval = setInterval(checkStatus, 30000);
    }

    updateStatus(status, message) {
        const indicator = document.querySelector('.status-indicator');
        const text = document.querySelector('.status-text');
        
        indicator.className = `status-indicator status-${status}`;
        text.textContent = message;
    }

    destroy() {
        if (this.statusInterval) {
            clearInterval(this.statusInterval);
        }
    }
}

// Toast Notification System
class Toast {
    static show(message, type = 'info', duration = 3000) {
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.innerHTML = `
            <span class="toast-icon">${this.getIcon(type)}</span>
            <span class="toast-message">${message}</span>
        `;
        
        document.body.appendChild(toast);
        
        // Trigger animation
        setTimeout(() => toast.classList.add('toast-show'), 10);
        
        // Remove after duration
        setTimeout(() => {
            toast.classList.remove('toast-show');
            setTimeout(() => toast.remove(), 300);
        }, duration);
    }

    static getIcon(type) {
        const icons = {
            success: '<i data-lucide="check-circle"></i>',
            error: '<i data-lucide="x-circle"></i>',
            warning: '<i data-lucide="triangle-alert"></i>',
            info: '<i data-lucide="info"></i>'
        };
        return icons[type] || icons.info;
    }
}

// Loading Spinner Component
class LoadingSpinner {
    static show(container, message = 'Loading...') {
        const spinner = document.createElement('div');
        spinner.className = 'loading-container';
        spinner.innerHTML = `
            <div class="loading-spinner"></div>
            <div class="loading-message">${message}</div>
        `;
        container.appendChild(spinner);
        return spinner;
    }

    static remove(spinner) {
        if (spinner && spinner.parentNode) {
            spinner.remove();
        }
    }
}

// API Client
class TestGenieAPI {
    constructor() {
        this.baseUrl = '/api';
    }

    async request(method, endpoint, data = null) {
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            }
        };

        if (data) {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(`${this.baseUrl}${endpoint}`, options);
            const result = await response.json();
            
            if (!response.ok) {
                throw new Error(result.error || 'Request failed');
            }
            
            return result;
        } catch (error) {
            console.error(`API Error: ${endpoint}`, error);
            throw error;
        }
    }

    // Test Suite Methods
    async generateTestSuite(config) {
        const payload = { ...(config || {}) };
        payload.model = getSelectedModel();
        return this.request('POST', '/test-suite/generate', payload);
    }

    async getTestSuite(suiteId) {
        return this.request('GET', `/test-suite/${suiteId}`);
    }

    async listTestSuites() {
        return this.request('GET', '/test-suites');
    }

    async executeTestSuite(suiteId, config) {
        return this.request('POST', `/test-suite/${suiteId}/execute`, config);
    }

    // Coverage Methods
    async analyzeCoverage(config) {
        return this.request('POST', '/test-analysis/coverage', config);
    }

    async getCoverageAnalysis(scenarioName) {
        return this.request('GET', `/test-analysis/coverage/${scenarioName}`);
    }

    // Vault Methods
    async createTestVault(config) {
        return this.request('POST', '/test-vault/create', config);
    }

    async getTestVault(vaultId) {
        return this.request('GET', `/test-vault/${vaultId}`);
    }

    async executeTestVault(vaultId, config) {
        return this.request('POST', `/test-vault/${vaultId}/execute`, config);
    }

    // System Methods
    async getSystemStatus() {
        return this.request('GET', '/system/status');
    }

    async getSystemMetrics() {
        return this.request('GET', '/system/metrics');
    }
}

// Shared Styles (to be included in each page)
const sharedStyles = `
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        :root {
            /* Color Scheme */
            --bg-primary: #0a0a0a;
            --bg-secondary: #1a1a1a;
            --bg-panel: #2a2a2a;
            --bg-card: #333333;
            --accent-primary: #00ff41;
            --accent-secondary: #00d4ff;
            --accent-warning: #ff6b35;
            --accent-error: #ff0040;
            --accent-success: #39ff14;
            --text-primary: #e0e0e0;
            --text-secondary: #a0a0a0;
            --text-muted: #606060;
            --border: rgba(255, 255, 255, 0.1);
            --glow-primary: rgba(0, 255, 65, 0.3);
            --glow-secondary: rgba(0, 212, 255, 0.3);
            
            /* Typography */
            --font-mono: 'JetBrains Mono', 'Consolas', monospace;
            --font-sans: 'Inter', 'Arial', sans-serif;
            
            /* Spacing */
            --spacing-xs: 0.5rem;
            --spacing-sm: 1rem;
            --spacing-md: 1.5rem;
            --spacing-lg: 2rem;
            --spacing-xl: 3rem;
            
            /* Dimensions */
            --header-height: 60px;
            --sidebar-width: 280px;
            --panel-border-radius: 8px;
        }

        body {
            font-family: var(--font-sans);
            background: var(--bg-primary);
            color: var(--text-primary);
            overflow-x: hidden;
            background-image: 
                linear-gradient(rgba(0, 255, 65, 0.05) 1px, transparent 1px),
                linear-gradient(90deg, rgba(0, 255, 65, 0.05) 1px, transparent 1px);
            background-size: 20px 20px;
        }

        /* Toast Notifications */
        .toast {
            position: fixed;
            bottom: 20px;
            right: 20px;
            padding: 12px 20px;
            background: var(--bg-panel);
            border: 1px solid var(--border);
            border-radius: 8px;
            display: flex;
            align-items: center;
            gap: 10px;
            transform: translateX(400px);
            transition: transform 0.3s ease;
            z-index: 10000;
        }

        .toast-show {
            transform: translateX(0);
        }

        .toast-success { border-color: var(--accent-success); }
        .toast-error { border-color: var(--accent-error); }
        .toast-warning { border-color: var(--accent-warning); }
        .toast-info { border-color: var(--accent-secondary); }

        /* Loading Spinner */
        .loading-container {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 40px;
            gap: 20px;
        }

        .loading-spinner {
            width: 40px;
            height: 40px;
            border: 3px solid var(--border);
            border-top-color: var(--accent-primary);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        .loading-message {
            color: var(--text-secondary);
            font-size: 14px;
        }

        /* Icon Styles */
        .nav-icon {
            width: 18px;
            height: 18px;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: var(--spacing-xs);
        }
        
        .logo i {
            width: 24px;
            height: 24px;
        }

        .header-button i {
            width: 16px;
            height: 16px;
        }

        .toast-icon i {
            width: 16px;
            height: 16px;
        }
    </style>
`;

// Export for use in other files
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        Navigation,
        Header,
        Toast,
        LoadingSpinner,
        TestGenieAPI,
        sharedStyles
    };
}
