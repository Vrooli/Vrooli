import { SERVER_VERSION, type EndpointDefinition } from "@vrooli/shared";
import axios, { type AxiosError, type AxiosInstance, type AxiosRequestConfig } from "axios";
import FormData from "form-data";
import { io, type Socket } from "socket.io-client";
import { type ConfigManager } from "./config.js";
import { HTTP_STATUS, LIMITS, WEBSOCKET_CONFIG } from "./constants.js";
import { CLI_CONFIG } from "../config/constants.js";
import { formatErrorWithTrace } from "./errorMessages.js";
import { logger } from "./logger.js";

export interface ApiError {
    message: string;
    code?: string;
    details?: unknown;
}

export class ApiClient {
    private axios: AxiosInstance;
    private socket: Socket | null = null;
    private reconnectAttempts = 0;
    private maxReconnectAttempts = WEBSOCKET_CONFIG.MAX_RECONNECT_ATTEMPTS;
    private reconnectTimeout: NodeJS.Timeout | null = null;
    private refreshAttempts = 0;
    private readonly maxRefreshAttempts = 3;
    private refreshPromise: Promise<void> | null = null;

    constructor(private config: ConfigManager) {
        this.axios = axios.create({
            baseURL: `${config.getServerUrl()}/api/v2`,
            timeout: CLI_CONFIG.API_TIMEOUT,
            headers: {
                "Content-Type": "application/json",
            },
        });

        // Request interceptor to add auth token
        this.axios.interceptors.request.use(
            (config) => {
                const token = this.config.getAuthToken();
                if (token) {
                    config.headers.Authorization = `Bearer ${token}`;
                }

                if (this.config.isDebug()) {
                    logger.debug(`API Request: ${config.method?.toUpperCase()} ${config.url}`, {
                        headers: config.headers,
                        data: config.data,
                    });
                }

                return config;
            },
            (error) => {
                logger.error("Request interceptor error", error);
                return Promise.reject(error);
            },
        );

        // Response interceptor for error handling and token refresh
        this.axios.interceptors.response.use(
            (response) => {
                if (this.config.isDebug()) {
                    logger.debug(`API Response: ${response.status} ${response.config.url}`, {
                        data: response.data,
                    });
                }
                return response;
            },
            async (error: AxiosError) => {
                if (this.config.isDebug()) {
                    logger.debug(`API Error: ${error.response?.status} ${error.config?.url}`, {
                        data: error.response?.data,
                    });
                }

                // Handle 401 - try to refresh token (with max attempts)
                if (error.response?.status === HTTP_STATUS.UNAUTHORIZED && error.config) {
                    const requestConfig = error.config as AxiosRequestConfig & { _retry?: boolean; _refreshAttempt?: number };
                    
                    // Initialize refresh attempt counter for this request
                    if (!requestConfig._refreshAttempt) {
                        requestConfig._refreshAttempt = 0;
                    }
                    
                    // Check if we haven't exceeded max refresh attempts for this request
                    if (!requestConfig._retry && requestConfig._refreshAttempt < this.maxRefreshAttempts) {
                        requestConfig._retry = true;
                        requestConfig._refreshAttempt++;
                        
                        try {
                            // Use the singleton refresh promise to prevent concurrent refreshes
                            await this.performTokenRefresh();
                            // Reset global refresh attempts on success
                            this.refreshAttempts = 0;
                            // Retry the original request with new token
                            const token = this.config.getAuthToken();
                            if (token && error.config) {
                                error.config.headers = error.config.headers || {};
                                error.config.headers.Authorization = `Bearer ${token}`;
                            }
                            return this.axios.request(error.config);
                        } catch (refreshError) {
                            this.refreshAttempts++;
                            
                            // If we've hit the global max attempts, clear auth
                            if (this.refreshAttempts >= this.maxRefreshAttempts) {
                                this.config.clearAuth();
                                this.refreshAttempts = 0; // Reset for next session
                                logger.error(`Token refresh failed after ${this.maxRefreshAttempts} attempts, clearing auth`);
                            }
                            
                            throw this.formatError(error);
                        }
                    }
                }

                throw this.formatError(error);
            },
        );
    }

    private formatError(error: AxiosError): Error {
        if (error.response) {
            // Handle the server's error response structure
            const data = error.response.data as {
                errors?: Array<{ trace?: string; code?: string }>;
                message?: string;
                error?: string;
                code?: string;
                details?: unknown
            };

            // Extract error information from the errors array if present
            let errorCode: string | undefined;
            let errorTrace: string | undefined;
            let fallbackMessage = error.message;

            if (data?.errors && Array.isArray(data.errors) && data.errors.length > 0) {
                const firstError = data.errors[0];
                errorCode = firstError.code;
                errorTrace = firstError.trace;
            } else {
                // Fallback to other error formats
                errorCode = data?.code;
                fallbackMessage = data?.message || data?.error || error.message;
            }

            // Create error with formatted message
            const formattedMessage = formatErrorWithTrace(errorCode, errorTrace, fallbackMessage);
            const err = new Error(formattedMessage);
            (err as ApiError).code = errorCode || error.code;
            (err as ApiError).details = data?.details || data;
            return err;
        } else if (error.request) {
            // Use our error message for connection failures
            return new Error(formatErrorWithTrace("CannotConnectToServer", undefined));
        }
        return error;
    }

