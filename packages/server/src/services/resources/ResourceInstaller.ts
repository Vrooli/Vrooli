import { spawn } from "child_process";
import { promises as fs } from "fs";
import { resolve } from "path";
import { EventEmitter } from "events";
import { logger } from "../../events/logger.js";
import type { ResourceRegistry } from "./ResourceRegistry.js";
import type { ResourceId } from "./typeRegistry.js";
import { ResourceCategory, type PublicResourceInfo, DiscoveryStatus, ResourceHealth } from "./types.js";

/**
 * Result of a resource installation operation
 */
export interface InstallationResult {
    success: boolean;
    resourceId: string;
    message: string;
    details?: {
        duration: number;
        rollbackPerformed?: boolean;
        errors?: string[];
        warnings?: string[];
    };
}

/**
 * Options for resource installation
 */
export interface InstallationOptions {
    force?: boolean;  // Force installation even if already present
    skipModels?: boolean;  // Skip model downloads (for AI resources)
    models?: string[];  // Specific models to install
    timeout?: number;  // Timeout in milliseconds (default: 300000 = 5 minutes)
    progressCallback?: (progress: InstallationProgress) => void;
}

/**
 * Installation progress information
 */
export interface InstallationProgress {
    resourceId: string;
    stage: "validation" | "download" | "installation" | "configuration" | "models" | "verification" | "cleanup";
    message: string;
    progress?: number;  // 0-100 percentage
    timestamp: Date;
}

/**
 * Rollback action for error recovery
 */
interface RollbackAction {
    description: string;
    execute: () => Promise<void>;
}

/**
 * ResourceInstaller - Unified TypeScript interface for managing local resources
 * 
 * This service addresses architectural complexity by providing a single point of control
 * where TypeScript orchestrates shell scripts rather than having separate systems.
 * 
 * Key improvements:
 * - Unified error handling and rollback mechanisms
 * - Progress reporting and status updates
 * - Integration with existing ResourceRegistry
 * - Atomic configuration management
 * - Comprehensive validation
 */
export class ResourceInstaller extends EventEmitter {
    private readonly scriptsPath: string;
    private readonly registry?: ResourceRegistry;
    private readonly activeInstallations = new Map<string, AbortController>();
    
    constructor(options?: { registry?: ResourceRegistry }) {
        super();
        
        // Resolve path to scripts directory
        this.scriptsPath = resolve(process.cwd(), "scripts", "resources");
        this.registry = options?.registry;
        
        logger.info("[ResourceInstaller] Initialized", { 
            scriptsPath: this.scriptsPath,
            hasRegistry: !!this.registry, 
        });
    }
    
