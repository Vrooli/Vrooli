/**
 * Judge0 Code Execution Resource Implementation
 * 
 * Provides infrastructure-level management for Judge0 service including:
 * - Service discovery and health monitoring
 * - Language availability checking
 * - Code submission and result retrieval
 * - Security configuration validation
 * - Integration with unified configuration system
 */

import { logger } from "../../../events/logger.js";
import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import type { Judge0Config } from "../typeRegistry.js";
import type { HealthCheckResult, IResource } from "../types.js";
import { DeploymentType, ResourceCategory } from "../types.js";

/**
 * Judge0 language information
 */
export interface Judge0Language {
    id: number;
    name: string;
    source_file?: string;
    compile_cmd?: string;
    run_cmd?: string;
}

/**
 * Judge0 submission status
 */
export interface Judge0Status {
    id: number;
    description: string;
}

/**
 * Judge0 submission result
 */
export interface Judge0Submission {
    token?: string;
    stdout?: string;
    stderr?: string;
    compile_output?: string;
    status: Judge0Status;
    time?: string;
    memory?: number;
    created_at?: string;
    finished_at?: string;
}

/**
 * Judge0 system information
 */
export interface Judge0SystemInfo {
    version?: string;
    cpu_time_limit?: number;
    cpu_extra_time?: number;
    wall_time_limit?: number;
    memory_limit?: number;
    stack_limit?: number;
    max_processes_and_or_threads?: number;
    enable_per_process_and_thread_time_limit?: boolean;
    enable_per_process_and_thread_memory_limit?: boolean;
    max_file_size?: number;
    number_of_runners?: number;
}

/**
 * Judge0 execution metadata
 */
export interface ExecutionMetadata {
    version?: string;
    capabilities: string[];
    lastUpdated: Date;
    discoveredAt?: Date;
    supportedLanguages?: number;
    securityLimits?: {
        cpuTime: number;
        wallTime: number;
        memory: number;
        processes: number;
        fileSize: number;
    };
    workers?: {
        active: number;
        configured: number;
    };
}

/**
 * Code execution parameters
 */
export interface ExecuteCodeParams {
    source_code: string;
    language_id: number;
    stdin?: string;
    expected_output?: string;
    cpu_time_limit?: number;
    wall_time_limit?: number;
    memory_limit?: number;
    stack_limit?: number;
    max_processes_and_or_threads?: number;
    enable_network?: boolean;
    number_of_runs?: number;
    additional_files?: string;
    callback_url?: string;
    compiler_options?: string;
    command_line_arguments?: string;
}

/**
 * Production Judge0 resource implementation with enhanced monitoring
 */
@RegisterResource
export class Judge0Resource extends ResourceProvider<"judge0", Judge0Config> implements IResource {
    // Resource identification
    readonly id = "judge0" as const;
    readonly category = ResourceCategory.Execution;
    readonly displayName = "Judge0";
    readonly description = "Secure sandboxed code execution supporting 60+ programming languages";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    // Language caching
    private availableLanguages: Map<number, Judge0Language> = new Map();
    private lastLanguageRefresh = 0;
    private readonly CACHE_TTL_MS = 300000; // 5 minutes

    // System info caching
    private systemInfo: Judge0SystemInfo | null = null;
    private lastSystemInfoRefresh = 0;

    /**
     * Perform service discovery by checking Judge0 API availability
     */
    protected async performDiscovery(): Promise<boolean> {
        if (!this.config?.baseUrl) {
            logger.debug(`[${this.id}] No baseUrl configured`);
            return false;
        }

        try {
            // Try to reach the system info endpoint
            const result = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/system_info`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
                headers: this.getAuthHeaders(),
            });

            return result.success && result.data?.version;
        } catch (error) {
            logger.debug(`[${this.id}] Discovery failed:`, error);
            return false;
        }
    }

    /**
     * Perform comprehensive health check
     */
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        if (!this.config?.baseUrl) {
            return {
                healthy: false,
                message: "No baseUrl configured",
                timestamp: new Date(),
            };
        }

        try {
            // Get system info
            const systemResult = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/system_info`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
                headers: this.getAuthHeaders(),
            });