    private async performTokenRefresh(): Promise<void> {
        // If a refresh is already in progress, wait for it
        if (this.refreshPromise) {
            return this.refreshPromise;
        }

        // Start a new refresh
        this.refreshPromise = this.refreshAuth()
            .finally(() => {
                // Clear the promise after completion (success or failure)
                this.refreshPromise = null;
            });

        return this.refreshPromise;
    }

    private async refreshAuth(): Promise<void> {
        const refreshToken = this.config.getRefreshToken();
        if (!refreshToken) {
            throw new Error("No refresh token available");
        }

        try {
            const response = await this.axios.post("/auth/refresh", {
                refreshToken,
            });

            const { accessToken, refreshToken: newRefreshToken, expiresIn } = response.data;
            this.config.setAuth(accessToken, newRefreshToken, expiresIn);
        } catch (error) {
            logger.error("Token refresh failed", error);
            throw error;
        }
    }

    // HTTP methods
    public async get<T = unknown>(path: string, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.get<T>(path, config);
        return response.data;
    }

    public async post<T = unknown>(path: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.post<T>(path, data, config);
        return response.data;
    }

    public async put<T = unknown>(path: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.put<T>(path, data, config);
        return response.data;
    }

    public async delete<T = unknown>(path: string, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.delete<T>(path, config);
        return response.data;
    }

    public async patch<T = unknown>(path: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.patch<T>(path, data, config);
        return response.data;
    }

    // WebSocket connection with automatic reconnection
    public connectWebSocket(): Socket {
        // Clear any existing reconnection timeout FIRST
        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
        }
        
        // If already connected, return existing socket
        if (this.socket && this.socket.connected) {
            return this.socket;
        }
        
        // ALWAYS clean up old socket completely before creating new one
        if (this.socket) {
            this.socket.removeAllListeners();
            this.socket.disconnect();
            this.socket = null;
        }

        const token = this.config.getAuthToken();
        this.socket = io(this.config.getServerUrl(), {
            auth: {
                token,
            },
            transports: ["websocket"],
            autoConnect: true,
            reconnection: false, // We'll handle reconnection manually for better control
            timeout: WEBSOCKET_CONFIG.CONNECTION_TIMEOUT_MS, // 10 second connection timeout
        });

        this.socket.on("connect", () => {
            // Clear any pending reconnection timeout on successful connection
            if (this.reconnectTimeout) {
                clearTimeout(this.reconnectTimeout);
                this.reconnectTimeout = null;
            }
            this.reconnectAttempts = 0; // Reset reconnection counter on successful connection
            if (this.config.isDebug()) {
                logger.debug("WebSocket connected");
            }
        });

        this.socket.on("disconnect", (reason) => {
            if (this.config.isDebug()) {
                logger.debug("WebSocket disconnected", { reason });
            }
            
            // Clear any pending reconnection timeout to prevent duplicates
            if (this.reconnectTimeout) {
                clearTimeout(this.reconnectTimeout);
                this.reconnectTimeout = null;
            }
            
            // Only attempt reconnection for certain disconnect reasons
            if (this.shouldReconnect(reason)) {
                this.scheduleReconnection();
            }
        });

        this.socket.on("connect_error", (error) => {
            logger.error("WebSocket connection error", error);
            
            // Clear any pending reconnection timeout to prevent duplicates
            if (this.reconnectTimeout) {
                clearTimeout(this.reconnectTimeout);
                this.reconnectTimeout = null;
            }
            
            // Only schedule reconnection if not already scheduled
            if (!this.reconnectTimeout) {
                this.scheduleReconnection();
            }
        });

        this.socket.on("error", (error) => {
            logger.error("WebSocket error", error);
        });

