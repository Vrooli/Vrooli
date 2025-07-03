/**
 * HTTP Client Service for External API Calls
 * 
 * Provides a secure, robust HTTP client for making external API calls within the execution engine.
 * Features:
 * - Multiple authentication methods (Bearer, API Key, Basic, Custom)
 * - SSRF protection and URL validation
 * - Request/response sanitization
 * - Built-in retry logic and timeout handling
 * - Comprehensive error handling and logging
 */

import { fetchWrapper, MB_10_BYTES, SECONDS_30_MS, type FetchWrapperOptions } from "@vrooli/shared";
import { logger } from "../../events/logger.js";

// Default configuration constants
const DEFAULT_TIMEOUT_MS = SECONDS_30_MS;
const DEFAULT_RETRIES = 3;
const MAX_BODY_SIZE = MB_10_BYTES;
const FORBIDDEN_HEADERS = ["host", "user-agent", "content-length", "transfer-encoding"];

// SSRF Protection - blocked domains and IP ranges
const BLOCKED_DOMAINS = [
    "localhost",
    "127.0.0.1",
    "0.0.0.0",
    "10.", // Private IP ranges
    "172.16.",
    "192.168.",
    "169.254.", // Link-local
    "::1", // IPv6 localhost
    "fc00::", // IPv6 private
];

export interface APICallOptions {
    url: string;
    method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS";
    headers?: Record<string, string>;
    body?: any;
    auth?: AuthConfig;
    timeout?: number;
    retries?: number;
    validateSSL?: boolean;
}

export interface AuthConfig {
    type: "bearer" | "apikey" | "basic" | "custom";
    token?: string;
    headerName?: string;
    value?: string;
    username?: string;
    password?: string;
}

export interface APIResponse {
    success: boolean;
    status: number;
    statusText: string;
    headers: Record<string, string>;
    data: any;
    error?: string;
    metadata: {
        url: string;
        method: string;
        executionTime: number;
        retries: number;
        finalUrl?: string;
    };
}

export class HTTPClient {
    /**
     * Make an HTTP request to an external API
     */
    async makeRequest(options: APICallOptions): Promise<APIResponse> {
        const startTime = Date.now();
        const retryCount = 0;
        const maxRetries = options.retries ?? DEFAULT_RETRIES;

        // Validate and sanitize inputs
        this.validateUrl(options.url);
        const sanitizedHeaders = this.sanitizeHeaders(options.headers || {});
        this.validateBody(options.body);

        // Build request configuration
        const requestInit: RequestInit = {
            method: options.method,
            headers: {
                "Content-Type": "application/json",
                ...sanitizedHeaders,
            },
        };

        // Build fetchWrapper options
        const fetchOptions: FetchWrapperOptions = {
            timeout: options.timeout || DEFAULT_TIMEOUT_MS,
            maxRetries,
        };

        // Add authentication
        if (options.auth) {
            this.applyAuthentication(requestInit, options.auth);
        }

        // Add body if provided
        if (options.body && !["GET", "HEAD"].includes(options.method)) {
            requestInit.body = typeof options.body === "string"
                ? options.body
                : JSON.stringify(options.body);
        }

        // Log the request (sanitized)
        logger.info("[HTTPClient] Making external API request", {
            url: options.url,
            method: options.method,
            hasAuth: !!options.auth,
            authType: options.auth?.type,
            timeout: fetchOptions.timeout,
        });

        try {
            // Make the request using fetchWrapper
            const response = await fetchWrapper(options.url, requestInit, fetchOptions);

            const executionTime = Date.now() - startTime;

            // Parse response data
            let responseData: any;
            const contentType = response.headers.get("content-type") || "";

            if (contentType.includes("application/json")) {
                try {
                    responseData = await response.json();
                } catch {
                    responseData = await response.text();
                }
            } else {
                responseData = await response.text();
            }

            // Extract response headers
            const responseHeaders: Record<string, string> = {};
            response.headers.forEach((value, key) => {
                responseHeaders[key] = value;
            });

            const result: APIResponse = {
                success: response.ok,
                status: response.status,
                statusText: response.statusText,
                headers: responseHeaders,
                data: responseData,
                error: response.ok ? undefined : `HTTP ${response.status}: ${response.statusText}`,
                metadata: {
                    url: options.url,
                    method: options.method,
                    executionTime,
                    retries: retryCount,
                    finalUrl: response.url,
                },
            };

            // Log the response (sanitized)
            logger.info("[HTTPClient] API request completed", {
                url: options.url,
                status: response.status,
                success: response.ok,
                executionTime,
                retries: retryCount,
            });

            return result;

        } catch (error) {
            const executionTime = Date.now() - startTime;
            const errorMessage = error instanceof Error ? error.message : String(error);

            logger.error("[HTTPClient] API request failed", {
                url: options.url,
                method: options.method,
                error: errorMessage,
                executionTime,
                retries: retryCount,
            });

            return {
                success: false,
                status: 0,
                statusText: "Network Error",
                headers: {},
                data: null,
                error: errorMessage,
                metadata: {
                    url: options.url,
                    method: options.method,
                    executionTime,
                    retries: retryCount,
                },
            };
        }
    }

