/**
 * Frontend Application Test Module
 * Test fixture for JavaScript code analysis and web development workflows
 */

// Modern JavaScript with ES6+ features for testing
class ResourceManager {
    constructor(baseUrl = 'http://localhost:3000') {
        this.baseUrl = baseUrl;
        this.resources = new Map();
        this.eventListeners = new Map();
        this.initialized = false;
        
        // Bind methods to preserve context
        this.handleError = this.handleError.bind(this);
        this.updateUI = this.updateUI.bind(this);
    }
    
    /**
     * Initialize the resource manager
     * @param {Object} config - Configuration options
     * @param {boolean} config.autoRefresh - Enable automatic refresh
     * @param {number} config.refreshInterval - Refresh interval in milliseconds
     * @returns {Promise<void>}
     */
    async initialize(config = {}) {
        const defaultConfig = {
            autoRefresh: true,
            refreshInterval: 30000,
            retryAttempts: 3,
            timeout: 10000
        };
        
        this.config = { ...defaultConfig, ...config };
        
        try {
            console.log('Initializing Resource Manager...');
            
            // Discover available resources
            await this.discoverResources();
            
            // Set up event listeners
            this.setupEventListeners();
            
            // Start auto-refresh if enabled
            if (this.config.autoRefresh) {
                this.startAutoRefresh();
            }
            
            this.initialized = true;
            console.log('Resource Manager initialized successfully');
            
            // Dispatch initialization event
            this.dispatchEvent('initialized', { resources: this.resources.size });
            
        } catch (error) {
            console.error('Failed to initialize Resource Manager:', error);
            this.handleError(error);
            throw error;
        }
    }
    
    /**
     * Discover available resources from the backend
     * @returns {Promise<Array>} Array of discovered resources
     */
    async discoverResources() {
        const response = await this.makeRequest('/api/resources/discover', {
            method: 'GET',
            timeout: this.config.timeout
        });
        
        if (!response.success) {
            throw new Error(`Resource discovery failed: ${response.error}`);
        }
        
        // Process discovered resources
        const resources = response.data.resources || [];
        
        for (const resource of resources) {
            this.addResource(resource);
        }
        
        console.log(`Discovered ${resources.length} resources`);
        return resources;
    }
    
    /**
     * Add a resource to the manager
     * @param {Object} resource - Resource configuration
     * @param {string} resource.id - Resource identifier
     * @param {string} resource.name - Resource name
     * @param {string} resource.type - Resource type (ai, automation, storage, etc.)
     * @param {string} resource.status - Resource status
     * @param {string} resource.baseUrl - Resource base URL
     */
    addResource(resource) {
        if (!resource.id || !resource.name) {
            throw new Error('Resource must have id and name properties');
        }
        
        const enrichedResource = {
            ...resource,
            addedAt: new Date().toISOString(),
            lastChecked: null,
            metrics: {
                requestCount: 0,
                errorCount: 0,
                avgResponseTime: 0
            }
        };
        
        this.resources.set(resource.id, enrichedResource);
        this.dispatchEvent('resource-added', enrichedResource);
    }
    
    /**
     * Get resource by ID
     * @param {string} resourceId - Resource identifier
     * @returns {Object|null} Resource object or null if not found
     */
    getResource(resourceId) {
        return this.resources.get(resourceId) || null;
    }
    
    /**
     * Get resources by type
     * @param {string} type - Resource type to filter by
     * @returns {Array} Array of resources matching the type
     */
    getResourcesByType(type) {
        return Array.from(this.resources.values())
            .filter(resource => resource.type === type);
    }
    
    /**
     * Check health of a specific resource
     * @param {string} resourceId - Resource identifier
     * @returns {Promise<Object>} Health check result
     */
    async checkResourceHealth(resourceId) {
        const resource = this.getResource(resourceId);
        if (!resource) {
            throw new Error(`Resource not found: ${resourceId}`);
        }
        
        const startTime = Date.now();
        
        try {
            const response = await this.makeRequest(`/api/resources/${resourceId}/health`, {
                method: 'GET',
                timeout: 5000
            });
            
            const responseTime = Date.now() - startTime;
            
            // Update resource metrics
            resource.metrics.requestCount++;
            resource.metrics.avgResponseTime = 
                (resource.metrics.avgResponseTime + responseTime) / 2;
            resource.lastChecked = new Date().toISOString();
            
            const healthData = {
                resourceId,
                healthy: response.success,
                responseTime,
                details: response.data || {},
                timestamp: new Date().toISOString()
            };
            
            this.dispatchEvent('health-check', healthData);
            return healthData;
            
        } catch (error) {
            resource.metrics.errorCount++;
            
            const errorData = {
                resourceId,
                healthy: false,
                error: error.message,
                responseTime: Date.now() - startTime,
                timestamp: new Date().toISOString()
            };
            
            this.dispatchEvent('health-check-error', errorData);
            throw error;
        }
    }
    
