/**
 * ApiClient - Centralized API communication for Ecosystem Manager
 *
 * This module handles all HTTP requests to the backend API,
 * providing a clean interface for data fetching and mutations.
 *
 * MIGRATION NOTE: This module extracts API logic from app.js.
 * Future work: Migrate all direct fetch() calls in app.js to use this.api
 * Example: Replace `fetch(\`\${this.apiBase}/tasks\`)` with `this.api.getTasks()`
 *
 * Benefits:
 * - Consistent error handling
 * - Type-safe API methods
 * - Easier to mock for testing
 * - Single source of truth for API endpoints
 */

export class ApiClient {
    constructor(apiBase = '/api') {
        this.apiBase = apiBase;
    }

    /**
     * Generic fetch wrapper with error handling
     */
    async fetchJSON(url, options = {}) {
        const response = await fetch(url, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`API Error (${response.status}): ${errorText}`);
        }

        return response.json();
    }

    // ==================== Task Management ====================

    /**
     * Fetch all tasks with optional filters
     */
    async getTasks(filters = {}) {
        const params = new URLSearchParams();

        if (filters.status) params.append('status', filters.status);
        if (filters.type) params.append('type', filters.type);
        if (filters.operation) params.append('operation', filters.operation);
        if (filters.category) params.append('category', filters.category);
        if (filters.priority) params.append('priority', filters.priority);

        const queryString = params.toString();
        const url = `${this.apiBase}/tasks${queryString ? '?' + queryString : ''}`;

        return this.fetchJSON(url);
    }

    /**
     * Get a specific task by ID
     */
    async getTask(taskId) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}`);
    }

    /**
     * Create a new task
     */
    async createTask(taskData) {
        return this.fetchJSON(`${this.apiBase}/tasks`, {
            method: 'POST',
            body: JSON.stringify(taskData)
        });
    }

    /**
     * Update a task
     */
    async updateTask(taskId, updates) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}`, {
            method: 'PUT',
            body: JSON.stringify(updates)
        });
    }

    /**
     * Update task status
     */
    async updateTaskStatus(taskId, status, additionalData = {}) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}/status`, {
            method: 'PUT',
            body: JSON.stringify({ status, ...additionalData })
        });
    }

    /**
     * Delete a task
     */
    async deleteTask(taskId) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}`, {
            method: 'DELETE'
        });
    }

    /**
     * Get task logs
     */
    async getTaskLogs(taskId) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}/logs`);
    }

    /**
     * Get task prompt configuration
     */
    async getTaskPrompt(taskId) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}/prompt`);
    }

    /**
     * Get assembled prompt for a task
     */
    async getAssembledPrompt(taskId) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}/prompt/assembled`);
    }

    /**
     * Get active targets (tasks in progress)
     */
    async getActiveTargets(type, operation) {
        const params = new URLSearchParams();
        if (type) params.append('type', type);
        if (operation) params.append('operation', operation);

        return this.fetchJSON(`${this.apiBase}/tasks/active-targets?${params.toString()}`);
    }

    // ==================== Queue Management ====================

    /**
     * Get queue processor status
     */
    async getQueueStatus() {
        return this.fetchJSON(`${this.apiBase}/queue/status`);
    }

    /**
     * Trigger manual queue processing
     */
    async triggerQueueProcessing() {
        return this.fetchJSON(`${this.apiBase}/queue/trigger`, {
            method: 'POST'
        });
    }

    /**
     * Start queue processor
     */
    async startQueueProcessor() {
        return this.fetchJSON(`${this.apiBase}/queue/start`, {
            method: 'POST'
        });
    }

    /**
     * Stop queue processor
     */
    async stopQueueProcessor() {
        return this.fetchJSON(`${this.apiBase}/queue/stop`, {
            method: 'POST'
        });
    }

    /**
     * Reset rate limit pause
     */
    async resetRateLimit() {
        return this.fetchJSON(`${this.apiBase}/queue/reset-rate-limit`, {
            method: 'POST'
        });
    }

    /**
     * Get running processes
     */
    async getRunningProcesses() {
        return this.fetchJSON(`${this.apiBase}/processes/running`);
    }

    /**
     * Terminate a specific process
     */
    async terminateProcess(taskId) {
        return this.fetchJSON(`${this.apiBase}/queue/processes/terminate`, {
            method: 'POST',
            body: JSON.stringify({ task_id: taskId })
        });
    }

    // ==================== Settings Management ====================

    /**
     * Get current settings
     */
    async getSettings() {
        return this.fetchJSON(`${this.apiBase}/settings`);
    }

    /**
     * Update settings
     */
    async updateSettings(settings) {
        return this.fetchJSON(`${this.apiBase}/settings`, {
            method: 'PUT',
            body: JSON.stringify(settings)
        });
    }

    /**
     * Reset settings to defaults
     */
    async resetSettings() {
        return this.fetchJSON(`${this.apiBase}/settings/reset`, {
            method: 'POST'
        });
    }

    /**
     * Get available recycler models
     */
    async getRecyclerModels(provider) {
        const params = new URLSearchParams({ provider });
        return this.fetchJSON(`${this.apiBase}/settings/recycler/models?${params.toString()}`);
    }

    // ==================== Discovery ====================

    /**
     * Get available resources
     */
    async getResources() {
        return this.fetchJSON(`${this.apiBase}/resources`);
    }

    /**
     * Get available scenarios
     */
    async getScenarios() {
        return this.fetchJSON(`${this.apiBase}/scenarios`);
    }

    /**
     * Get resource status
     */
    async getResourceStatus(resourceName) {
        return this.fetchJSON(`${this.apiBase}/resources/${resourceName}/status`);
    }

    /**
     * Get scenario status
     */
    async getScenarioStatus(scenarioName) {
        return this.fetchJSON(`${this.apiBase}/scenarios/${scenarioName}/status`);
    }

    /**
     * Get available operations
     */
    async getOperations() {
        return this.fetchJSON(`${this.apiBase}/operations`);
    }

    /**
     * Get available categories
     */
    async getCategories() {
        return this.fetchJSON(`${this.apiBase}/categories`);
    }

    // ==================== Logs ====================

    /**
     * Get system logs
     */
    async getSystemLogs(limit = 500) {
        return this.fetchJSON(`${this.apiBase}/logs?limit=${limit}`);
    }

    // ==================== Execution History ====================

    /**
     * Get execution history for ALL tasks
     */
    async getAllExecutionHistory() {
        return this.fetchJSON(`${this.apiBase}/executions`);
    }

    /**
     * Get execution history for a specific task
     */
    async getExecutionHistory(taskId) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}/executions`);
    }

    /**
     * Get execution prompt file
     */
    async getExecutionPrompt(taskId, executionId) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}/executions/${executionId}/prompt`);
    }

    /**
     * Get execution output file
     */
    async getExecutionOutput(taskId, executionId) {
        return this.fetchJSON(`${this.apiBase}/tasks/${taskId}/executions/${executionId}/output`);
    }

    // ==================== Recycler & Testing ====================

    /**
     * Test recycler with custom output
     */
    async testRecycler(payload) {
        return this.fetchJSON(`${this.apiBase}/recycler/test`, {
            method: 'POST',
            body: JSON.stringify(payload)
        });
    }

    /**
     * Get recycler prompt preview
     */
    async getRecyclerPromptPreview(outputText) {
        return this.fetchJSON(`${this.apiBase}/recycler/preview-prompt`, {
            method: 'POST',
            body: JSON.stringify({ output_text: outputText })
        });
    }

    /**
     * Get prompt viewer/preview
     */
    async getPromptPreview(taskConfig) {
        return this.fetchJSON(`${this.apiBase}/prompt-viewer`, {
            method: 'POST',
            body: JSON.stringify(taskConfig)
        });
    }

    // ==================== Maintenance ====================

    /**
     * Set maintenance state
     */
    async setMaintenanceState(active) {
        return this.fetchJSON(`${this.apiBase}/maintenance/state`, {
            method: 'POST',
            body: JSON.stringify({ active })
        });
    }
}
