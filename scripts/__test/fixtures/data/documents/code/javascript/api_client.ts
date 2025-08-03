/**
 * Vrooli API Client
 * TypeScript test fixture for API integration and type analysis
 */

// Type definitions for API requests and responses
interface ApiResponse<T = any> {
    success: boolean;
    data?: T;
    error?: string;
    timestamp: string;
    requestId: string;
}

interface ResourceConfig {
    id: string;
    name: string;
    type: 'ai' | 'automation' | 'storage' | 'search' | 'agent';
    enabled: boolean;
    baseUrl: string;
    healthCheck?: HealthCheckConfig;
    metadata?: Record<string, any>;
}

interface HealthCheckConfig {
    endpoint?: string;
    interval?: number;
    timeout?: number;
    retries?: number;
}

interface TaskRequest {
    task: string;
    parameters: Record<string, any>;
    timeout?: number;
    priority?: 'low' | 'medium' | 'high';
}

interface TaskResult<T = any> {
    taskId: string;
    status: 'pending' | 'running' | 'completed' | 'failed';
    result?: T;
    error?: string;
    startedAt: string;
    completedAt?: string;
    executionTime?: number;
}

interface PaginatedResponse<T> {
    items: T[];
    pagination: {
        page: number;
        limit: number;
        total: number;
        totalPages: number;
        hasNext: boolean;
        hasPrev: boolean;
    };
}

interface ErrorDetails {
    code: string;
    message: string;
    details?: Record<string, any>;
    stack?: string;
}

// Custom error classes for better error handling
class ApiError extends Error {
    constructor(
        message: string,
        public statusCode: number,
        public errorCode?: string,
        public details?: Record<string, any>
    ) {
        super(message);
        this.name = 'ApiError';
    }
}

class NetworkError extends Error {
    constructor(message: string, public originalError?: Error) {
        super(message);
        this.name = 'NetworkError';
    }
}

class ValidationError extends Error {
    constructor(
        message: string,
        public field?: string,
        public value?: any
    ) {
        super(message);
        this.name = 'ValidationError';
    }
}

// Configuration options for the API client
interface ApiClientConfig {
    baseUrl: string;
    timeout: number;
    retries: number;
    retryDelay: number;
    headers: Record<string, string>;
    interceptors?: {
        request?: (config: RequestInit) => RequestInit | Promise<RequestInit>;
        response?: <T>(response: ApiResponse<T>) => ApiResponse<T> | Promise<ApiResponse<T>>;
        error?: (error: Error) => Error | Promise<Error>;
    };
}

/**
 * Main API client class for Vrooli resource management
 */
export class VrooliApiClient {
    private config: ApiClientConfig;
    private requestId: number = 0;