        return this.socket;
    }

    private shouldReconnect(reason: string): boolean {
        // Don't reconnect for these reasons
        const noReconnectReasons = [
            "io server disconnect", // Server initiated disconnect
            "io client disconnect", // Client initiated disconnect
            "ping timeout", // May indicate server issues
        ];
        
        return !noReconnectReasons.includes(reason) && this.reconnectAttempts < this.maxReconnectAttempts;
    }

    private scheduleReconnection(): void {
        // Prevent concurrent reconnection attempts
        if (this.reconnectTimeout) {
            return; // Already scheduled
        }
        
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            logger.error(`WebSocket max reconnection attempts (${this.maxReconnectAttempts}) exceeded`);
            // Clean up socket completely on max attempts
            if (this.socket) {
                this.socket.removeAllListeners();
                this.socket.disconnect();
                this.socket = null;
            }
            return;
        }

        // Exponential backoff: 1s, 2s, 4s, 8s, 16s
        const delay = Math.min(
            WEBSOCKET_CONFIG.BASE_RECONNECT_DELAY_MS * Math.pow(2, this.reconnectAttempts),
            WEBSOCKET_CONFIG.MAX_RECONNECT_DELAY_MS,
        );
        this.reconnectAttempts++;

        if (this.config.isDebug()) {
            logger.debug(`Scheduling WebSocket reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);
        }

        this.reconnectTimeout = setTimeout(() => {
            this.reconnectTimeout = null;
            // Double-check we're not connected before attempting reconnection
            if (!this.socket?.connected) {
                if (this.config.isDebug()) {
                    logger.debug(`Attempting WebSocket reconnection (attempt ${this.reconnectAttempts})`);
                }
                try {
                    this.connectWebSocket();
                } catch (error) {
                    logger.error("Failed to reconnect WebSocket", error);
                    // Schedule another attempt if we haven't exceeded max attempts
                    if (this.reconnectAttempts < this.maxReconnectAttempts) {
                        this.scheduleReconnection();
                    }
                }
            }
        }, delay);
    }

    public disconnectWebSocket(): void {
        // Clear any scheduled reconnection
        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
        }
        
        // Reset reconnection attempts
        this.reconnectAttempts = 0;
        
        if (this.socket) {
            // Remove all listeners to prevent memory leaks
            this.socket.removeAllListeners();
            this.socket.disconnect();
            this.socket = null;
        }
    }

    // Utility method for handling paginated results
    public async *paginate<T>(
        path: string,
        params: Record<string, unknown> = {},
        pageSize = LIMITS.DEFAULT_PAGE_SIZE,
    ): AsyncGenerator<T[], void, unknown> {
        let page = 0;
        let hasMore = true;

        while (hasMore) {
            const response = await this.get<{ items: T[]; total: number }>(path, {
                params: {
                    ...params,
                    limit: pageSize,
                    offset: page * pageSize,
                },
            });

            yield response.items;

            hasMore = response.items.length === pageSize;
            page++;
        }
    }

    // Upload file with progress
    public async uploadFile(
        path: string,
        file: Buffer | NodeJS.ReadableStream,
        filename: string,
        onProgress?: (percent: number) => void,
    ): Promise<unknown> {
        const form = new FormData();
        form.append("file", file, filename);

        return this.post(path, form, {
            headers: {
                ...form.getHeaders(),
            },
            onUploadProgress: (progressEvent) => {
                if (onProgress && progressEvent.total) {
                    const percentCompleted = Math.round(
                        (progressEvent.loaded * 100) / progressEvent.total,
                    );
                    onProgress(percentCompleted);
                }
            },
        });
    }

    // GraphQL-style request method for API endpoints
    public async request<T = unknown>(
        operationName: string,
        variables: unknown,
        config?: AxiosRequestConfig,
    ): Promise<T> {
        // Convert operation name to REST endpoint path
        // e.g., "routine_findMany" -> "/routine/findMany"  
        const endpoint = operationName.replace(/_/g, "/");

        // Send as POST request with variables as body
        const response = await this.axios.post<{ data: T } | T>(`/api/${endpoint}`, variables, config);

        // Return just the data portion
        if (response.data && typeof response.data === "object" && "data" in response.data) {
            return (response.data as { data: T }).data;
        }
        return response.data as T;
    }

    /**
     * Execute a request using an endpoint definition from pairs.ts
     */
    public async requestWithEndpoint<T = unknown>(
        endpointDef: EndpointDefinition,
        variables?: Record<string, unknown>,
        config?: AxiosRequestConfig,
    ): Promise<T> {
        // Replace path parameters (e.g., :id, :publicId) with values from variables
        let path = endpointDef.endpoint;
        const pathParams: Record<string, unknown> = {};
        const queryParams: Record<string, unknown> = {};

        if (variables) {
            // Extract path parameters
            const paramMatches = path.matchAll(/:([\w]+)/g);
            for (const match of paramMatches) {
                const paramName = match[1];
                if (paramName in variables) {
                    path = path.replace(`:${paramName}`, String(variables[paramName]));
                    pathParams[paramName] = variables[paramName];
                }
            }

            // Remaining variables are query params for GET or body for POST/PUT/DELETE
            Object.entries(variables).forEach(([key, value]) => {
                if (!(key in pathParams)) {
                    queryParams[key] = value;
                }
            });
        }

        // Add /api/v2 prefix if not present
        if (!path.startsWith("/api")) {
            path = `/api/${SERVER_VERSION}${path}`;
        }

        // Execute request based on method
        const method = endpointDef.method.toLowerCase() as "get" | "post" | "put" | "delete";
        const requestConfig: AxiosRequestConfig = {
            ...config,
            method,
            url: path,
        };

        if (method === "get") {
            requestConfig.params = queryParams;
        } else {
            requestConfig.data = queryParams;
        }

        const response = await this.axios.request<T>(requestConfig);
        return response.data;
    }
}