    /**
     * Install a resource with comprehensive error handling and rollback
     */
    async installResource(
        resourceId: ResourceId, 
        options: InstallationOptions = {},
    ): Promise<InstallationResult> {
        const startTime = Date.now();
        const rollbackActions: RollbackAction[] = [];
        
        // Check if installation is already in progress
        if (this.activeInstallations.has(resourceId)) {
            return {
                success: false,
                resourceId,
                message: `Installation of ${resourceId} is already in progress`,
            };
        }
        
        // Create abort controller for cancellation
        const abortController = new AbortController();
        this.activeInstallations.set(resourceId, abortController);
        
        const progressCallback = options.progressCallback;
        const reportProgress = (stage: InstallationProgress["stage"], message: string, progress?: number) => {
            const progressInfo: InstallationProgress = {
                resourceId,
                stage,
                message,
                progress,
                timestamp: new Date(),
            };
            
            progressCallback?.(progressInfo);
            this.emit("progress", progressInfo);
            logger.info(`[ResourceInstaller] ${resourceId}: ${stage} - ${message}`, { progress });
        };
        
        try {
            // Stage 1: Validation
            reportProgress("validation", "Validating installation prerequisites", 10);
            await this.validatePrerequisites(resourceId);
            
            // Stage 2: Pre-installation checks
            reportProgress("validation", "Checking resource availability", 20);
            const category = this.getResourceCategory(resourceId);
            const scriptPath = this.getResourceScriptPath(resourceId, category);
            
            // Verify script exists
            try {
                await fs.access(scriptPath);
            } catch (error) {
                throw new Error(`Resource script not found: ${scriptPath}. Resource '${resourceId}' may not be implemented yet.`);
            }
            
            // Stage 3: Execute installation script
            reportProgress("installation", "Running installation script", 30);
            const scriptResult = await this.executeResourceScript(
                resourceId,
                "install",
                options,
                abortController.signal,
                (message, progress) => reportProgress("installation", message, 30 + (progress || 0) * 0.6),
            );
            
            if (!scriptResult.success) {
                throw new Error(`Installation script failed: ${scriptResult.error}`);
            }
            
            // Add rollback action for script installation
            rollbackActions.push({
                description: `Uninstall ${resourceId}`,
                execute: async () => {
                    await this.executeResourceScript(resourceId, "uninstall", { force: true });
                },
            });
            
            // Stage 4: Verify installation
            reportProgress("verification", "Verifying installation", 90);
            await this.verifyInstallation(resourceId);
            
            // Stage 5: Trigger resource discovery
            reportProgress("verification", "Updating resource registry", 95);
            if (this.registry) {
                // Wait a moment for services to start
                await new Promise(resolve => setTimeout(resolve, 2000));
                
                try {
                    // Trigger rediscovery to pick up the new resource
                    await this.registry.rediscoverResources();
                    
                    // Verify the resource was discovered
                    const resource = this.registry.getResource(resourceId);
                    if (resource) {
                        const info = resource.getPublicInfo();
                        logger.info(`[ResourceInstaller] ${resourceId} discovered by registry`, {
                            status: info.status,
                            health: info.health,
                        });
                    }
                } catch (registryError) {
                    logger.warn(`[ResourceInstaller] Failed to update registry for ${resourceId}`, registryError);
                    // Don't fail the installation for registry issues
                }
            }
            
            // Stage 6: Complete
            reportProgress("verification", "Installation completed successfully", 100);
            
            const duration = Date.now() - startTime;
            logger.info(`[ResourceInstaller] Successfully installed ${resourceId}`, { duration });
            
            return {
                success: true,
                resourceId,
                message: `Successfully installed ${resourceId}`,
                details: {
                    duration,
                    warnings: scriptResult.warnings,
                },
            };
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            logger.error(`[ResourceInstaller] Failed to install ${resourceId}`, error);
            
            // Perform rollback
            reportProgress("cleanup", "Rolling back installation due to error", 0);
            const rollbackResult = await this.performRollback(rollbackActions);
            
            const duration = Date.now() - startTime;
            
            return {
                success: false,
                resourceId,
                message: `Failed to install ${resourceId}: ${errorMessage}`,
                details: {
                    duration,
                    rollbackPerformed: rollbackResult.performed,
                    errors: [errorMessage, ...rollbackResult.errors],
                },
            };
            
        } finally {
            // Clean up
            this.activeInstallations.delete(resourceId);
        }
    }
    
    /**
     * Uninstall a resource with dependency checking
     */
    async uninstallResource(resourceId: ResourceId, options: { force?: boolean } = {}): Promise<InstallationResult> {
        const startTime = Date.now();
        
        try {
            // Check dependencies before uninstalling
            if (!options.force && this.registry) {
                const canUninstall = this.registry.canSafelyShutdownResource(resourceId);
                if (!canUninstall) {
                    const dependents = this.registry.getDependentResources(resourceId);
                    throw new Error(
                        `Cannot uninstall ${resourceId} - it has dependent resources: ${dependents.join(", ")}. ` +
                        "Use force option to override.",
                    );
                }
            }
            
            // Execute uninstall script
            const scriptResult = await this.executeResourceScript(resourceId, "uninstall", options);
            
            if (!scriptResult.success) {
                throw new Error(`Uninstall script failed: ${scriptResult.error}`);
            }
            
            // Update registry
            if (this.registry) {
                await this.registry.rediscoverResources();
            }
            
            const duration = Date.now() - startTime;
            
            return {
                success: true,
                resourceId,
                message: `Successfully uninstalled ${resourceId}`,
                details: { duration },
            };
            
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            logger.error(`[ResourceInstaller] Failed to uninstall ${resourceId}`, error);
            
            return {
                success: false,
                resourceId,
                message: `Failed to uninstall ${resourceId}: ${errorMessage}`,
                details: {
                    duration: Date.now() - startTime,
                    errors: [errorMessage],
                },
            };
        }
    }
    