    constructor(config: Partial<ApiClientConfig> = {}) {
        this.config = {
            baseUrl: 'http://localhost:3000/api',
            timeout: 30000,
            retries: 3,
            retryDelay: 1000,
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json',
                'User-Agent': 'Vrooli-API-Client/1.0.0'
            },
            ...config
        };
    }

    /**
     * Generic HTTP request method with retry logic and error handling
     */
    private async request<T>(
        endpoint: string,
        options: RequestInit = {}
    ): Promise<ApiResponse<T>> {
        const requestId = `req_${++this.requestId}_${Date.now()}`;
        const url = `${this.config.baseUrl}${endpoint}`;
        
        // Apply request interceptor if provided
        let requestOptions = {
            ...options,
            headers: {
                ...this.config.headers,
                ...options.headers,
                'X-Request-ID': requestId
            }
        };

        if (this.config.interceptors?.request) {
            requestOptions = await this.config.interceptors.request(requestOptions);
        }

        let lastError: Error;

        // Retry logic
        for (let attempt = 1; attempt <= this.config.retries; attempt++) {
            try {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

                const response = await fetch(url, {
                    ...requestOptions,
                    signal: controller.signal
                });

                clearTimeout(timeoutId);

                // Parse response
                let responseData: ApiResponse<T>;
                
                if (response.headers.get('content-type')?.includes('application/json')) {
                    responseData = await response.json();
                } else {
                    const text = await response.text();
                    responseData = {
                        success: response.ok,
                        data: text as any,
                        error: response.ok ? undefined : `HTTP ${response.status}: ${response.statusText}`,
                        timestamp: new Date().toISOString(),
                        requestId
                    };
                }

                // Apply response interceptor if provided
                if (this.config.interceptors?.response) {
                    responseData = await this.config.interceptors.response(responseData);
                }

                // Handle HTTP errors
                if (!response.ok) {
                    throw new ApiError(
                        responseData.error || `HTTP ${response.status}: ${response.statusText}`,
                        response.status,
                        responseData.error,
                        responseData.data
                    );
                }

                return responseData;

            } catch (error) {
                lastError = error as Error;
                
                // Apply error interceptor if provided
                if (this.config.interceptors?.error) {
                    lastError = await this.config.interceptors.error(lastError);
                }

                // Don't retry on certain errors
                if (error instanceof ApiError && error.statusCode >= 400 && error.statusCode < 500) {
                    throw error;
                }

                // Log retry attempt
                console.warn(`Request attempt ${attempt} failed:`, error);

                // Wait before retry (exponential backoff)
                if (attempt < this.config.retries) {
                    const delay = this.config.retryDelay * Math.pow(2, attempt - 1);
                    await new Promise(resolve => setTimeout(resolve, delay));
                }
            }
        }

        // If all retries failed, throw the last error
        if (lastError instanceof Error) {
            throw new NetworkError(`All ${this.config.retries} attempts failed`, lastError);
        }

        throw new NetworkError('Unknown network error occurred');
    }

    // Resource Management Methods

    /**
     * Discover available resources
     */
    async discoverResources(): Promise<ApiResponse<{ resources: ResourceConfig[] }>> {
        return this.request<{ resources: ResourceConfig[] }>('/resources/discover');
    }

    /**
     * Get all resources with optional filtering and pagination
     */
    async getResources(params?: {
        type?: string;
        enabled?: boolean;
        page?: number;
        limit?: number;
    }): Promise<ApiResponse<PaginatedResponse<ResourceConfig>>> {
        const queryParams = new URLSearchParams();
        
        if (params) {
            Object.entries(params).forEach(([key, value]) => {
                if (value !== undefined) {
                    queryParams.append(key, value.toString());
                }
            });
        }

        const endpoint = `/resources${queryParams.toString() ? `?${queryParams}` : ''}`;
        return this.request<PaginatedResponse<ResourceConfig>>(endpoint);
    }

    /**
     * Get a specific resource by ID
     */
    async getResource(resourceId: string): Promise<ApiResponse<ResourceConfig>> {
        if (!resourceId) {
            throw new ValidationError('Resource ID is required', 'resourceId', resourceId);
        }

        return this.request<ResourceConfig>(`/resources/${encodeURIComponent(resourceId)}`);
    }

    /**
     * Create or update a resource configuration
     */
    async saveResource(resource: Omit<ResourceConfig, 'id'> & { id?: string }): Promise<ApiResponse<ResourceConfig>> {
        // Validate required fields
        if (!resource.name) {
            throw new ValidationError('Resource name is required', 'name', resource.name);
        }
        if (!resource.type) {
            throw new ValidationError('Resource type is required', 'type', resource.type);
        }
        if (!resource.baseUrl) {
            throw new ValidationError('Resource base URL is required', 'baseUrl', resource.baseUrl);
        }

        const method = resource.id ? 'PUT' : 'POST';
        const endpoint = resource.id ? `/resources/${encodeURIComponent(resource.id)}` : '/resources';

        return this.request<ResourceConfig>(endpoint, {
            method,
            body: JSON.stringify(resource)
        });
    }

    /**
     * Delete a resource
     */
    async deleteResource(resourceId: string): Promise<ApiResponse<{ deleted: boolean }>> {
        if (!resourceId) {
            throw new ValidationError('Resource ID is required', 'resourceId', resourceId);
        }

        return this.request<{ deleted: boolean }>(`/resources/${encodeURIComponent(resourceId)}`, {
            method: 'DELETE'
        });
    }

    // Health Check Methods

    /**
     * Check health of a specific resource
     */
    async checkResourceHealth(resourceId: string): Promise<ApiResponse<{
        healthy: boolean;
        responseTime: number;
        details: Record<string, any>;
    }>> {
        if (!resourceId) {
            throw new ValidationError('Resource ID is required', 'resourceId', resourceId);
        }

        return this.request(`/resources/${encodeURIComponent(resourceId)}/health`);
    }

    /**
     * Check health of all resources
     */
    async checkAllResourcesHealth(): Promise<ApiResponse<{
        overall: boolean;
        resources: Array<{
            id: string;
            healthy: boolean;
            responseTime: number;
            error?: string;
        }>;
    }>> {
        return this.request('/resources/health');
    }

    // Task Execution Methods

    /**
     * Execute a task on a specific resource
     */
    async executeTask<T = any>(
        resourceId: string,
        taskRequest: TaskRequest
    ): Promise<ApiResponse<TaskResult<T>>> {
        if (!resourceId) {
            throw new ValidationError('Resource ID is required', 'resourceId', resourceId);
        }
        if (!taskRequest.task) {
            throw new ValidationError('Task name is required', 'task', taskRequest.task);
        }

        return this.request<TaskResult<T>>(`/resources/${encodeURIComponent(resourceId)}/execute`, {
            method: 'POST',
            body: JSON.stringify(taskRequest)
        });
    }

    /**
     * Get task status and result
     */
    async getTaskStatus(taskId: string): Promise<ApiResponse<TaskResult>> {
        if (!taskId) {
            throw new ValidationError('Task ID is required', 'taskId', taskId);
        }

        return this.request<TaskResult>(`/tasks/${encodeURIComponent(taskId)}`);
    }

    /**
     * Cancel a running task
     */
    async cancelTask(taskId: string): Promise<ApiResponse<{ cancelled: boolean }>> {
        if (!taskId) {
            throw new ValidationError('Task ID is required', 'taskId', taskId);
        }

        return this.request<{ cancelled: boolean }>(`/tasks/${encodeURIComponent(taskId)}/cancel`, {
            method: 'POST'
        });
    }

    /**
     * Get list of tasks with optional filtering
     */
    async getTasks(params?: {
        status?: TaskResult['status'];
        resourceId?: string;
        page?: number;
        limit?: number;
    }): Promise<ApiResponse<PaginatedResponse<TaskResult>>> {
        const queryParams = new URLSearchParams();
        
        if (params) {
            Object.entries(params).forEach(([key, value]) => {
                if (value !== undefined) {
                    queryParams.append(key, value.toString());
                }
            });
        }

        const endpoint = `/tasks${queryParams.toString() ? `?${queryParams}` : ''}`;
        return this.request<PaginatedResponse<TaskResult>>(endpoint);
    }

    // Utility Methods

    /**
     * Test API connectivity
     */
    async ping(): Promise<ApiResponse<{ message: string; timestamp: string }>> {
        return this.request<{ message: string; timestamp: string }>('/ping');
    }

    /**
     * Get API version and build information
     */
    async getVersion(): Promise<ApiResponse<{
        version: string;
        build: string;
        environment: string;
        uptime: number;
    }>> {
        return this.request('/version');
    }

    /**
     * Update client configuration
     */
    updateConfig(newConfig: Partial<ApiClientConfig>): void {
        this.config = { ...this.config, ...newConfig };
    }

    /**
     * Get current client configuration
     */
    getConfig(): Readonly<ApiClientConfig> {
        return { ...this.config };
    }
}

// Convenience factory function
export function createVrooliApiClient(config?: Partial<ApiClientConfig>): VrooliApiClient {
    return new VrooliApiClient(config);
}

// Default client instance
export const defaultApiClient = new VrooliApiClient();

// Export types for external use
export type {
    ApiResponse,
    ResourceConfig,
    HealthCheckConfig,
    TaskRequest,
    TaskResult,
    PaginatedResponse,
    ErrorDetails,
    ApiClientConfig
};

export {
    ApiError,
    NetworkError,
    ValidationError
};