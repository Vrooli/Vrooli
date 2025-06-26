import axios, { AxiosInstance } from "axios";
import type { AxiosError, AxiosRequestConfig } from "./types.js";
import { io, Socket } from "socket.io-client";
import { ConfigManager } from "./config.js";
import { logger } from "./logger.js";
import chalk from "chalk";
import FormData from "form-data";

export interface ApiError {
    message: string;
    code?: string;
    details?: any;
}

export class ApiClient {
    private axios: AxiosInstance;
    private socket: Socket | null = null;

    constructor(private config: ConfigManager) {
        this.axios = axios.create({
            baseURL: config.getServerUrl(),
            timeout: 30000,
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

                // Handle 401 - try to refresh token
                if (error.response?.status === 401 && error.config && !error.config._retry) {
                    error.config._retry = true;

                    try {
                        await this.refreshAuth();
                        // Retry the original request
                        return this.axios.request(error.config);
                    } catch (refreshError) {
                        // Refresh failed, clear auth
                        this.config.clearAuth();
                        throw this.formatError(error);
                    }
                }

                throw this.formatError(error);
            },
        );
    }

    private formatError(error: AxiosError): Error {
        if (error.response) {
            const data = error.response.data as any;
            const message = data?.message || data?.error || error.message;
            const err = new Error(message);
            (err as any).code = data?.code || error.code;
            (err as any).details = data?.details || data;
            return err;
        } else if (error.request) {
            return new Error("No response from server. Is the server running?");
        }
        return error;
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
    public async get<T = any>(path: string, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.get<T>(path, config);
        return response.data;
    }

    public async post<T = any>(path: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.post<T>(path, data, config);
        return response.data;
    }

    public async put<T = any>(path: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.put<T>(path, data, config);
        return response.data;
    }

    public async delete<T = any>(path: string, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.delete<T>(path, config);
        return response.data;
    }

    public async patch<T = any>(path: string, data?: any, config?: AxiosRequestConfig): Promise<T> {
        const response = await this.axios.patch<T>(path, data, config);
        return response.data;
    }

    // WebSocket connection
    public connectWebSocket(): Socket {
        if (this.socket && this.socket.connected) {
            return this.socket;
        }

        const token = this.config.getAuthToken();
        this.socket = io(this.config.getServerUrl(), {
            auth: {
                token,
            },
            transports: ["websocket"],
        });

        this.socket.on("connect", () => {
            if (this.config.isDebug()) {
                logger.debug("WebSocket connected");
            }
        });

        this.socket.on("disconnect", (reason) => {
            if (this.config.isDebug()) {
                logger.debug("WebSocket disconnected", { reason });
            }
        });

        this.socket.on("error", (error) => {
            logger.error("WebSocket error", error);
        });

        return this.socket;
    }

    public disconnectWebSocket(): void {
        if (this.socket) {
            this.socket.disconnect();
            this.socket = null;
        }
    }

    // Utility method for handling paginated results
    public async *paginate<T>(
        path: string,
        params: Record<string, any> = {},
        pageSize: number = 50,
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
    ): Promise<any> {
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
    public async request<T = any>(
        operationName: string,
        variables: any,
        config?: AxiosRequestConfig,
    ): Promise<T> {
        // Convert operation name to REST endpoint path
        // e.g., "routine_findMany" -> "/routine/findMany"  
        const endpoint = operationName.replace(/_/g, '/');
        
        // Send as POST request with variables as body
        const response = await this.axios.post<{ data: T }>(`/api/${endpoint}`, variables, config);
        
        // Return just the data portion
        return response.data.data || response.data;
    }
}