    /**
     * Get the status of a resource (combines script and registry information)
     */
    async getResourceStatus(resourceId: ResourceId): Promise<{
        installed: boolean;
        running: boolean;
        healthy: boolean;
        details: any;
    }> {
        try {
            // Get information from script
            const scriptResult = await this.executeResourceScript(resourceId, "status");
            
            // Get information from registry if available
            let registryInfo: PublicResourceInfo | null = null;
            if (this.registry) {
                const resource = this.registry.getResource(resourceId);
                if (resource) {
                    registryInfo = resource.getPublicInfo();
                }
            }
            
            return {
                installed: scriptResult.success,
                running: registryInfo?.status === DiscoveryStatus.Available || false,
                healthy: registryInfo?.health === ResourceHealth.Healthy || false,
                details: {
                    script: scriptResult,
                    registry: registryInfo,
                },
            };
            
        } catch (error) {
            logger.error(`[ResourceInstaller] Failed to get status for ${resourceId}`, error);
            return {
                installed: false,
                running: false,
                healthy: false,
                details: { error: error instanceof Error ? error.message : String(error) },
            };
        }
    }
    
    /**
     * Cancel an ongoing installation
     */
    cancelInstallation(resourceId: ResourceId): boolean {
        const controller = this.activeInstallations.get(resourceId);
        if (controller) {
            controller.abort();
            this.activeInstallations.delete(resourceId);
            logger.info(`[ResourceInstaller] Cancelled installation of ${resourceId}`);
            return true;
        }
        return false;
    }
    
    /**
     * Get list of resources that can be installed
     */
    async getAvailableResources(): Promise<Array<{
        id: ResourceId;
        category: ResourceCategory;
        available: boolean;
        description?: string;
    }>> {
        const resources: Array<{ id: ResourceId; category: ResourceCategory; available: boolean; description?: string }> = [];
        
        try {
            // Execute the list command from the main orchestrator
            const result = await this.executeShellCommand(
                resolve(this.scriptsPath, "index.sh"),
                ["--action", "list"],
                { timeout: 10000 },
            );
            
            if (result.success) {
                // Parse the output to extract resource information
                // This is a simplified parser - in production you'd want more robust parsing
                const lines = result.stdout.split("\n");
                for (const line of lines) {
                    if (line.includes("✅ Available") || line.includes("❌ Not implemented")) {
                        const match = line.match(/- (\w+): (✅ Available|❌ Not implemented)/);
                        if (match) {
                            const [, id, status] = match;
                            // We need to determine the category from the resource ID
                            const category = this.getResourceCategory(id as ResourceId);
                            resources.push({
                                id: id as ResourceId,
                                category,
                                available: status === "✅ Available",
                            });
                        }
                    }
                }
            }
        } catch (error) {
            logger.error("[ResourceInstaller] Failed to get available resources", error);
        }
        
        return resources;
    }
    
    // Private helper methods
    
    private async validatePrerequisites(resourceId: ResourceId): Promise<void> {
        // Check if we have necessary permissions
        try {
            await fs.access(this.scriptsPath);
        } catch (error) {
            throw new Error(`Scripts directory not accessible: ${this.scriptsPath}`);
        }
        
        // Check for required system dependencies
        const requiredCommands = ["bash", "curl"];
        for (const cmd of requiredCommands) {
            try {
                await this.executeShellCommand("which", [cmd], { timeout: 5000 });
            } catch (error) {
                throw new Error(`Required command not found: ${cmd}. Please install it before proceeding.`);
            }
        }
    }
    
    private getResourceCategory(resourceId: string): ResourceCategory {
        // AI resources
        if (["ollama", "whisper", "comfyui", "cloudflare", "openrouter"].includes(resourceId)) {
            return ResourceCategory.AI;
        }
        // Automation resources
        if (["n8n", "node-red", "windmill", "activepieces", "huginn", "airflow", "temporal"].includes(resourceId)) {
            return ResourceCategory.Automation;
        }
        // Agent resources
        if (["browserless", "claude-code"].includes(resourceId)) {
            return ResourceCategory.Agents;
        }
        // Storage resources
        if (["minio", "ipfs", "rclone"].includes(resourceId)) {
            return ResourceCategory.Storage;
        }
        
        throw new Error(`Unknown resource category for: ${resourceId}`);
    }
    
    private getResourceScriptPath(resourceId: string, category: ResourceCategory): string {
        const categoryDir = category.toLowerCase();
        return resolve(this.scriptsPath, categoryDir, `${resourceId}.sh`);
    }
    
