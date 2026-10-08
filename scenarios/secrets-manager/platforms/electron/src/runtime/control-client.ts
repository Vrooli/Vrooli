/**
 * Runtime Control Client
 *
 * DOC: docs/internal/SEAMS.md#runtime-control-client
 *
 * HTTP client for the bundled runtime control API.
 * Handles authentication, request formatting, and response parsing.
 */

import type {
    IRuntimeControlClient,
    IRuntimeHttpClient,
    IRuntimeFileSystem,
    ITimer,
    RuntimeControlConfig,
    RuntimeRequestOptions,
    RuntimeReadyResponse,
    RuntimePortsResponse,
    RuntimeTelemetryResponse,
    RuntimeDiagnostics,
    RuntimeValidationResponse,
} from "./types";

/**
 * Create a runtime control client with injected dependencies.
 */
export function createRuntimeControlClient(
    http: IRuntimeHttpClient,
    fs: IRuntimeFileSystem,
    timer: ITimer,
    config: RuntimeControlConfig
): IRuntimeControlClient {
    /**
     * Read the auth token from disk.
     */
    async function readToken(): Promise<string> {
        if (!config.tokenPath) return "";
        try {
            return (await fs.readFile(config.tokenPath, "utf-8")).trim();
        } catch {
            // Token may not exist yet during startup
            return "";
        }
    }

    /**
     * Build the full URL for an endpoint.
     */
    function buildUrl(endpoint: string): string {
        return `http://${config.host}:${config.port}${endpoint}`;
    }

    const client: IRuntimeControlClient = {
        async request<T>(endpoint: string, opts?: RuntimeRequestOptions): Promise<T | string> {
            if (!config.enabled) {
                throw new Error("runtime control not enabled");
            }

            const token = await readToken();
            const url = buildUrl(endpoint);
            const headers: Record<string, string> = {};

            // Add auth header for non-health endpoints
            if (token && endpoint !== "/healthz") {
                headers.Authorization = `Bearer ${token}`;
            }

            // Build request options
            const requestOpts: { method: string; headers: Record<string, string>; body?: string } = {
                method: opts?.method ?? "GET",
                headers,
            };

            // Add content type and body for POST requests
            if (opts?.body !== undefined) {
                headers["Content-Type"] = "application/json";
                requestOpts.body = JSON.stringify(opts.body);
            }

            const response = await http.request<T>(url, requestOpts);

            if (!response.ok) {
                const text = await response.text();
                throw new Error(`runtime request failed (${response.status}): ${text || "Unknown error"}`);
            }

            if (opts?.expectText) {
                return response.text();
            }

            return response.json();
        },

        async waitForHealth(timeoutMs: number): Promise<void> {
            if (!config.enabled) return;

            const url = buildUrl("/healthz");
            const deadline = timer.now() + timeoutMs;

            while (timer.now() < deadline) {
                try {
                    const response = await http.request(url);
                    if (response.ok) {
                        return;
                    }
                } catch {
                    // Keep retrying
                }
                await timer.sleep(200);
            }

            throw new Error("Runtime control API did not respond before timeout");
        },

        async collectDiagnostics(serviceIds?: string[]): Promise<RuntimeDiagnostics> {
            const [readyResult, portsResult, telemetryResult] = await Promise.all([
                client.request<RuntimeReadyResponse>("/readyz"),
                client.request<RuntimePortsResponse>("/ports"),
                client.request<RuntimeTelemetryResponse>("/telemetry"),
            ]);

            // Type guard the results
            const ready: RuntimeReadyResponse = typeof readyResult === "string" ? { ready: false } : readyResult;
            const ports: RuntimePortsResponse = typeof portsResult === "string" ? {} : portsResult;
            const telemetryInfo: RuntimeTelemetryResponse | null = typeof telemetryResult === "string" ? null : telemetryResult;

            // Collect service IDs from responses
            const services = new Set<string>(serviceIds ?? []);
            Object.keys(ready.details ?? {}).forEach((svc) => services.add(svc));
            Object.keys(ports.services ?? {}).forEach((svc) => services.add(svc));

            // Fetch logs for each service
            const logs: Record<string, string> = {};
            for (const serviceId of services) {
                try {
                    const logData = await client.request<string>(
                        `/logs/tail?serviceId=${encodeURIComponent(serviceId)}&lines=${config.logLines}`,
                        { expectText: true }
                    );
                    if (typeof logData === "string") {
                        logs[serviceId] = logData;
                    }
                } catch {
                    // Log fetch failed, continue with other services
                }
            }

            const diagnostics: RuntimeDiagnostics = {
                ready,
                ports,
                logs,
            };

            if (ready.gpu) {
                diagnostics.gpu = ready.gpu;
            }
            if (telemetryInfo?.path) {
                diagnostics.telemetryPath = telemetryInfo.path;
            }
            if (telemetryInfo?.upload_url) {
                diagnostics.telemetryUploadUrl = telemetryInfo.upload_url;
            }

            return diagnostics;
        },

        async validate(): Promise<RuntimeValidationResponse | null> {
            if (!config.enabled) return null;

            try {
                const response = await client.request<RuntimeValidationResponse>("/validate");
                if (typeof response === "string") {
                    return null;
                }
                return response;
            } catch {
                return null;
            }
        },
    };

    return client;
}

/**
 * Create a fetch-based HTTP client for production use.
 */
export function createFetchRuntimeHttpClient(): IRuntimeHttpClient {
    return {
        async request<T>(url: string, opts?: { method?: string; headers?: Record<string, string>; body?: string }) {
            const init: RequestInit = {
                method: opts?.method ?? "GET",
            };
            if (opts?.headers) {
                init.headers = opts.headers;
            }
            if (opts?.body) {
                init.body = opts.body;
            }
            const response = await fetch(url, init);
            return {
                ok: response.ok,
                status: response.status,
                text: () => response.text(),
                json: () => response.json() as Promise<T>,
            };
        },
    };
}

/**
 * Create a real timer for production use.
 */
export function createRealTimer(): ITimer {
    return {
        now: () => Date.now(),
        sleep: (ms: number) => new Promise((resolve) => setTimeout(resolve, ms)),
    };
}

/**
 * Create a Node.js fs-based filesystem interface.
 */
export function createNodeRuntimeFileSystem(fsPromises: typeof import("node:fs").promises): IRuntimeFileSystem {
    return {
        readFile: (path, encoding) => fsPromises.readFile(path, encoding),
        access: (path) => fsPromises.access(path),
        stat: (path) => fsPromises.stat(path),
    };
}
