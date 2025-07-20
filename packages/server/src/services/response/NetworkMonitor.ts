import { HttpStatus, SECONDS_30_MS } from "@vrooli/shared";
import dns from "dns";
import fetch from "node-fetch";
import { promisify } from "util";
import { logger } from "../../events/logger.js";

const resolveDns = promisify(dns.resolve4);

export interface NetworkState {
    isOnline: boolean;
    connectivity: "online" | "offline" | "degraded";
    lastChecked: Date;
    cloudServicesReachable: boolean;
    localServicesReachable: boolean;
}

interface ServiceEndpoint {
    name: string;
    url: string;
    timeout: number;
}

/**
 * Service for monitoring network connectivity and service availability
 */
export class NetworkMonitor {
    private static instance: NetworkMonitor;
    private state: NetworkState;
    private updateInterval: NodeJS.Timeout | null = null;
    private readonly UPDATE_INTERVAL_MS = SECONDS_30_MS;
    /** Time to cache state before rechecking (ms) */
    private readonly CACHE_DURATION_MS = SECONDS_30_MS;
    private lastUpdateTime = 0;

    private readonly SERVER_ERROR_THRESHOLD = HttpStatus.InternalServerError;

    private readonly cloudEndpoints: ServiceEndpoint[] = [
        { name: "OpenRouter", url: "https://openrouter.ai/api/v1/models", timeout: 5000 },
        { name: "Cloudflare", url: "https://api.cloudflare.com/client/v4/user/tokens/verify", timeout: 5000 },
    ];

    private readonly localEndpoints: ServiceEndpoint[] = [
        { name: "Ollama", url: process.env.OLLAMA_BASE_URL || "http://localhost:11434", timeout: 2000 },
    ];

    private constructor() {
        this.state = {
            isOnline: true,
            connectivity: "online",
            lastChecked: new Date(),
            cloudServicesReachable: true,
            localServicesReachable: true,
        };
    }

    static getInstance(): NetworkMonitor {
        if (!NetworkMonitor.instance) {
            NetworkMonitor.instance = new NetworkMonitor();
        }
        return NetworkMonitor.instance;
    }

    /**
     * Start periodic network monitoring
     */
    start(): void {
        if (this.updateInterval) return;

        // Initial check
        this.updateState().catch(error => {
            logger.error("Failed to perform initial network check", { error });
        });

        // Periodic updates
        this.updateInterval = setInterval(() => {
            this.updateState().catch(error => {
                logger.error("Failed to update network state", { error });
            });
        }, this.UPDATE_INTERVAL_MS);
    }

    /**
     * Stop network monitoring
     */
    stop(): void {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
            this.updateInterval = null;
        }
    }

    /**
     * Get current network state (cached)
     */
    async getState(): Promise<NetworkState> {
        const now = Date.now();

        // Return cached state if fresh
        if (now - this.lastUpdateTime < this.CACHE_DURATION_MS) {
            return this.state;
        }

        // Update state if stale
        await this.updateState();
        return this.state;
    }

    /**
     * Force an immediate network state update
     */
    async forceUpdate(): Promise<NetworkState> {
        await this.updateState();
        return this.state;
    }

    /**
     * Check if a specific URL is reachable
     */
    private async checkEndpoint(endpoint: ServiceEndpoint): Promise<boolean> {
        try {
            const controller = new AbortController();
            const timeout = setTimeout(() => controller.abort(), endpoint.timeout);

            const response = await fetch(endpoint.url, {
                method: "HEAD",
                signal: controller.signal as any,
            });

            clearTimeout(timeout);
            return response.ok || response.status < this.SERVER_ERROR_THRESHOLD;
        } catch (error) {
            return false;
        }
    }

    /**
     * Check DNS resolution
     */
    private async checkDns(): Promise<boolean> {
        try {
            await resolveDns("google.com");
            return true;
        } catch {
            return false;
        }
    }

    /**
     * Update the network state
     */
    private async updateState(): Promise<void> {
        this.lastUpdateTime = Date.now();

        // Check DNS first
        const dnsWorks = await this.checkDns();
        if (!dnsWorks) {
            this.state = {
                isOnline: false,
                connectivity: "offline",
                lastChecked: new Date(),
                cloudServicesReachable: false,
                localServicesReachable: false,
            };
            return;
        }

        // Check cloud services in parallel
        const cloudChecks = await Promise.all(
            this.cloudEndpoints.map(endpoint => this.checkEndpoint(endpoint)),
        );
        const cloudServicesReachable = cloudChecks.some(result => result);

        // Check local services
        const localChecks = await Promise.all(
            this.localEndpoints.map(endpoint => this.checkEndpoint(endpoint)),
        );
        const localServicesReachable = localChecks.some(result => result);

        // Determine overall connectivity
        let connectivity: "online" | "offline" | "degraded";
        if (cloudServicesReachable && localServicesReachable) {
            connectivity = "online";
        } else if (!cloudServicesReachable && !localServicesReachable) {
            connectivity = "offline";
        } else {
            connectivity = "degraded";
        }

        this.state = {
            isOnline: cloudServicesReachable || localServicesReachable,
            connectivity,
            lastChecked: new Date(),
            cloudServicesReachable,
            localServicesReachable,
        };

        logger.info("Network state updated", { state: this.state });
    }

    /**
     * Check if we should use local models based on network state
     */
    shouldUseLocalModels(): boolean {
        return !this.state.cloudServicesReachable && this.state.localServicesReachable;
    }

    /**
     * Check if we're completely offline
     */
    isOffline(): boolean {
        return !this.state.isOnline;
    }
}