    private async executeResourceScript(
        resourceId: ResourceId,
        action: string,
        options: any = {},
        signal?: AbortSignal,
        progressCallback?: (message: string, progress?: number) => void,
    ): Promise<{
        success: boolean;
        error?: string;
        warnings?: string[];
    }> {
        const category = this.getResourceCategory(resourceId);
        const scriptPath = this.getResourceScriptPath(resourceId, category);
        
        const args = ["--action", action];
        
        // Add options as arguments
        if (options.force) args.push("--force", "yes");
        if (options.skipModels) args.push("--skip-models", "yes");
        if (options.models) args.push("--models", options.models.join(","));
        args.push("--yes", "yes"); // Always use non-interactive mode
        
        const result = await this.executeShellCommand(
            scriptPath,
            args,
            {
                timeout: options.timeout || 300000, // 5 minute default timeout
                signal,
                progressCallback,
            },
        );
        
        return {
            success: result.success,
            error: result.success ? undefined : result.stderr || result.error,
            warnings: result.warnings,
        };
    }
    
    private async executeShellCommand(
        command: string,
        args: string[],
        options: {
            timeout?: number;
            signal?: AbortSignal;
            progressCallback?: (message: string, progress?: number) => void;
        } = {},
    ): Promise<{
        success: boolean;
        stdout: string;
        stderr: string;
        error?: string;
        warnings?: string[];
    }> {
        return new Promise((resolve) => {
            const { timeout = 120000, signal, progressCallback } = options;
            
            logger.debug(`[ResourceInstaller] Executing: ${command} ${args.join(" ")}`);
            
            const child = spawn(command, args, {
                stdio: ["pipe", "pipe", "pipe"],
                signal,
            });
            
            let stdout = "";
            let stderr = "";
            const warnings: string[] = [];
            
            // Set up timeout
            const timeoutId = setTimeout(() => {
                child.kill("SIGTERM");
                resolve({
                    success: false,
                    stdout,
                    stderr,
                    error: `Command timed out after ${timeout}ms`,
                });
            }, timeout);
            
            // Handle stdout
            child.stdout?.on("data", (data) => {
                const output = data.toString();
                stdout += output;
                
                // Extract progress information and warnings
                const lines = output.split("\n");
                for (const line of lines) {
                    if (line.includes("[INFO]") || line.includes("[SUCCESS]")) {
                        progressCallback?.(line.replace(/^\[.*?\]\s*/, ""));
                    } else if (line.includes("[WARN]")) {
                        warnings.push(line.replace(/^\[.*?\]\s*/, ""));
                    }
                }
            });
            
            // Handle stderr
            child.stderr?.on("data", (data) => {
                stderr += data.toString();
            });
            
            // Handle completion
            child.on("close", (code, signal) => {
                clearTimeout(timeoutId);
                
                if (signal) {
                    resolve({
                        success: false,
                        stdout,
                        stderr,
                        error: `Command was killed with signal ${signal}`,
                    });
                } else {
                    resolve({
                        success: code === 0,
                        stdout,
                        stderr,
                        error: code !== 0 ? `Command exited with code ${code}` : undefined,
                        warnings,
                    });
                }
            });
            
            // Handle errors
            child.on("error", (error) => {
                clearTimeout(timeoutId);
                resolve({
                    success: false,
                    stdout,
                    stderr,
                    error: error.message,
                });
            });
        });
    }
    
    private async verifyInstallation(resourceId: ResourceId): Promise<void> {
        // Execute status check to verify installation
        const result = await this.executeResourceScript(resourceId, "status");
        
        if (!result.success) {
            throw new Error(`Installation verification failed: ${result.error}`);
        }
    }
    
    private async performRollback(actions: RollbackAction[]): Promise<{
        performed: boolean;
        errors: string[];
    }> {
        if (actions.length === 0) {
            return { performed: false, errors: [] };
        }
        
        const errors: string[] = [];
        
        // Execute rollback actions in reverse order
        for (let i = actions.length - 1; i >= 0; i--) {
            const action = actions[i];
            try {
                logger.info(`[ResourceInstaller] Performing rollback: ${action.description}`);
                await action.execute();
            } catch (error) {
                const errorMessage = error instanceof Error ? error.message : String(error);
                logger.error(`[ResourceInstaller] Rollback action failed: ${action.description}`, error);
                errors.push(`Rollback failed - ${action.description}: ${errorMessage}`);
            }
        }
        
        return {
            performed: true,
            errors,
        };
    }
}
