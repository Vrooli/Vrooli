// Task Management Module
export class TaskManager {
    constructor(apiBase, showToast, showLoading) {
        this.apiBase = apiBase;
        this.showToast = showToast;
        this.showLoading = showLoading;
        this.categoryOptions = {};
    }

    async loadTasks(status) {
        try {
            const response = await fetch(`${this.apiBase}/tasks?status=${status}`);
            if (!response.ok) {
                if (response.status === 429) {
                    const retryAfter = response.headers.get('Retry-After') || '60';
                    throw { 
                        message: `Rate limit exceeded. Try again in ${retryAfter} seconds.`,
                        isRateLimit: true,
                        retryAfter: parseInt(retryAfter)
                    };
                }
                throw new Error(`Failed to load ${status} tasks: ${response.status} ${response.statusText}`);
            }
            
            const tasks = await response.json();
            return tasks || [];
        } catch (error) {
            if (error.isRateLimit) {
                throw error;
            }
            console.error(`Error loading ${status} tasks:`, error);
            throw error;
        }
    }

    async createTask(taskData) {
        const response = await fetch(`${this.apiBase}/tasks`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(taskData)
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Failed to create task: ${errorText}`);
        }

        return await response.json();
    }

    async updateTask(taskId, updates) {
        const response = await fetch(`${this.apiBase}/tasks/${taskId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(updates)
        });

        if (!response.ok) {
            throw new Error(`Failed to update task: ${response.status} ${response.statusText}`);
        }

        return await response.json();
    }

    async deleteTask(taskId, status) {
        const response = await fetch(`${this.apiBase}/tasks/${taskId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            if (response.status === 404) {
                return { success: true, notFound: true };
            }
            throw new Error(`Failed to delete task: ${response.status} ${response.statusText}`);
        }

        // Check if response has content before parsing JSON
        const text = await response.text();
        if (!text.trim()) {
            // Empty response means successful deletion
            return { success: true };
        }
        
        try {
            return JSON.parse(text);
        } catch (error) {
            // If JSON parsing fails but response was ok, assume success
            return { success: true };
        }
    }

    async getTaskDetails(taskId) {
        const response = await fetch(`${this.apiBase}/tasks/${taskId}`);
        if (!response.ok) {
            throw new Error(`Failed to load task: ${response.status} ${response.statusText}`);
        }
        return await response.json();
    }

    async getTaskPrompt(taskId) {
        const response = await fetch(`${this.apiBase}/tasks/${taskId}/prompt/assembled`);
        if (!response.ok) {
            throw new Error(`Failed to load prompt: ${response.status} ${response.statusText}`);
        }
        return await response.json();
    }

    formatErrorText(errorText) {
        if (!errorText) return '';
        
        // Check if it's an object (JSON)
        if (typeof errorText === 'object') {
            return JSON.stringify(errorText, null, 2);
        }
        
        // Format error text for better readability
        return errorText
            .replace(/\\n/g, '\n')
            .replace(/\\t/g, '  ')
            .replace(/\\/g, '');
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}