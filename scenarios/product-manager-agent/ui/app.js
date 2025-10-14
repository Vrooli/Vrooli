import { initIframeBridgeChild } from '/node_modules/@vrooli/iframe-bridge/dist/iframeBridgeChild.js';

(function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window.parent === window || window.__productManagerBridgeInitialized) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[ProductManagerAgent] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'product-manager-agent' });
    window.__productManagerBridgeInitialized = true;
})();

// Product Manager Agent - Main Application
const API_URL = `http://localhost:${window.location.port === '4200' ? '9200' : window.location.port - 1000}`;

class ProductManagerApp {
    constructor() {
        this.currentView = 'dashboard';
        this.features = [];
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadDashboard();
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const view = e.currentTarget.dataset.view;
                this.switchView(view);
            });
        });

        // New Feature Button
        document.getElementById('new-feature-btn').addEventListener('click', () => {
            this.showFeatureModal();
        });

        // Modal Controls
        document.getElementById('cancel-modal').addEventListener('click', () => {
            this.hideFeatureModal();
        });

        document.getElementById('feature-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.submitFeature(e.target);
        });

        // Close modal on outside click
        document.getElementById('feature-modal').addEventListener('click', (e) => {
            if (e.target.id === 'feature-modal') {
                this.hideFeatureModal();
            }
        });
    }

    switchView(view) {
        // Update navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-view="${view}"]`).classList.add('active');

        // Update view title
        const titles = {
            dashboard: 'Dashboard',
            features: 'Feature Management',
            roadmap: 'Product Roadmap',
            analytics: 'Analytics & Insights',
            decisions: 'Decision Log'
        };
        document.getElementById('view-title').textContent = titles[view] || view;

        // Show/hide views
        document.querySelectorAll('.view').forEach(v => {
            v.classList.remove('active');
        });
        const viewElement = document.getElementById(`${view}-view`);
        if (viewElement) {
            viewElement.classList.add('active');
        }

        this.currentView = view;

        // Load view-specific data
        switch(view) {
            case 'features':
                this.loadFeatures();
                break;
            case 'dashboard':
                this.loadDashboard();
                break;
        }
    }

    async loadDashboard() {
        // In a real app, this would fetch from the API
        // For now, we'll use mock data or the prioritizer workflow
        try {
            const response = await fetch(`${API_URL}/api/dashboard`);
            if (response.ok) {
                const data = await response.json();
                this.updateDashboard(data);
            }
        } catch (error) {
            console.log('Using mock dashboard data');
            // Continue with existing mock data
        }
    }

    async loadFeatures() {
        const featuresGrid = document.getElementById('features-grid');
        featuresGrid.innerHTML = '<div class="loading">Loading features...</div>';

        try {
            const response = await fetch(`${API_URL}/api/features`);
            if (response.ok) {
                this.features = await response.json();
                this.renderFeatures();
            } else {
                throw new Error('Failed to load features');
            }
        } catch (error) {
            // Use mock data for demonstration
            this.features = [
                {
                    name: 'Dark Mode Support',
                    description: 'Implement system-wide dark mode with user preferences',
                    reach: 5000,
                    impact: 4,
                    confidence: 0.9,
                    effort: 5,
                    priority: 'CRITICAL',
                    score: 7.2
                },
                {
                    name: 'API Rate Limiting',
                    description: 'Implement rate limiting to prevent API abuse',
                    reach: 3000,
                    impact: 5,
                    confidence: 0.95,
                    effort: 4,
                    priority: 'HIGH',
                    score: 8.9
                },
                {
                    name: 'Export to PDF',
                    description: 'Allow users to export reports and data to PDF format',
                    reach: 2000,
                    impact: 3,
                    confidence: 0.8,
                    effort: 3,
                    priority: 'MEDIUM',
                    score: 5.3
                }
            ];
            this.renderFeatures();
        }
    }

    renderFeatures() {
        const featuresGrid = document.getElementById('features-grid');
        
        if (this.features.length === 0) {
            featuresGrid.innerHTML = '<div class="empty-state">No features found. Add your first feature!</div>';
            return;
        }

        featuresGrid.innerHTML = this.features.map(feature => `
            <div class="card feature-card ${feature.priority.toLowerCase()}">
                <div class="feature-header">
                    <span class="priority-badge">${feature.priority}</span>
                    <span class="feature-score">Score: ${feature.score}</span>
                </div>
                <h3>${feature.name}</h3>
                <p>${feature.description}</p>
                <div class="feature-metrics">
                    <div class="metric">
                        <span class="label">Reach:</span>
                        <span class="value">${feature.reach}</span>
                    </div>
                    <div class="metric">
                        <span class="label">Impact:</span>
                        <span class="value">${feature.impact}/5</span>
                    </div>
                    <div class="metric">
                        <span class="label">Confidence:</span>
                        <span class="value">${(feature.confidence * 100).toFixed(0)}%</span>
                    </div>
                    <div class="metric">
                        <span class="label">Effort:</span>
                        <span class="value">${feature.effort}/10</span>
                    </div>
                </div>
            </div>
        `).join('');
    }

    showFeatureModal() {
        document.getElementById('feature-modal').classList.add('active');
    }

    hideFeatureModal() {
        document.getElementById('feature-modal').classList.remove('active');
        document.getElementById('feature-form').reset();
    }

    async submitFeature(form) {
        const formData = new FormData(form);
        const feature = {
            name: formData.get('name'),
            description: formData.get('description'),
            reach: parseInt(formData.get('reach')),
            impact: parseInt(formData.get('impact')),
            confidence: parseFloat(formData.get('confidence')),
            effort: parseInt(formData.get('effort'))
        };

        try {
            // Call the prioritizer workflow
            const response = await fetch('http://localhost:5678/webhook/feature-prioritizer', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    features: [feature],
                    method: 'RICE',
                    context: 'User submitted feature for prioritization'
                })
            });

            if (response.ok) {
                const result = await response.json();
                console.log('Feature prioritized:', result);
                
                // Add to local features list
                if (result.prioritization_results && result.prioritization_results.prioritized_features) {
                    this.features.unshift(...result.prioritization_results.prioritized_features);
                }
                
                this.hideFeatureModal();
                
                // Refresh current view
                if (this.currentView === 'features') {
                    this.renderFeatures();
                } else {
                    this.loadDashboard();
                }
                
                // Show success message
                this.showNotification('Feature added and prioritized successfully!', 'success');
            } else {
                throw new Error('Failed to prioritize feature');
            }
        } catch (error) {
            console.error('Error submitting feature:', error);
            this.showNotification('Failed to add feature. Please try again.', 'error');
        }
    }

    updateDashboard(data) {
        // Update dashboard with real data
        // This would be implemented based on actual API response
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 1rem 1.5rem;
            background: ${type === 'success' ? 'var(--success)' : 'var(--danger)'};
            color: white;
            border-radius: 0.5rem;
            z-index: 2000;
            animation: slideIn 0.3s ease;
        `;
        
        document.body.appendChild(notification);
        
        // Remove after 3 seconds
        setTimeout(() => {
            notification.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => notification.remove(), 300);
        }, 3000);
    }
}

// Add animation styles
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(100%);
            opacity: 0;
        }
    }
    
    .feature-card {
        transition: transform 0.2s, box-shadow 0.2s;
    }
    
    .feature-card:hover {
        transform: translateY(-4px);
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
    }
    
    .feature-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1rem;
    }
    
    .feature-card h3 {
        margin-bottom: 0.75rem;
        color: var(--text-primary);
    }
    
    .feature-card p {
        color: var(--text-secondary);
        font-size: 0.875rem;
        margin-bottom: 1rem;
    }
    
    .feature-metrics {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 0.5rem;
        padding-top: 1rem;
        border-top: 1px solid var(--border-color);
    }
    
    .feature-metrics .metric {
        display: flex;
        justify-content: space-between;
        font-size: 0.75rem;
    }
    
    .feature-metrics .label {
        color: var(--text-secondary);
    }
    
    .feature-metrics .value {
        color: var(--text-primary);
        font-weight: 500;
    }
    
    .empty-state {
        text-align: center;
        padding: 3rem;
        color: var(--text-secondary);
    }
    
    .loading {
        text-align: center;
        padding: 2rem;
        color: var(--text-secondary);
    }
`;
document.head.appendChild(style);

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.app = new ProductManagerApp();
});
