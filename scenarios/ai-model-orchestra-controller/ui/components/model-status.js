/**
 * Model Status Component
 * Displays AI model availability, health, and performance metrics
 */

import apiClient from '../services/api-client.js';
import formatters from '../utils/formatters.js';

export class ModelStatusComponent {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.refreshInterval = 10000; // 10 seconds
        this.intervalId = null;
        this.models = [];
    }

    /**
     * Initialize and start the component
     */
    async init() {
        this.render();
        await this.loadModels();
        this.startAutoRefresh();
    }

    /**
     * Load model data from API
     */
    async loadModels() {
        try {
            const response = await apiClient.getModelsStatus(true);
            if (response.success) {
                this.models = response.data.models || [];
                this.updateDisplay();
            } else {
                this.showError('Failed to load model status');
            }
        } catch (error) {
            console.error('Error loading models:', error);
            this.showError('Error connecting to API');
        }
    }

    /**
     * Render the component structure
     */
    render() {
        this.container.innerHTML = `
            <div class="model-status-component">
                <div class="component-header">
                    <h2 class="text-xl font-semibold">AI Models</h2>
                    <div class="header-actions">
                        <button id="refresh-models" class="btn-icon" title="Refresh">
                            🔄
                        </button>
                        <span class="status-indicator" id="model-status-indicator">
                            <span class="pulse"></span>
                        </span>
                    </div>
                </div>
                <div id="models-grid" class="models-grid">
                    <div class="loading">Loading models...</div>
                </div>
                <div id="model-error" class="error-message hidden"></div>
            </div>
        `;

        // Attach event listeners
        this.attachEventListeners();
    }

    /**
     * Update the display with model data
     */
    updateDisplay() {
        const grid = document.getElementById('models-grid');
        
        if (this.models.length === 0) {
            grid.innerHTML = '<div class="no-data">No models available</div>';
            return;
        }

        const modelsHTML = this.models.map(model => this.createModelCard(model)).join('');
        grid.innerHTML = `<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">${modelsHTML}</div>`;
        
        // Update status indicator
        this.updateStatusIndicator();
    }

    /**
     * Create a model card HTML
     */
    createModelCard(model) {
        const statusColor = model.healthy ? 'bg-green-100' : 'bg-red-100';
        const statusText = model.healthy ? 'Healthy' : 'Unhealthy';
        const statusIcon = model.healthy ? '✅' : '❌';
        
        return `
            <div class="model-card ${statusColor} p-4 rounded-lg shadow">
                <div class="model-header">
                    <h3 class="font-semibold">${model.model_name}</h3>
                    <span class="status-badge">${statusIcon} ${statusText}</span>
                </div>
                
                <div class="model-metrics">
                    <div class="metric">
                        <span class="label">Response Time:</span>
                        <span class="value">${formatters.formatResponseTime(model.avg_response_time_ms)}</span>
                    </div>
                    <div class="metric">
                        <span class="label">Requests:</span>
                        <span class="value">${model.request_count}</span>
                    </div>
                    <div class="metric">
                        <span class="label">Success Rate:</span>
                        <span class="value">${this.calculateSuccessRate(model)}%</span>
                    </div>
                    <div class="metric">
                        <span class="label">Memory:</span>
                        <span class="value">${formatters.formatMemoryGB(model.ram_required_gb || 0)}</span>
                    </div>
                </div>
                
                <div class="model-details">
                    <div class="capabilities">
                        ${formatters.formatCapabilities(model.capabilities)}
                    </div>
                    <div class="quality-tier">
                        ${formatters.getModelTierBadge(model.quality_tier || 'basic')}
                    </div>
                </div>
                
                <div class="model-footer">
                    <span class="last-used">
                        ${model.last_used ? 
                            `Last used: ${formatters.formatRelativeTime(model.last_used)}` : 
                            'Never used'}
                    </span>
                </div>
            </div>
        `;
    }

    /**
     * Calculate success rate for a model
     */
    calculateSuccessRate(model) {
        if (!model.request_count || model.request_count === 0) return 100;
        return ((model.success_count / model.request_count) * 100).toFixed(1);
    }

    /**
     * Update status indicator
     */
    updateStatusIndicator() {
        const indicator = document.getElementById('model-status-indicator');
        const healthyCount = this.models.filter(m => m.healthy).length;
        const totalCount = this.models.length;
        
        if (healthyCount === totalCount) {
            indicator.className = 'status-indicator status-healthy';
        } else if (healthyCount > 0) {
            indicator.className = 'status-indicator status-degraded';
        } else {
            indicator.className = 'status-indicator status-unhealthy';
        }
    }

    /**
     * Show error message
     */
    showError(message) {
        const errorDiv = document.getElementById('model-error');
        errorDiv.textContent = message;
        errorDiv.classList.remove('hidden');
        setTimeout(() => errorDiv.classList.add('hidden'), 5000);
    }

    /**
     * Attach event listeners
     */
    attachEventListeners() {
        const refreshBtn = document.getElementById('refresh-models');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.loadModels());
        }
    }

    /**
     * Start auto-refresh
     */
    startAutoRefresh() {
        this.intervalId = setInterval(() => this.loadModels(), this.refreshInterval);
    }

    /**
     * Stop auto-refresh
     */
    stopAutoRefresh() {
        if (this.intervalId) {
            clearInterval(this.intervalId);
            this.intervalId = null;
        }
    }

    /**
     * Cleanup on destroy
     */
    destroy() {
        this.stopAutoRefresh();
        this.container.innerHTML = '';
    }
}

export default ModelStatusComponent;