    /**
     * Execute a task on a resource
     * @param {string} resourceId - Resource identifier
     * @param {string} task - Task to execute
     * @param {Object} params - Task parameters
     * @returns {Promise<Object>} Task execution result
     */
    async executeTask(resourceId, task, params = {}) {
        const resource = this.getResource(resourceId);
        if (!resource) {
            throw new Error(`Resource not found: ${resourceId}`);
        }
        
        const taskData = {
            resourceId,
            task,
            params,
            timestamp: new Date().toISOString()
        };
        
        console.log(`Executing task '${task}' on resource '${resourceId}'`);
        this.dispatchEvent('task-started', taskData);
        
        try {
            const response = await this.makeRequest(`/api/resources/${resourceId}/execute`, {
                method: 'POST',
                body: JSON.stringify({ task, params }),
                headers: {
                    'Content-Type': 'application/json'
                }
            });
            
            if (!response.success) {
                throw new Error(`Task execution failed: ${response.error}`);
            }
            
            const result = {
                ...taskData,
                result: response.data,
                success: true,
                completedAt: new Date().toISOString()
            };
            
            this.dispatchEvent('task-completed', result);
            return result;
            
        } catch (error) {
            const errorResult = {
                ...taskData,
                error: error.message,
                success: false,
                completedAt: new Date().toISOString()
            };
            
            this.dispatchEvent('task-error', errorResult);
            throw error;
        }
    }
    
    /**
     * Make HTTP request with retry logic
     * @private
     */
    async makeRequest(endpoint, options = {}) {
        const url = `${this.baseUrl}${endpoint}`;
        const defaultOptions = {
            method: 'GET',
            timeout: this.config.timeout,
            headers: {
                'Accept': 'application/json'
            }
        };
        
        const requestOptions = { ...defaultOptions, ...options };
        
        for (let attempt = 1; attempt <= this.config.retryAttempts; attempt++) {
            try {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), requestOptions.timeout);
                
                const response = await fetch(url, {
                    ...requestOptions,
                    signal: controller.signal
                });
                
                clearTimeout(timeoutId);
                
                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                }
                