    /**
     * Apply authentication to the request
     */
    private applyAuthentication(requestInit: RequestInit, auth: AuthConfig): void {
        const headers = new Headers(requestInit.headers);

        switch (auth.type) {
            case "bearer":
                if (auth.token) {
                    headers.set("Authorization", `Bearer ${auth.token}`);
                }
                break;

            case "apikey":
                if (auth.token) {
                    const headerName = auth.headerName || "X-API-Key";
                    headers.set(headerName, auth.token);
                }
                break;

            case "basic":
                if (auth.username && auth.password) {
                    const credentials = Buffer.from(`${auth.username}:${auth.password}`).toString("base64");
                    headers.set("Authorization", `Basic ${credentials}`);
                }
                break;

            case "custom":
                if (auth.headerName && auth.value) {
                    headers.set(auth.headerName, auth.value);
                }
                break;

            default:
                logger.warn("[HTTPClient] Unknown auth type", { authType: auth.type });
        }

        requestInit.headers = headers;
    }

    /**
     * Validate URL to prevent SSRF attacks
     */
    private validateUrl(url: string): void {
        try {
            const urlObj = new URL(url);

            // Only allow HTTP and HTTPS
            if (!["http:", "https:"].includes(urlObj.protocol)) {
                throw new Error(`Invalid protocol: ${urlObj.protocol}. Only HTTP and HTTPS are allowed.`);
            }

            // Check against blocked domains/IPs
            const hostname = urlObj.hostname.toLowerCase();
            for (const blocked of BLOCKED_DOMAINS) {
                if (hostname === blocked || hostname.startsWith(blocked)) {
                    throw new Error(`Blocked domain/IP: ${hostname}`);
                }
            }

            // Validate port (no unusual ports)
            const port = urlObj.port;
            if (port && !["80", "443", "8080", "8443"].includes(port)) {
                logger.warn("[HTTPClient] Unusual port detected", { url, port });
            }

        } catch (error) {
            if (error instanceof Error) {
                throw new Error(`Invalid URL: ${error.message}`);
            }
            throw new Error(`Invalid URL format: ${url}`);
        }
    }

    /**
     * Sanitize and validate headers
     */
    private sanitizeHeaders(headers: Record<string, string>): Record<string, string> {
        const sanitized: Record<string, string> = {};

        for (const [key, value] of Object.entries(headers)) {
            const lowerKey = key.toLowerCase();

            // Block dangerous headers
            if (FORBIDDEN_HEADERS.includes(lowerKey)) {
                logger.warn("[HTTPClient] Blocked forbidden header", { header: key });
                continue;
            }

            // Sanitize header values (remove control characters)
            const sanitizedValue = String(value).replace(/[\p{Cc}\u007F]/gu, "");
            if (sanitizedValue.length > 0) {
                sanitized[key] = sanitizedValue;
            }
        }

        return sanitized;
    }

    /**
     * Validate request body size
     */
    private validateBody(body: any): void {
        if (!body) return;

        const bodyString = typeof body === "string" ? body : JSON.stringify(body);
        const bodySize = Buffer.byteLength(bodyString, "utf8");

        if (bodySize > MAX_BODY_SIZE) {
            throw new Error(`Request body too large: ${bodySize} bytes (max: ${MAX_BODY_SIZE})`);
        }
    }
}
