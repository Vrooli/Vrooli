import { SECONDS_30_MS } from "@vrooli/shared";
import { randomBytes } from "crypto";
import { constants, promises as fs } from "fs";
import { dirname, resolve } from "path";
import { logger } from "../../events/logger.js";
import type { TypedResourcesConfig } from "./resourcesConfig.js";
import type { ResourceId } from "./typeRegistry.js";

// Constants
const TEMP_SUFFIX_LENGTH = 8;

/**
 * File lock for preventing concurrent access
 */
interface FileLock {
    path: string;
    lockFile: string;
    acquired: Date;
}

/**
 * Configuration update operation
 */
export interface ConfigurationUpdate {
    operation: "set" | "delete" | "merge";
    path: string; // JSON path (e.g., "services.ai.ollama")
    value?: any;
    backup?: boolean; // Whether to create a backup before update
}

/**
 * Result of a configuration operation
 */
export interface ConfigurationResult {
    success: boolean;
    message: string;
    backupPath?: string;
    errors?: string[];
    warnings?: string[];
}

/**
 * Configuration validation error
 */
export interface ValidationError {
    path: string;
    message: string;
    value?: any;
}

/**
 * ConfigurationManager - Robust, atomic configuration file management
 * 
 * This service addresses fragile configuration management by providing:
 * - Atomic file updates using temp files + move operations
 * - File locking to prevent concurrent modifications
 * - JSON schema validation
 * - Environment variable expansion
 * - Backup and rollback capabilities
 * - Graceful dependency handling (jq optional)
 */
export class ConfigurationManager {
    private readonly configPath: string;
    private readonly lockTimeout: number;
    private readonly activeLocks = new Map<string, FileLock>();

    constructor(configPath?: string, options: { lockTimeout?: number } = {}) {
        this.configPath = configPath || resolve(process.env.HOME || process.cwd(), ".vrooli", "resources.local.json");
        this.lockTimeout = options.lockTimeout || SECONDS_30_MS;

        logger.debug("[ConfigurationManager] Initialized", {
            configPath: this.configPath,
            lockTimeout: this.lockTimeout,
        });
    }

    /**
     * Atomically update configuration with validation and locking
     */
    async updateConfiguration(
        updates: ConfigurationUpdate | ConfigurationUpdate[],
    ): Promise<ConfigurationResult> {
        const updateArray = Array.isArray(updates) ? updates : [updates];
        const lockKey = this.configPath;

        try {
            // Acquire file lock
            const lock = await this.acquireLock(lockKey);

            try {
                // Ensure config directory exists
                await this.ensureConfigDirectory();

                // Load current configuration
                const currentConfig = await this.loadConfiguration();

                // Create backup if requested
                let backupPath: string | undefined;
                if (updateArray.some(u => u.backup !== false)) {
                    backupPath = await this.createBackup(currentConfig);
                }

                // Apply updates
                let updatedConfig = { ...currentConfig };
                const warnings: string[] = [];

                for (const update of updateArray) {
                    const result = this.applyUpdate(updatedConfig, update);
                    updatedConfig = result.config;
                    if (result.warnings) {
                        warnings.push(...result.warnings);
                    }
                }

                // Validate the updated configuration
                const validationResult = await this.validateConfiguration(updatedConfig);
                if (!validationResult.valid) {
                    throw new Error(`Configuration validation failed: ${validationResult.errors?.map(e => e.message).join(", ")}`);
                }

                // Perform atomic write
                await this.atomicWrite(this.configPath, updatedConfig);

                logger.info("[ConfigurationManager] Configuration updated successfully", {
                    updates: updateArray.length,
                    backupCreated: !!backupPath,
                });

                return {
                    success: true,
                    message: `Successfully updated configuration with ${updateArray.length} changes`,
                    backupPath,
                    warnings: warnings.length > 0 ? warnings : undefined,
                };

            } finally {
                // Always release the lock
                await this.releaseLock(lock);
            }

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            logger.error("[ConfigurationManager] Configuration update failed", error);

            return {
                success: false,
                message: `Configuration update failed: ${errorMessage}`,
                errors: [errorMessage],
            };
        }
    }