                const data = await response.json();
                return { success: true, data };
                
            } catch (error) {
                console.warn(`Request attempt ${attempt} failed:`, error.message);
                
                if (attempt === this.config.retryAttempts) {
                    return { success: false, error: error.message };
                }
                
                // Exponential backoff
                await new Promise(resolve => 
                    setTimeout(resolve, Math.pow(2, attempt) * 1000)
                );
            }
        }
    }
    
    /**
     * Set up event listeners for UI interactions
     * @private
     */
    setupEventListeners() {
        // Listen for resource updates
        this.addEventListener('resource-added', this.updateUI);
        this.addEventListener('health-check', this.updateUI);
        this.addEventListener('task-completed', this.updateUI);
        
        // Handle window events
        window.addEventListener('beforeunload', () => {
            this.cleanup();
        });
        
        // Handle visibility changes for performance optimization
        document.addEventListener('visibilitychange', () => {
            if (document.hidden) {
                this.pauseAutoRefresh();
            } else {
                this.resumeAutoRefresh();
            }
        });
    }
    
    /**
     * Start automatic resource refresh
     * @private
     */
    startAutoRefresh() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
        
        this.refreshInterval = setInterval(async () => {
            try {
                await this.discoverResources();
            } catch (error) {
                console.error('Auto-refresh failed:', error);
            }
        }, this.config.refreshInterval);
    }
    
    /**
     * Stop automatic resource refresh
     */
    stopAutoRefresh() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
            this.refreshInterval = null;
        }
    }
    
    /**
     * Update the UI with current resource state
     * @private
     */
    updateUI(event) {
        const resourceList = document.getElementById('resource-list');
        if (!resourceList) return;
        
        // Clear existing content
        resourceList.innerHTML = '';
        
        // Render resources
        Array.from(this.resources.values()).forEach(resource => {
            const resourceElement = this.createResourceElement(resource);
            resourceList.appendChild(resourceElement);
        });
    }
    
    /**
     * Create DOM element for a resource
     * @private
     */
    createResourceElement(resource) {
        const element = document.createElement('div');
        element.className = `resource-item ${resource.status}`;
        element.innerHTML = `
            <div class="resource-header">
                <h3>${resource.name}</h3>
                <span class="resource-status ${resource.status}">${resource.status}</span>
            </div>
            <div class="resource-details">
                <p><strong>Type:</strong> ${resource.type}</p>
                <p><strong>URL:</strong> ${resource.baseUrl}</p>
                <p><strong>Requests:</strong> ${resource.metrics.requestCount}</p>
                <p><strong>Avg Response:</strong> ${resource.metrics.avgResponseTime.toFixed(2)}ms</p>
            </div>
            <div class="resource-actions">
                <button onclick="resourceManager.checkResourceHealth('${resource.id}')">
                    Check Health
                </button>
                <button onclick="resourceManager.executeTask('${resource.id}', 'status')">
                    Get Status
                </button>
            </div>
        `;
        
        return element;
    }
    
    /**
     * Add event listener
     */
    addEventListener(eventType, callback) {
        if (!this.eventListeners.has(eventType)) {
            this.eventListeners.set(eventType, []);
        }
        this.eventListeners.get(eventType).push(callback);
    }
    
    /**
     * Dispatch custom event
     * @private
     */
    dispatchEvent(eventType, data) {
        const listeners = this.eventListeners.get(eventType) || [];
        listeners.forEach(listener => {
            try {
                listener({ type: eventType, data });
            } catch (error) {
                console.error(`Event listener error for ${eventType}:`, error);
            }
        });
    }
    
    /**
     * Handle errors consistently
     * @private
     */
    handleError(error) {
        console.error('Resource Manager Error:', error);
        
        // Show user-friendly error message
        const errorContainer = document.getElementById('error-container');
        if (errorContainer) {
            errorContainer.innerHTML = `
                <div class="error-message">
                    <strong>Error:</strong> ${error.message}
                    <button onclick="this.parentElement.remove()">Dismiss</button>
                </div>
            `;
        }
        
        // Report error to monitoring system
        this.reportError(error);
    }
    
    /**
     * Report error to monitoring system
     * @private
     */
    async reportError(error) {
        try {
            await this.makeRequest('/api/errors/report', {
                method: 'POST',
                body: JSON.stringify({
                    message: error.message,
                    stack: error.stack,
                    timestamp: new Date().toISOString(),
                    userAgent: navigator.userAgent,
                    url: window.location.href
                }),
                headers: {
                    'Content-Type': 'application/json'
                }
            });
        } catch (reportError) {
            console.warn('Failed to report error:', reportError);
        }
    }
    
    /**
     * Clean up resources and event listeners
     */
    cleanup() {
        this.stopAutoRefresh();
        this.eventListeners.clear();
        this.resources.clear();
        console.log('Resource Manager cleaned up');
    }
    
    // Utility methods for debugging and monitoring
    getStats() {
        return {
            totalResources: this.resources.size,
            resourcesByType: this.getResourceCountByType(),
            averageResponseTime: this.getAverageResponseTime(),
            totalRequests: this.getTotalRequests(),
            totalErrors: this.getTotalErrors()
        };
    }
    
    getResourceCountByType() {
        const counts = {};
        Array.from(this.resources.values()).forEach(resource => {
            counts[resource.type] = (counts[resource.type] || 0) + 1;
        });
        return counts;
    }
    
    getAverageResponseTime() {
        const resources = Array.from(this.resources.values());
        const totalTime = resources.reduce((sum, r) => sum + r.metrics.avgResponseTime, 0);
        return resources.length > 0 ? totalTime / resources.length : 0;
    }
    
    getTotalRequests() {
        return Array.from(this.resources.values())
            .reduce((sum, r) => sum + r.metrics.requestCount, 0);
    }
    
    getTotalErrors() {
        return Array.from(this.resources.values())
            .reduce((sum, r) => sum + r.metrics.errorCount, 0);
    }
}

// Initialize the resource manager when the page loads
let resourceManager;

document.addEventListener('DOMContentLoaded', async () => {
    try {
        resourceManager = new ResourceManager();
        await resourceManager.initialize({
            autoRefresh: true,
            refreshInterval: 30000
        });
        
        console.log('Application initialized successfully');
        
    } catch (error) {
        console.error('Failed to initialize application:', error);
    }
});

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { ResourceManager };
}

// Global access for debugging
window.ResourceManager = ResourceManager;