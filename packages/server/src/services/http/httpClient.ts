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

// Profile system for different security contexts
export interface HTTPProfile {
    name: string;
    allowLocalhost: boolean;
    allowPrivateIPs: boolean;
    requireSSL: boolean;
    blockedDomains: string[];
    defaultTimeout: number;
    defaultRetries: number;
    forbiddenHeaders: string[];
}

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
    // Profile definitions for different security contexts
    static readonly PROFILES: Record<string, HTTPProfile> = {
        external: {
            name: "external",
            allowLocalhost: false,
            allowPrivateIPs: false,
            requireSSL: false, // Allow both HTTP and HTTPS for external APIs
            blockedDomains: BLOCKED_DOMAINS,
            defaultTimeout: DEFAULT_TIMEOUT_MS,
            defaultRetries: DEFAULT_RETRIES,
            forbiddenHeaders: FORBIDDEN_HEADERS,
        },
        resource: {
            name: "resource",
            allowLocalhost: true,
            allowPrivateIPs: true,
            requireSSL: false,
            blockedDomains: [], // Resources can access anything
            defaultTimeout: 5000, // Faster timeout for local services
            defaultRetries: 5, // More retries for local service discovery
            forbiddenHeaders: ["host", "content-length", "transfer-encoding"], // Allow user-agent
        },
        internal: {
            name: "internal",
            allowLocalhost: true,
            allowPrivateIPs: true,
            requireSSL: false,
            blockedDomains: ["0.0.0.0"], // Only block 0.0.0.0
            defaultTimeout: 10000,
            defaultRetries: 2,
            forbiddenHeaders: ["content-length", "transfer-encoding"],
        },
    };

    // Instance property for active profile
    private profile: HTTPProfile;

    // Constructor to accept profile
    constructor(profile: keyof typeof HTTPClient.PROFILES = "external") {
        this.profile = HTTPClient.PROFILES[profile] || HTTPClient.PROFILES.external;
    }

    /**
     * Make an HTTP request to an external API
     */
    async makeRequest(options: APICallOptions): Promise<APIResponse> {
        const startTime = Date.now();
        const retryCount = 0;
        // Use profile defaults if not specified
        const maxRetries = options.retries ?? this.profile.defaultRetries;

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
            timeout: options.timeout || this.profile.defaultTimeout,
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
        logger.info("[HTTPClient] Making API request", {
            url: options.url,
            method: options.method,
            profile: this.profile.name,
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
     * Validate URL to prevent SSRF attacks based on profile
     */
    private validateUrl(url: string): void {
        try {
            const urlObj = new URL(url);

            // Check protocol requirements
            if (this.profile.requireSSL && urlObj.protocol === "http:") {
                throw new Error("HTTPS required for this profile");
            }

            if (!["http:", "https:"].includes(urlObj.protocol)) {
                throw new Error(`Invalid protocol: ${urlObj.protocol}. Only HTTP and HTTPS are allowed.`);
            }

            const hostname = urlObj.hostname.toLowerCase();
            
            // Check if localhost is allowed
            if (!this.profile.allowLocalhost) {
                if (hostname === "localhost" || hostname === "127.0.0.1" || hostname === "::1") {
                    throw new Error("Localhost access not allowed in this profile");
                }
            }

            // Check against profile's blocked domains
            for (const blocked of this.profile.blockedDomains) {
                if (hostname === blocked || hostname.startsWith(blocked)) {
                    throw new Error(`Blocked domain/IP: ${hostname}`);
                }
            }

            // Check private IPs if not allowed
            if (!this.profile.allowPrivateIPs) {
                if (this.isPrivateIP(hostname)) {
                    throw new Error("Private IP access not allowed in this profile");
                }
            }

            // Port validation remains the same for all profiles
            const port = urlObj.port;
            if (port && !["80", "443", "8080", "8443", "11434", "8081", "5001"].includes(port)) {
                logger.warn("[HTTPClient] Unusual port detected", { url, port, profile: this.profile.name });
            }

        } catch (error) {
            if (error instanceof Error) {
                throw new Error(`Invalid URL: ${error.message}`);
            }
            throw new Error(`Invalid URL format: ${url}`);
        }
    }

    /**
     * Check if a hostname is a private IP address
     */
    private isPrivateIP(hostname: string): boolean {
        const privateRanges = ["10.", "172.16.", "172.17.", "172.18.", "172.19.", "172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.", "172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.", "192.168.", "169.254."];
        return privateRanges.some(range => hostname.startsWith(range));
    }

    /**
     * Sanitize and validate headers
     */
    private sanitizeHeaders(headers: Record<string, string>): Record<string, string> {
        const sanitized: Record<string, string> = {};

        for (const [key, value] of Object.entries(headers)) {
            const lowerKey = key.toLowerCase();

            // Use profile's forbidden headers
            if (this.profile.forbiddenHeaders.includes(lowerKey)) {
                logger.warn("[HTTPClient] Blocked forbidden header", { 
                    header: key, 
                    profile: this.profile.name, 
                });
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

    /**
     * Factory method for external API calls (default security)
     */
    static forExternalAPIs(): HTTPClient {
        return new HTTPClient("external");
    }

    /**
     * Factory method for local resource access
     */
    static forResources(): HTTPClient {
        return new HTTPClient("resource");
    }

    /**
     * Factory method for internal service communication
     */
    static forInternal(): HTTPClient {
        return new HTTPClient("internal");
    }
}