    /**
     * Load and validate current configuration
     */
    async loadConfiguration(): Promise<TypedResourcesConfig> {
        try {
            // Check if file exists
            try {
                await fs.access(this.configPath, constants.F_OK);
            } catch (error) {
                // File doesn't exist, return default configuration
                return this.getDefaultConfiguration();
            }

            // Read and parse file
            const content = await fs.readFile(this.configPath, "utf8");

            if (!content.trim()) {
                // Empty file, return default configuration
                return this.getDefaultConfiguration();
            }

            let config: TypedResourcesConfig;
            try {
                config = JSON.parse(content);
            } catch (parseError) {
                throw new Error(`Invalid JSON in configuration file: ${parseError instanceof Error ? parseError.message : "Unknown parsing error"}`);
            }

            // Expand environment variables
            config = this.expandEnvironmentVariables(config);

            // Validate configuration
            const validationResult = await this.validateConfiguration(config);
            if (!validationResult.valid) {
                logger.warn("[ConfigurationManager] Configuration validation failed, using defaults", {
                    errors: validationResult.errors,
                });
                // Return default config if validation fails
                return this.getDefaultConfiguration();
            }

            return config;

        } catch (error) {
            logger.error("[ConfigurationManager] Failed to load configuration, using defaults", error);
            return this.getDefaultConfiguration();
        }
    }

    /**
     * Update a specific resource configuration
     */
    async updateResourceConfiguration(
        resourceId: ResourceId,
        category: keyof NonNullable<TypedResourcesConfig["services"]>,
        resourceConfig: any,
    ): Promise<ConfigurationResult> {
        return this.updateConfiguration({
            operation: "set",
            path: `services.${category}.${resourceId}`,
            value: resourceConfig,
            backup: true,
        });
    }

    /**
     * Remove a resource configuration
     */
    async removeResourceConfiguration(
        resourceId: ResourceId,
        category: keyof NonNullable<TypedResourcesConfig["services"]>,
    ): Promise<ConfigurationResult> {
        return this.updateConfiguration({
            operation: "delete",
            path: `services.${category}.${resourceId}`,
            backup: true,
        });
    }

    /**
     * Create a configuration backup
     */
    async createBackup(config?: TypedResourcesConfig): Promise<string> {
        const configToBackup = config || await this.loadConfiguration();
        const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
        const backupPath = `${this.configPath}.backup.${timestamp}`;

        await this.atomicWrite(backupPath, configToBackup);

        logger.info("[ConfigurationManager] Configuration backup created", { backupPath });
        return backupPath;
    }

    /**
     * Restore configuration from backup
     */
    async restoreFromBackup(backupPath: string): Promise<ConfigurationResult> {
        try {
            // Verify backup file exists
            await fs.access(backupPath, constants.F_OK);

            // Load backup configuration
            const backupContent = await fs.readFile(backupPath, "utf8");
            const backupConfig: TypedResourcesConfig = JSON.parse(backupContent);

            // Validate backup configuration
            const validationResult = await this.validateConfiguration(backupConfig);
            if (!validationResult.valid) {
                throw new Error(`Backup configuration is invalid: ${validationResult.errors?.map(e => e.message).join(", ")}`);
            }

            // Create a backup of current config before restore
            const currentBackupPath = await this.createBackup();

            // Restore from backup
            await this.atomicWrite(this.configPath, backupConfig);

            logger.info("[ConfigurationManager] Configuration restored from backup", {
                backupPath,
                currentBackupPath,
            });

            return {
                success: true,
                message: `Configuration restored from backup: ${backupPath}`,
                backupPath: currentBackupPath,
            };

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            logger.error("[ConfigurationManager] Failed to restore from backup", error);

            return {
                success: false,
                message: `Failed to restore from backup: ${errorMessage}`,
                errors: [errorMessage],
            };
        }
    }