            if (!systemResult.success) {
                return {
                    healthy: false,
                    message: "Service not responding",
                    timestamp: new Date(),
                    details: { error: systemResult.error },
                };
            }

            // Cache system info
            this.systemInfo = systemResult.data;
            this.lastSystemInfoRefresh = Date.now();

            // Try to get languages count
            let languageCount = 0;
            try {
                await this.refreshLanguages();
                languageCount = this.availableLanguages.size;
            } catch (error) {
                logger.debug(`[${this.id}] Could not fetch languages:`, error);
            }

            return {
                healthy: true,
                message: `Judge0 v${systemResult.data.version || "unknown"} - ${languageCount} languages available`,
                timestamp: new Date(),
                details: {
                    version: systemResult.data.version,
                    languages: languageCount,
                    limits: {
                        cpuTime: systemResult.data.cpu_time_limit,
                        memory: systemResult.data.memory_limit,
                        processes: systemResult.data.max_processes_and_or_threads,
                    },
                },
            };

        } catch (error) {
            logger.error(`[${this.id}] Health check failed:`, error);
            return {
                healthy: false,
                message: error instanceof Error ? error.message : "Health check failed",
                timestamp: new Date(),
            };
        }
    }

    /**
     * Get resource-specific metadata
     */
    protected getMetadata(): ExecutionMetadata {
        return {
            version: this.systemInfo?.version,
            capabilities: [
                "code-execution",
                "multi-language",
                "sandboxed",
                "resource-limits",
                "batch-submissions",
            ],
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            supportedLanguages: this.availableLanguages.size,
            securityLimits: this.systemInfo ? {
                cpuTime: this.systemInfo.cpu_time_limit || 0,
                wallTime: this.systemInfo.wall_time_limit || 0,
                memory: this.systemInfo.memory_limit || 0,
                processes: this.systemInfo.max_processes_and_or_threads || 0,
                fileSize: this.systemInfo.max_file_size || 0,
            } : undefined,
            workers: {
                active: this.systemInfo?.number_of_runners || 0,
                configured: this.config?.workers || 2,
            },
        };
    }

    /**
     * Execute code with Judge0
     */
    async executeCode(params: ExecuteCodeParams): Promise<Judge0Submission> {
        if (!this.config?.baseUrl) {
            throw new Error("Judge0 baseUrl not configured");
        }

        // Apply security limits from config
        const submission = {
            ...params,
            cpu_time_limit: params.cpu_time_limit || this.config.securityLimits?.cpuTime || 5,
            wall_time_limit: params.wall_time_limit || this.config.securityLimits?.wallTime || 10,
            memory_limit: params.memory_limit || this.config.securityLimits?.memoryLimit || 262144,
            stack_limit: params.stack_limit || this.config.securityLimits?.stackLimit || 262144,
            max_processes_and_or_threads: params.max_processes_and_or_threads || this.config.securityLimits?.maxProcesses || 30,
            enable_network: params.enable_network ?? false,
        };

        const result = await this.httpClient!.makeRequest({
            url: `${this.config.baseUrl}/submissions?wait=true`,
            method: "POST",
            headers: {
                ...this.getAuthHeaders(),
                "Content-Type": "application/json",
            },
            body: submission,
            timeout: (submission.wall_time_limit + 5) * 1000, // Add 5s buffer
        });

        if (!result.success) {
            throw new Error(`Code execution failed: ${result.error}`);
        }

        // Decode base64 outputs
        const response = result.data;
        if (response.stdout) {
            response.stdout = Buffer.from(response.stdout, "base64").toString();
        }
        if (response.stderr) {
            response.stderr = Buffer.from(response.stderr, "base64").toString();
        }
        if (response.compile_output) {
            response.compile_output = Buffer.from(response.compile_output, "base64").toString();
        }

        return response;
    }

    /**
     * Submit batch of code executions
     */
    async submitBatch(submissions: ExecuteCodeParams[]): Promise<Judge0Submission[]> {
        if (!this.config?.baseUrl) {
            throw new Error("Judge0 baseUrl not configured");
        }

        // Apply security limits to all submissions
        const securedSubmissions = submissions.map(params => ({
            ...params,
            cpu_time_limit: params.cpu_time_limit || this.config!.securityLimits?.cpuTime || 5,
            wall_time_limit: params.wall_time_limit || this.config!.securityLimits?.wallTime || 10,
            memory_limit: params.memory_limit || this.config!.securityLimits?.memoryLimit || 262144,
            enable_network: params.enable_network ?? false,
        }));

        const result = await this.httpClient!.makeRequest({
            url: `${this.config.baseUrl}/submissions/batch?wait=true`,
            method: "POST",
            headers: {
                ...this.getAuthHeaders(),
                "Content-Type": "application/json",
            },
            body: { submissions: securedSubmissions },
            timeout: 60000, // 1 minute for batch
        });

        if (!result.success) {
            throw new Error(`Batch submission failed: ${result.error}`);
        }

        // Decode base64 outputs for all submissions
        return result.data.map((submission: Judge0Submission) => {
            if (submission.stdout) {
                submission.stdout = Buffer.from(submission.stdout, "base64").toString();
            }
            if (submission.stderr) {
                submission.stderr = Buffer.from(submission.stderr, "base64").toString();
            }
            if (submission.compile_output) {
                submission.compile_output = Buffer.from(submission.compile_output, "base64").toString();
            }
            return submission;
        });
    }

    /**
     * List available languages
     */
    async listLanguages(): Promise<Judge0Language[]> {
        await this.refreshLanguages();
        return Array.from(this.availableLanguages.values());
    }

    /**
     * Get language by ID
     */
    async getLanguage(id: number): Promise<Judge0Language | undefined> {
        await this.refreshLanguages();
        return this.availableLanguages.get(id);
    }

    /**
     * Find language by name
     */
    async findLanguageByName(name: string): Promise<Judge0Language | undefined> {
        await this.refreshLanguages();
        const lowerName = name.toLowerCase();
        
        for (const lang of this.availableLanguages.values()) {
            if (lang.name.toLowerCase().includes(lowerName)) {
                return lang;
            }
        }
        
        return undefined;
    }

    /**
     * Refresh language information
     */
    private async refreshLanguages(): Promise<void> {
        const now = Date.now();
        if (now - this.lastLanguageRefresh < this.CACHE_TTL_MS) {
            return; // Use cached data
        }

        if (!this.config?.baseUrl) {
            return;
        }

        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/languages`,
                method: "GET",
                headers: this.getAuthHeaders(),
                timeout: 5000,
            });

            if (result.success && Array.isArray(result.data)) {
                this.availableLanguages.clear();
                
                for (const lang of result.data) {
                    this.availableLanguages.set(lang.id, {
                        id: lang.id,
                        name: lang.name,
                        source_file: lang.source_file,
                        compile_cmd: lang.compile_cmd,
                        run_cmd: lang.run_cmd,
                    });
                }

                this.lastLanguageRefresh = now;
                logger.debug(`[${this.id}] Refreshed ${this.availableLanguages.size} languages`);
            }
        } catch (error) {
            logger.error(`[${this.id}] Failed to refresh languages:`, error);
            throw error;
        }
    }

    /**
     * Get authentication headers
     */
    private getAuthHeaders(): Record<string, string> {
        if (!this.config?.apiKey) {
            return {};
        }

        return {
            "X-Auth-Token": this.config.apiKey,
        };
    }

    /**
     * Get system information
     */
    async getSystemInfo(): Promise<Judge0SystemInfo> {
        if (!this.config?.baseUrl) {
            throw new Error("Judge0 baseUrl not configured");
        }

        // Use cache if fresh
        const now = Date.now();
        if (this.systemInfo && now - this.lastSystemInfoRefresh < this.CACHE_TTL_MS) {
            return this.systemInfo;
        }

        const result = await this.httpClient!.makeRequest({
            url: `${this.config.baseUrl}/system_info`,
            method: "GET",
            headers: this.getAuthHeaders(),
            timeout: 5000,
        });

        if (!result.success) {
            throw new Error(`Failed to get system info: ${result.error}`);
        }

        this.systemInfo = result.data;
        this.lastSystemInfoRefresh = now;
        
        return this.systemInfo!;
    }
}
