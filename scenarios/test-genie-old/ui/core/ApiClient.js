/**
 * ApiClient - Centralized API Communication Layer
 * Handles all HTTP requests with error handling, retries, and event emission
 */

import { eventBus, EVENT_TYPES } from './EventBus.js';
import { API_BASE_URL } from '../utils/constants.js';

/**
 * ApiClient class - Centralized API communication
 */
export class ApiClient {
    constructor(baseUrl = API_BASE_URL, eventBusInstance = eventBus) {
        this.baseUrl = baseUrl;
        this.eventBus = eventBusInstance;

        // Request configuration
        this.defaultTimeout = 30000; // 30 seconds
        this.maxRetries = 3;
        this.retryDelay = 1000; // 1 second

        // Request interceptors
        this.requestInterceptors = [];
        this.responseInterceptors = [];

        // In-flight requests tracking (for cancellation)
        this.inflightRequests = new Map();

        // Debug mode
        this.debug = false;
    }

    /**
     * Generic request method with error handling
     * @param {string} endpoint - API endpoint
     * @param {Object} options - Fetch options
     * @returns {Promise<any>}
     */
    async request(endpoint, options = {}) {
        const url = endpoint.startsWith('http') ? endpoint : `${this.baseUrl}${endpoint}`;
        const requestId = `${options.method || 'GET'}-${endpoint}-${Date.now()}`;

        // Default options
        const defaultOptions = {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
            timeout: this.defaultTimeout,
            retry: this.maxRetries,
            ...options
        };

        if (this.debug) {
            console.log(`[ApiClient] ${defaultOptions.method} ${endpoint}`, { options: defaultOptions });
        }

        // Apply request interceptors
        let finalOptions = defaultOptions;
        for (const interceptor of this.requestInterceptors) {
            finalOptions = await interceptor(url, finalOptions);
        }

        // Create abort controller for timeout
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), finalOptions.timeout);

        // Track in-flight request
        this.inflightRequests.set(requestId, controller);

        try {
            // Make the request
            const response = await fetch(url, {
                ...finalOptions,
                signal: controller.signal
            });

            clearTimeout(timeoutId);

            // Check for HTTP errors
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            // Parse response
            const contentType = response.headers.get('content-type');
            let data;
            if (contentType && contentType.includes('application/json')) {
                data = await response.json();
            } else {
                data = await response.text();
            }

            // Apply response interceptors
            for (const interceptor of this.responseInterceptors) {
                data = await interceptor(data, response);
            }

            if (this.debug) {
                console.log(`[ApiClient] Success: ${defaultOptions.method} ${endpoint}`, { data });
            }

            return data;

        } catch (error) {
            clearTimeout(timeoutId);

            // Handle abort/timeout
            if (error.name === 'AbortError') {
                const timeoutError = new Error(`Request timeout after ${finalOptions.timeout}ms`);
                timeoutError.isTimeout = true;
                throw timeoutError;
            }

            // Retry logic for retryable errors
            if (finalOptions.retry > 0 && this._isRetryableError(error)) {
                if (this.debug) {
                    console.log(`[ApiClient] Retrying (${finalOptions.retry} attempts left)...`);
                }

                await this._delay(this.retryDelay);

                return this.request(endpoint, {
                    ...options,
                    retry: finalOptions.retry - 1
                });
            }

            // Emit error event
            this.eventBus.emit(EVENT_TYPES.DATA_ERROR, {
                endpoint,
                error: error.message,
                method: finalOptions.method
            });

            if (this.debug) {
                console.error(`[ApiClient] Error: ${defaultOptions.method} ${endpoint}`, error);
            }

            throw error;

        } finally {
            this.inflightRequests.delete(requestId);
        }
    }

    /**
     * GET request with error handling
     * @param {string} endpoint
     * @param {Object} options
     * @returns {Promise<any>}
     */
    async get(endpoint, options = {}) {
        return this.request(endpoint, { ...options, method: 'GET' });
    }

    /**
     * POST request
     * @param {string} endpoint
     * @param {Object} data
     * @param {Object} options
     * @returns {Promise<any>}
     */
    async post(endpoint, data = null, options = {}) {
        return this.request(endpoint, {
            ...options,
            method: 'POST',
            body: data ? JSON.stringify(data) : undefined
        });
    }

    /**
     * PUT request
     * @param {string} endpoint
     * @param {Object} data
     * @param {Object} options
     * @returns {Promise<any>}
     */
    async put(endpoint, data = null, options = {}) {
        return this.request(endpoint, {
            ...options,
            method: 'PUT',
            body: data ? JSON.stringify(data) : undefined
        });
    }

    /**
     * DELETE request
     * @param {string} endpoint
     * @param {Object} options
     * @returns {Promise<any>}
     */
    async delete(endpoint, options = {}) {
        return this.request(endpoint, { ...options, method: 'DELETE' });
    }

    // System endpoints

    /**
     * Check API health
     * @returns {Promise<Object>}
     */
    async checkHealth() {
        try {
            const data = await this.get('/health');
            this.eventBus.emit(EVENT_TYPES.HEALTH_CHANGED, data);
            return data;
        } catch (error) {
            const errorData = {
                healthy: false,
                status: 'error',
                message: 'Connection failed',
                error: error.message
            };
            this.eventBus.emit(EVENT_TYPES.HEALTH_CHANGED, errorData);
            return errorData;
        }
    }

    /**
     * Get system metrics
     * @returns {Promise<Object>}
     */
    async getSystemMetrics() {
        return this.get('/system/metrics');
    }

    // Test Suite endpoints

    /**
     * Get all test suites
     * @returns {Promise<Array>}
     */
    async getTestSuites() {
        return this.get('/test-suites');
    }

    /**
     * Get specific test suite
     * @param {string} suiteId
     * @returns {Promise<Object>}
     */
    async getTestSuite(suiteId) {
        return this.get(`/test-suite/${suiteId}`);
    }

    /**
     * Generate test suite
     * @param {Object} config - Generation configuration
     * @returns {Promise<Object>}
     */
    async generateTestSuite(config) {
        const result = await this.post('/test-suite/generate', config);
        this.eventBus.emit(EVENT_TYPES.SUITE_CREATED, result);
        return result;
    }

    /**
     * Generate test suites in batch
     * @param {Object} config - Batch generation configuration
     * @returns {Promise<Object>}
     */
    async generateTestSuitesBatch(config) {
        const result = await this.post('/test-suite/generate-batch', config);
        this.eventBus.emit(EVENT_TYPES.SUITE_CREATED, result);
        return result;
    }

    /**
     * Execute test suite
     * @param {string} suiteId
     * @param {Object} config
     * @returns {Promise<Object>}
     */
    async executeTestSuite(suiteId, config = {}) {
        const result = await this.post(`/test-suite/${suiteId}/execute`, config);
        this.eventBus.emit(EVENT_TYPES.SUITE_EXECUTED, { suiteId, result });
        return result;
    }

    // Execution endpoints

    /**
     * Get all test executions
     * @param {Object} params - Query parameters
     * @returns {Promise<Array>}
     */
    async getTestExecutions(params = {}) {
        const query = new URLSearchParams(params).toString();
        const endpoint = query ? `/test-executions?${query}` : '/test-executions';
        return this.get(endpoint);
    }

    /**
     * Get specific test execution
     * @param {string} executionId
     * @returns {Promise<Object>}
     */
    async getTestExecution(executionId) {
        return this.get(`/test-execution/${executionId}/results`);
    }

    /**
     * Delete test execution
     * @param {string} executionId
     * @returns {Promise<Object>}
     */
    async deleteTestExecution(executionId) {
        const result = await this.delete(`/test-execution/${executionId}`);
        this.eventBus.emit(EVENT_TYPES.EXECUTION_DELETED, { executionId });
        return result;
    }

    /**
     * Clear all test executions
     * @returns {Promise<Object>}
     */
    async clearAllExecutions() {
        return this.delete('/test-executions');
    }

    // Coverage endpoints

    /**
     * Get coverage summaries
     * @returns {Promise<Array>}
     */
    async getCoverageSummaries() {
        return this.get('/test-analysis/coverage');
    }

    /**
     * Get coverage analysis for scenario
     * @param {string} scenarioName
     * @returns {Promise<Object>}
     */
    async getCoverageAnalysis(scenarioName) {
        return this.get(`/test-analysis/coverage/${encodeURIComponent(scenarioName)}`);
    }

    /**
     * Generate coverage analysis
     * @param {Object} config
     * @returns {Promise<Object>}
     */
    async generateCoverageAnalysis(config) {
        const result = await this.post('/test-analysis/coverage', config);
        this.eventBus.emit(EVENT_TYPES.COVERAGE_GENERATED, result);
        return result;
    }

    // Vault endpoints

    /**
     * Get all test vaults
     * @returns {Promise<Array>}
     */
    async getTestVaults() {
        return this.get('/test-vaults');
    }

    /**
     * Get specific test vault
     * @param {string} vaultId
     * @returns {Promise<Object>}
     */
    async getTestVault(vaultId) {
        return this.get(`/test-vault/${vaultId}`);
    }

    /**
     * Create test vault
     * @param {Object} config
     * @returns {Promise<Object>}
     */
    async createTestVault(config) {
        const result = await this.post('/test-vault/create', config);
        this.eventBus.emit(EVENT_TYPES.VAULT_CREATED, result);
        return result;
    }

    /**
     * Get vault executions
     * @param {Object} params - Query parameters
     * @returns {Promise<Array>}
     */
    async getVaultExecutions(params = {}) {
        const query = new URLSearchParams(params).toString();
        const endpoint = query ? `/vault-executions?${query}` : '/vault-executions';
        return this.get(endpoint);
    }

    /**
     * Get vault execution results
     * @param {string} executionId
     * @returns {Promise<Object>}
     */
    async getVaultExecutionResults(executionId) {
        return this.get(`/vault-execution/${executionId}/results`);
    }

    // Scenario endpoints

    /**
     * Get all scenarios
     * @returns {Promise<Array>}
     */
    async getScenarios() {
        return this.get('/scenarios');
    }

    // Reports endpoints

    /**
     * Get reports overview
     * @param {number} windowDays
     * @returns {Promise<Object>}
     */
    async getReportsOverview(windowDays = 30) {
        return this.get(`/reports/overview?window_days=${windowDays}`);
    }

    /**
     * Get reports trends
     * @param {number} windowDays
     * @returns {Promise<Object>}
     */
    async getReportsTrends(windowDays = 30) {
        return this.get(`/reports/trends?window_days=${windowDays}`);
    }

    /**
     * Get reports insights
     * @param {number} windowDays
     * @returns {Promise<Object>}
     */
    async getReportsInsights(windowDays = 30) {
        return this.get(`/reports/insights?window_days=${windowDays}`);
    }

    /**
     * Get all reports data
     * @param {number} windowDays
     * @returns {Promise<Object>}
     */
    async getAllReports(windowDays = 30) {
        const [overview, trends, insights] = await Promise.all([
            this.getReportsOverview(windowDays),
            this.getReportsTrends(windowDays),
            this.getReportsInsights(windowDays)
        ]);

        return { overview, trends, insights };
    }

    // Interceptor management

    /**
     * Add request interceptor
     * @param {Function} interceptor - async (url, options) => modifiedOptions
     */
    addRequestInterceptor(interceptor) {
        this.requestInterceptors.push(interceptor);
    }

    /**
     * Add response interceptor
     * @param {Function} interceptor - async (data, response) => modifiedData
     */
    addResponseInterceptor(interceptor) {
        this.responseInterceptors.push(interceptor);
    }

    /**
     * Cancel all in-flight requests
     */
    cancelAllRequests() {
        for (const [requestId, controller] of this.inflightRequests.entries()) {
            controller.abort();
            if (this.debug) {
                console.log(`[ApiClient] Cancelled request: ${requestId}`);
            }
        }
        this.inflightRequests.clear();
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[ApiClient] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            baseUrl: this.baseUrl,
            defaultTimeout: this.defaultTimeout,
            maxRetries: this.maxRetries,
            inflightRequests: this.inflightRequests.size,
            requestInterceptors: this.requestInterceptors.length,
            responseInterceptors: this.responseInterceptors.length
        };
    }

    // Private methods

    /**
     * Check if error is retryable
     * @private
     */
    _isRetryableError(error) {
        // Retry on network errors, timeouts, and 5xx server errors
        return (
            error.message.includes('NetworkError') ||
            error.message.includes('Failed to fetch') ||
            error.message.includes('timeout') ||
            (error.message.match(/HTTP 5\d\d/))
        );
    }

    /**
     * Delay helper for retries
     * @private
     */
    _delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Export singleton instance
export const apiClient = new ApiClient();

// Export default for convenience
export default apiClient;