    /**
     * Validate configuration against schema
     */
    async validateConfiguration(config: any): Promise<{
        valid: boolean;
        errors?: ValidationError[];
        warnings?: string[];
    }> {
        const errors: ValidationError[] = [];
        const warnings: string[] = [];

        try {
            // Basic structure validation
            if (typeof config !== "object" || config === null) {
                errors.push({
                    path: "root",
                    message: "Configuration must be an object",
                });
                return { valid: false, errors };
            }

            // Check required fields
            if (!config.version || typeof config.version !== "string") {
                errors.push({
                    path: "version",
                    message: "Version is required and must be a string",
                });
            }

            if (typeof config.enabled !== "boolean") {
                errors.push({
                    path: "enabled",
                    message: "Enabled field must be a boolean",
                });
            }

            // Validate services structure
            if (config.services) {
                if (typeof config.services !== "object") {
                    errors.push({
                        path: "services",
                        message: "Services must be an object",
                    });
                } else {
                    // Validate each service category
                    const validCategories = ["ai", "automation", "agents", "storage"];
                    for (const [category, services] of Object.entries(config.services)) {
                        if (!validCategories.includes(category)) {
                            warnings.push(`Unknown service category: ${category}`);
                            continue;
                        }

                        if (services && typeof services === "object") {
                            for (const [serviceId, serviceConfig] of Object.entries(services as any)) {
                                this.validateServiceConfig(serviceId, serviceConfig as any, `services.${category}.${serviceId}`, errors);
                            }
                        }
                    }
                }
            }

            // Validate security configuration if present
            if (config.security) {
                this.validateSecurityConfig(config.security, errors);
            }

            return {
                valid: errors.length === 0,
                errors: errors.length > 0 ? errors : undefined,
                warnings: warnings.length > 0 ? warnings : undefined,
            };

        } catch (error) {
            logger.error("[ConfigurationManager] Configuration validation error", error);
            return {
                valid: false,
                errors: [{
                    path: "validation",
                    message: error instanceof Error ? error.message : "Unknown validation error",
                }],
            };
        }
    }

    // Private helper methods

    private async ensureConfigDirectory(): Promise<void> {
        const configDir = dirname(this.configPath);
        try {
            await fs.mkdir(configDir, { recursive: true });
        } catch (error) {
            if ((error as any).code !== "EEXIST") {
                throw error;
            }
        }
    }

    private async acquireLock(key: string): Promise<FileLock> {
        const lockFile = `${this.configPath}.lock`;
        const maxAttempts = Math.ceil(this.lockTimeout / 100); // Check every 100ms

        for (let attempt = 0; attempt < maxAttempts; attempt++) {
            try {
                // Try to create lock file exclusively
                await fs.writeFile(lockFile, JSON.stringify({
                    pid: process.pid,
                    acquired: new Date().toISOString(),
                    key,
                }), { flag: "wx" });

                const lock: FileLock = {
                    path: key,
                    lockFile,
                    acquired: new Date(),
                };

                this.activeLocks.set(key, lock);
                logger.debug("[ConfigurationManager] Lock acquired", { key, lockFile });
                return lock;

            } catch (error) {
                if ((error as any).code === "EEXIST") {
                    // Lock file exists, check if it's stale
                    try {
                        const lockInfo = JSON.parse(await fs.readFile(lockFile, "utf8"));
                        const lockAge = Date.now() - new Date(lockInfo.acquired).getTime();

                        if (lockAge > this.lockTimeout) {
                            // Lock is stale, remove it
                            await fs.unlink(lockFile);
                            logger.warn("[ConfigurationManager] Removed stale lock", { lockFile, age: lockAge });
                            continue;
                        }
                    } catch (lockError) {
                        // Corrupted lock file, remove it
                        await fs.unlink(lockFile).catch(() => {
                            // Ignore cleanup errors
                        });
                        continue;
                    }

                    // Wait and retry
                    await new Promise(resolve => setTimeout(resolve, 100));
                    continue;
                } else {
                    throw error;
                }
            }
        }

        throw new Error(`Failed to acquire lock after ${this.lockTimeout}ms: ${lockFile}`);
    }

    private async releaseLock(lock: FileLock): Promise<void> {
        try {
            await fs.unlink(lock.lockFile);
            this.activeLocks.delete(lock.path);
            logger.debug("[ConfigurationManager] Lock released", { path: lock.path });
        } catch (error) {
            logger.warn("[ConfigurationManager] Failed to release lock", {
                path: lock.path,
                error: error instanceof Error ? error.message : "Unknown error",
            });
        }
    }

    private async atomicWrite(filePath: string, config: TypedResourcesConfig): Promise<void> {
        // Create temporary file in the same directory to ensure atomic move
        const tempPath = `${filePath}.tmp.${randomBytes(TEMP_SUFFIX_LENGTH).toString("hex")}`;

        try {
            // Write to temporary file
            const content = JSON.stringify(config, null, 2);
            await fs.writeFile(tempPath, content, "utf8");

            // Atomic move
            await fs.rename(tempPath, filePath);

            logger.debug("[ConfigurationManager] Atomic write completed", { filePath });

        } catch (error) {
            // Clean up temporary file on error
            try {
                await fs.unlink(tempPath);
            } catch (cleanupError) {
                // Ignore cleanup errors
            }
            throw error;
        }
    }

