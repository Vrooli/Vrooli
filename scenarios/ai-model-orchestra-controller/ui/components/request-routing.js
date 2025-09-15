/**
 * Request Routing Component
 * Interactive AI request routing interface with model selection and testing
 */

import apiClient from '../services/api-client.js';
import formatters from '../utils/formatters.js';

export class RequestRoutingComponent {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.isProcessing = false;
        this.lastResponse = null;
    }

    /**
     * Initialize the component
     */
    async init() {
        this.render();
        this.attachEventListeners();
    }

    /**
     * Render the component structure
     */
    render() {
        this.container.innerHTML = `
            <div class="request-routing-component">
                <div class="component-header">
                    <h2 class="text-xl font-semibold">Request Routing</h2>
                    <div class="header-info">
                        Test AI model selection and routing
                    </div>
                </div>
                
                <div class="routing-interface">
                    <!-- Model Selection Test -->
                    <div class="section-card">
                        <h3 class="section-title">Model Selection</h3>
                        <div class="form-group">
                            <label for="task-type">Task Type</label>
                            <select id="task-type" class="form-control">
                                <option value="completion">Completion</option>
                                <option value="reasoning">Reasoning</option>
                                <option value="code">Code Generation</option>
                                <option value="embedding">Embedding</option>
                                <option value="analysis">Analysis</option>
                            </select>
                        </div>
                        
                        <div class="form-row">
                            <div class="form-group">
                                <label for="complexity">Complexity</label>
                                <select id="complexity" class="form-control">
                                    <option value="">Auto</option>
                                    <option value="simple">Simple</option>
                                    <option value="moderate">Moderate</option>
                                    <option value="complex">Complex</option>
                                </select>
                            </div>
                            
                            <div class="form-group">
                                <label for="priority">Priority</label>
                                <select id="priority" class="form-control">
                                    <option value="">Normal</option>
                                    <option value="low">Low</option>
                                    <option value="high">High</option>
                                    <option value="critical">Critical</option>
                                </select>
                            </div>
                        </div>
                        
                        <div class="form-group">
                            <label for="cost-limit">Cost Limit (optional)</label>
                            <input type="number" id="cost-limit" class="form-control" 
                                   placeholder="0.01" step="0.001" min="0">
                        </div>
                        
                        <button id="select-model-btn" class="btn btn-primary">
                            Select Optimal Model
                        </button>
                    </div>
                    
                    <!-- Request Routing Test -->
                    <div class="section-card">
                        <h3 class="section-title">Route Request</h3>
                        <div class="form-group">
                            <label for="prompt">Prompt</label>
                            <textarea id="prompt" class="form-control" rows="4" 
                                placeholder="Enter your prompt here...">Explain quantum computing in simple terms.</textarea>
                        </div>
                        
                        <div class="form-row">
                            <div class="form-group">
                                <label for="max-tokens">Max Tokens</label>
                                <input type="number" id="max-tokens" class="form-control" 
                                       value="500" min="1" max="4000">
                            </div>
                            
                            <div class="form-group">
                                <label for="temperature">Temperature</label>
                                <input type="number" id="temperature" class="form-control" 
                                       value="0.7" min="0" max="2" step="0.1">
                            </div>
                        </div>
                        
                        <button id="route-request-btn" class="btn btn-primary">
                            Route Request
                        </button>
                    </div>
                </div>
                
                <!-- Results Display -->
                <div id="routing-results" class="results-section hidden">
                    <h3 class="section-title">Results</h3>
                    <div id="results-content" class="results-content"></div>
                </div>
                
                <!-- Processing Indicator -->
                <div id="processing-indicator" class="processing hidden">
                    <div class="spinner"></div>
                    <span>Processing request...</span>
                </div>
                
                <!-- Error Display -->
                <div id="routing-error" class="error-message hidden"></div>
            </div>
        `;
    }

    /**
     * Attach event listeners
     */
    attachEventListeners() {
        // Model selection button
        const selectBtn = document.getElementById('select-model-btn');
        if (selectBtn) {
            selectBtn.addEventListener('click', () => this.handleModelSelection());
        }

        // Route request button
        const routeBtn = document.getElementById('route-request-btn');
        if (routeBtn) {
            routeBtn.addEventListener('click', () => this.handleRequestRouting());
        }

        // Task type change updates placeholder
        const taskType = document.getElementById('task-type');
        if (taskType) {
            taskType.addEventListener('change', () => this.updatePromptPlaceholder());
        }
    }

    /**
     * Handle model selection
     */
    async handleModelSelection() {
        if (this.isProcessing) return;
        
        this.isProcessing = true;
        this.showProcessing(true);
        this.hideError();
        
        const taskType = document.getElementById('task-type').value;
        const complexity = document.getElementById('complexity').value;
        const priority = document.getElementById('priority').value;
        const costLimit = parseFloat(document.getElementById('cost-limit').value) || undefined;
        
        const requirements = {};
        if (complexity) requirements.complexity = complexity;
        if (priority) requirements.priority = priority;
        if (costLimit) requirements.costLimit = costLimit;
        
        try {
            const response = await apiClient.selectModel(taskType, requirements);
            
            if (response.success) {
                this.displaySelectionResults(response.data);
            } else {
                this.showError('Model selection failed: ' + (response.error || 'Unknown error'));
            }
        } catch (error) {
            console.error('Model selection error:', error);
            this.showError('Failed to select model: ' + error.message);
        } finally {
            this.isProcessing = false;
            this.showProcessing(false);
        }
    }

    /**
     * Handle request routing
     */
    async handleRequestRouting() {
        if (this.isProcessing) return;
        
        this.isProcessing = true;
        this.showProcessing(true);
        this.hideError();
        
        const taskType = document.getElementById('task-type').value;
        const prompt = document.getElementById('prompt').value.trim();
        const maxTokens = parseInt(document.getElementById('max-tokens').value) || 500;
        const temperature = parseFloat(document.getElementById('temperature').value) || 0.7;
        
        if (!prompt) {
            this.showError('Please enter a prompt');
            this.isProcessing = false;
            this.showProcessing(false);
            return;
        }
        
        const requirements = {
            maxTokens,
            temperature
        };
        
        try {
            const response = await apiClient.routeRequest(taskType, prompt, requirements);
            
            if (response.success) {
                this.displayRoutingResults(response.data);
            } else {
                this.showError('Request routing failed: ' + (response.error || 'Unknown error'));
            }
        } catch (error) {
            console.error('Request routing error:', error);
            this.showError('Failed to route request: ' + error.message);
        } finally {
            this.isProcessing = false;
            this.showProcessing(false);
        }
    }

    /**
     * Display model selection results
     */
    displaySelectionResults(data) {
        const resultsSection = document.getElementById('routing-results');
        const resultsContent = document.getElementById('results-content');
        
        resultsContent.innerHTML = `
            <div class="result-card selection-result">
                <h4>Model Selection Result</h4>
                <div class="result-details">
                    <div class="detail-row">
                        <span class="label">Request ID:</span>
                        <span class="value mono">${data.requestId}</span>
                    </div>
                    <div class="detail-row">
                        <span class="label">Selected Model:</span>
                        <span class="value highlight">${data.selectedModel}</span>
                    </div>
                    <div class="detail-row">
                        <span class="label">Task Type:</span>
                        <span class="value">${data.taskType}</span>
                    </div>
                    <div class="detail-row">
                        <span class="label">Fallback Used:</span>
                        <span class="value">${data.fallbackUsed ? '✅ Yes' : '❌ No'}</span>
                    </div>
                </div>
                
                ${data.alternatives && data.alternatives.length > 0 ? `
                    <div class="alternatives">
                        <h5>Alternative Models:</h5>
                        <ul>
                            ${data.alternatives.map(alt => `<li>${alt}</li>`).join('')}
                        </ul>
                    </div>
                ` : ''}
                
                ${data.systemMetrics ? `
                    <div class="system-metrics">
                        <h5>System Metrics:</h5>
                        <div class="metrics-grid">
                            <div class="metric">
                                <span>Memory Pressure:</span>
                                <span class="${this.getPressureClass(data.systemMetrics.memoryPressure)}">
                                    ${formatters.formatPercent(data.systemMetrics.memoryPressure * 100)}
                                </span>
                            </div>
                            <div class="metric">
                                <span>Available Memory:</span>
                                <span>${formatters.formatMemoryGB(data.systemMetrics.availableMemoryGb)}</span>
                            </div>
                            <div class="metric">
                                <span>CPU Usage:</span>
                                <span>${formatters.formatPercent(data.systemMetrics.cpuUsage)}</span>
                            </div>
                        </div>
                    </div>
                ` : ''}
                
                ${data.modelInfo ? `
                    <div class="model-info">
                        <h5>Model Information:</h5>
                        <div class="info-grid">
                            <div class="info-item">
                                <span>Speed:</span>
                                <span>${data.modelInfo.speed}</span>
                            </div>
                            <div class="info-item">
                                <span>Quality Tier:</span>
                                ${formatters.getModelTierBadge(data.modelInfo.quality_tier)}
                            </div>
                            <div class="info-item">
                                <span>Capabilities:</span>
                                <div>${formatters.formatCapabilities(data.modelInfo.capabilities)}</div>
                            </div>
                        </div>
                    </div>
                ` : ''}
            </div>
        `;
        
        resultsSection.classList.remove('hidden');
    }

    /**
     * Display routing results
     */
    displayRoutingResults(data) {
        const resultsSection = document.getElementById('routing-results');
        const resultsContent = document.getElementById('results-content');
        
        resultsContent.innerHTML = `
            <div class="result-card routing-result">
                <h4>Request Routing Result</h4>
                <div class="result-details">
                    <div class="detail-row">
                        <span class="label">Request ID:</span>
                        <span class="value mono">${data.requestId}</span>
                    </div>
                    <div class="detail-row">
                        <span class="label">Model Used:</span>
                        <span class="value highlight">${data.selectedModel}</span>
                    </div>
                    <div class="detail-row">
                        <span class="label">Fallback Used:</span>
                        <span class="value">${data.fallbackUsed ? '✅ Yes' : '❌ No'}</span>
                    </div>
                </div>
                
                <div class="response-section">
                    <h5>Response:</h5>
                    <div class="response-content">
                        ${this.escapeHtml(data.response)}
                    </div>
                </div>
                
                ${data.metrics ? `
                    <div class="performance-metrics">
                        <h5>Performance Metrics:</h5>
                        <div class="metrics-grid">
                            <div class="metric">
                                <span>Response Time:</span>
                                <span>${formatters.formatResponseTime(data.metrics.responseTimeMs)}</span>
                            </div>
                            <div class="metric">
                                <span>Tokens Generated:</span>
                                <span>${data.metrics.tokensGenerated || 'N/A'}</span>
                            </div>
                            <div class="metric">
                                <span>Prompt Tokens:</span>
                                <span>${data.metrics.promptTokens || 'N/A'}</span>
                            </div>
                            <div class="metric">
                                <span>Memory Pressure:</span>
                                <span class="${this.getPressureClass(data.metrics.memoryPressure)}">
                                    ${formatters.formatPercent(data.metrics.memoryPressure * 100)}
                                </span>
                            </div>
                        </div>
                    </div>
                ` : ''}
            </div>
        `;
        
        resultsSection.classList.remove('hidden');
        this.lastResponse = data;
    }

    /**
     * Update prompt placeholder based on task type
     */
    updatePromptPlaceholder() {
        const taskType = document.getElementById('task-type').value;
        const promptField = document.getElementById('prompt');
        
        const placeholders = {
            'completion': 'Complete this sentence: The future of AI is...',
            'reasoning': 'Explain why water expands when it freezes.',
            'code': 'Write a Python function to calculate fibonacci numbers.',
            'embedding': 'Text to generate embeddings for...',
            'analysis': 'Analyze the sentiment of this text: I love this product!'
        };
        
        promptField.placeholder = placeholders[taskType] || 'Enter your prompt here...';
    }

    /**
     * Get pressure class for styling
     */
    getPressureClass(pressure) {
        if (pressure < 0.5) return 'text-green-500';
        if (pressure < 0.7) return 'text-yellow-500';
        if (pressure < 0.9) return 'text-orange-500';
        return 'text-red-500';
    }

    /**
     * Escape HTML for safe display
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Show/hide processing indicator
     */
    showProcessing(show) {
        const indicator = document.getElementById('processing-indicator');
        if (show) {
            indicator.classList.remove('hidden');
        } else {
            indicator.classList.add('hidden');
        }
    }

    /**
     * Show error message
     */
    showError(message) {
        const errorDiv = document.getElementById('routing-error');
        errorDiv.textContent = message;
        errorDiv.classList.remove('hidden');
    }

    /**
     * Hide error message
     */
    hideError() {
        const errorDiv = document.getElementById('routing-error');
        errorDiv.classList.add('hidden');
    }

    /**
     * Cleanup on destroy
     */
    destroy() {
        this.container.innerHTML = '';
    }
}

export default RequestRoutingComponent;