    private applyUpdate(config: TypedResourcesConfig, update: ConfigurationUpdate): {
        config: TypedResourcesConfig;
        warnings?: string[];
    } {
        const warnings: string[] = [];
        const pathParts = update.path.split(".");
        let current: any = config;

        if (update.operation === "delete") {
            // Navigate to parent and delete the key
            for (let i = 0; i < pathParts.length - 1; i++) {
                if (!current[pathParts[i]]) {
                    warnings.push(`Path does not exist for deletion: ${update.path}`);
                    return { config, warnings };
                }
                current = current[pathParts[i]];
            }

            const lastKey = pathParts[pathParts.length - 1];
            if (current[lastKey] !== undefined) {
                delete current[lastKey];
            } else {
                warnings.push(`Key does not exist for deletion: ${update.path}`);
            }

        } else if (update.operation === "set") {
            // Navigate to parent and set the value
            for (let i = 0; i < pathParts.length - 1; i++) {
                if (!current[pathParts[i]]) {
                    current[pathParts[i]] = {};
                }
                current = current[pathParts[i]];
            }

            current[pathParts[pathParts.length - 1]] = update.value;

        } else if (update.operation === "merge") {
            // Navigate to the target and merge
            for (const part of pathParts) {
                if (!current[part]) {
                    current[part] = {};
                }
                current = current[part];
            }

            if (typeof current === "object" && typeof update.value === "object") {
                Object.assign(current, update.value);
            } else {
                warnings.push(`Cannot merge non-object values at path: ${update.path}`);
            }
        }

        return { config, warnings };
    }

    private expandEnvironmentVariables(config: any): any {
        if (typeof config === "string") {
            // Replace ${VAR_NAME} with environment variable values
            return config.replace(/\$\{([^}]+)\}/g, (match, varName) => {
                return process.env[varName] || match;
            });
        } else if (Array.isArray(config)) {
            return config.map(item => this.expandEnvironmentVariables(item));
        } else if (typeof config === "object" && config !== null) {
            const expanded: any = {};
            for (const [key, value] of Object.entries(config)) {
                expanded[key] = this.expandEnvironmentVariables(value);
            }
            return expanded;
        }

        return config;
    }

    private validateServiceConfig(serviceId: string, config: any, path: string, errors: ValidationError[]): void {
        if (typeof config !== "object" || config === null) {
            errors.push({
                path,
                message: "Service configuration must be an object",
            });
            return;
        }

        // Check required fields
        if (typeof config.enabled !== "boolean") {
            errors.push({
                path: `${path}.enabled`,
                message: "Service enabled field must be a boolean",
            });
        }

        if (config.baseUrl && typeof config.baseUrl !== "string") {
            errors.push({
                path: `${path}.baseUrl`,
                message: "Service baseUrl must be a string",
            });
        }

        // Validate health check configuration
        if (config.healthCheck) {
            if (typeof config.healthCheck !== "object") {
                errors.push({
                    path: `${path}.healthCheck`,
                    message: "Health check configuration must be an object",
                });
            } else {
                if (config.healthCheck.intervalMs && typeof config.healthCheck.intervalMs !== "number") {
                    errors.push({
                        path: `${path}.healthCheck.intervalMs`,
                        message: "Health check interval must be a number",
                    });
                }

                if (config.healthCheck.timeoutMs && typeof config.healthCheck.timeoutMs !== "number") {
                    errors.push({
                        path: `${path}.healthCheck.timeoutMs`,
                        message: "Health check timeout must be a number",
                    });
                }
            }
        }
    }

    private validateSecurityConfig(security: any, errors: ValidationError[]): void {
        if (typeof security !== "object" || security === null) {
            errors.push({
                path: "security",
                message: "Security configuration must be an object",
            });
            return;
        }

        // Validate network configuration
        if (security.network) {
            if (typeof security.network !== "object") {
                errors.push({
                    path: "security.network",
                    message: "Network security configuration must be an object",
                });
            } else {
                if (security.network.allowedHosts && !Array.isArray(security.network.allowedHosts)) {
                    errors.push({
                        path: "security.network.allowedHosts",
                        message: "Allowed hosts must be an array",
                    });
                }
            }
        }
    }

    private getDefaultConfiguration(): TypedResourcesConfig {
        return {
            version: "1.0.0",
            enabled: true,
            services: {
                ai: {},
                automation: {},
                agents: {},
                storage: {},
            },
            security: {
                network: {
                    allowedHosts: ["localhost", "127.0.0.1"],
                    allowCustomHosts: false,
                },
                authentication: {
                    keyPrefix: "VROOLI_RESOURCE_",
                },
            },
        };
    }
}
