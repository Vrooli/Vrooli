import { existsSync, readFileSync } from "fs";
import { resolve } from "path";
import { logger } from "../../events/logger.js";
import type {
    AgentResourceId,
    AIResourceId,
    AutomationResourceId,
    GetConfig,
    ResourceConfigMap,
    ResourceId,
    StorageResourceId,
} from "./typeRegistry.js";
import { isResourceInCategory } from "./typeRegistry.js";
import { ResourceCategory } from "./types.js";

/**
 * Type definitions matching the JSON schema
 */
export interface ResourcesConfig {
    version: string;
    enabled: boolean;
    services?: {
        ai?: Record<string, any>;
        automation?: Record<string, any>;
        agents?: Record<string, any>;
        storage?: Record<string, any>;
    };
    discovery?: {
        enabled: boolean;
        strategy: "automatic" | "manual" | "hybrid";
        methods?: {
            portScanning?: {
                enabled: boolean;
                scanIntervalMs?: number;
                ports?: Record<string, number>;
            };
            dockerIntegration?: {
                enabled: boolean;
                socketPath?: string;
                labelPrefix?: string;
            };
            processScanning?: {
                enabled: boolean;
                patterns?: string[];
            };
        };
        notifications?: {
            onDiscovery?: boolean;
            onLoss?: boolean;
        };
    };
    security?: {
        network?: {
            allowedHosts?: string[];
            allowCustomHosts?: boolean;
            enforceHttps?: boolean;
        };
        authentication?: {
            requireForAll?: boolean;
            keyStorage?: "environment" | "vault" | "file";
            keyPrefix?: string;
        };
        rateLimit?: {
            enabled?: boolean;
            requestsPerMinute?: number;
            burstSize?: number;
        };
        audit?: {
            enabled?: boolean;
            logLevel?: "error" | "warn" | "info" | "debug";
            retentionDays?: number;
        };
    };
    _documentation?: any;
}

/**
 * Fully typed version of ResourcesConfig with proper type safety for services
 * Uses the comprehensive type system from typeRegistry.ts
 */
export interface TypedResourcesConfig extends Omit<ResourcesConfig, "services" | "_documentation"> {
    services?: {
        ai?: Partial<Pick<ResourceConfigMap, AIResourceId>>;
        automation?: Partial<Pick<ResourceConfigMap, AutomationResourceId>>;
        agents?: Partial<Pick<ResourceConfigMap, AgentResourceId>>;
        storage?: Partial<Pick<ResourceConfigMap, StorageResourceId>>;
    };
    /** Documentation field for JSON configs - ignored during processing */
    _documentation?: Record<string, unknown>;
}

/**
 * Default configuration with full type safety
 */
const DEFAULT_CONFIG: TypedResourcesConfig = {
    version: "1.0.0",
    enabled: false,
    services: {
        ai: {},
        automation: {},
        agents: {},
        storage: {},
    },
    discovery: {
        enabled: false,
        strategy: "manual",
        methods: {
            portScanning: {
                enabled: false,
                scanIntervalMs: 3600000, // 1 hour
            },
        },
    },
    security: {
        network: {
            allowedHosts: ["localhost", "127.0.0.1"],
            allowCustomHosts: false,
            enforceHttps: false,
        },
        authentication: {
            requireForAll: false,
            keyStorage: "environment",
            keyPrefix: "VROOLI_RESOURCE_",
        },
        rateLimit: {
            enabled: true,
            requestsPerMinute: 100,
            burstSize: 20,
        },
        audit: {
            enabled: true,
            logLevel: "info",
            retentionDays: 30,
        },
    },
};

/**
 * Load resources configuration from JSON files with full type safety
 */
export async function loadResourcesConfig(customPath?: string): Promise<TypedResourcesConfig> {
    const projectRoot = process.cwd();
    const vrooliDir = resolve(projectRoot, ".vrooli");

    // Configuration file paths in order of precedence
    const configPaths = [
        customPath,
        resolve(vrooliDir, "resources.local.json"),
        resolve(vrooliDir, "resources.json"),
    ].filter(Boolean) as string[];

    let config: TypedResourcesConfig = JSON.parse(JSON.stringify(DEFAULT_CONFIG));
    let configLoaded = false;

    // Try to load configuration files
    for (const configPath of configPaths) {
        if (existsSync(configPath)) {
            try {
                const fileContent = readFileSync(configPath, "utf8");
                const fileConfig = JSON.parse(fileContent);

                // Merge with existing config (later files override earlier ones)
                config = mergeConfigs(config, fileConfig);
                configLoaded = true;

                logger.info(`[ResourcesConfig] Loaded configuration from: ${configPath}`);

                // If we loaded a local config, stop here (highest precedence)
                if (configPath.endsWith("resources.local.json")) {
                    break;
                }
            } catch (error) {
                logger.error(`[ResourcesConfig] Failed to load config from ${configPath}`, error);
            }
        }
    }

    if (!configLoaded) {
        logger.info("[ResourcesConfig] No configuration files found, using defaults");
    }

    // Apply environment variable substitutions
    config = substituteEnvironmentVariables(config);

    // Validate configuration
    validateConfig(config);

    return config;
}

/**
 * Deep merge two configuration objects
 */
function mergeConfigs(base: any, override: any): any {
    const result = { ...base };

    for (const key in override) {
        if (override[key] === null || override[key] === undefined) {
            continue;
        }

        if (key === "_documentation") {
            continue; // Skip documentation fields
        }

        if (typeof override[key] === "object" && !Array.isArray(override[key])) {
            result[key] = mergeConfigs(result[key] || {}, override[key]);
        } else {
            result[key] = override[key];
        }
    }

    return result;
}

/**
 * Substitute environment variables in configuration
 * Replaces ${VAR_NAME} with process.env.VAR_NAME
 */
function substituteEnvironmentVariables(config: any): any {
    if (typeof config === "string") {
        // Check for ${VAR_NAME} pattern
        const match = config.match(/^\$\{([A-Z_]+)\}$/);
        if (match) {
            const varName = match[1];
            const value = process.env[varName];
            if (!value) {
                logger.warn(`[ResourcesConfig] Environment variable ${varName} not set`);
            }
            return value || config;
        }
        return config;
    }

    if (Array.isArray(config)) {
        return config.map(item => substituteEnvironmentVariables(item));
    }

    if (typeof config === "object" && config !== null) {
        const result: any = {};
        for (const key in config) {
            result[key] = substituteEnvironmentVariables(config[key]);
        }
        return result;
    }

    return config;
}

/**
 * Validate configuration
 */
function validateConfig(config: ResourcesConfig): void {
    // Check version format
    if (!config.version.match(/^\d+\.\d+\.\d+$/)) {
        throw new Error(`Invalid version format: ${config.version}`);
    }

    // Validate discovery settings
    if (config.discovery?.enabled && !config.discovery.strategy) {
        throw new Error("Discovery enabled but no strategy specified");
    }

    // Validate security settings
    if (config.security?.network?.allowedHosts && config.security.network.allowedHosts.length === 0) {
        logger.warn("[ResourcesConfig] No allowed hosts specified, this may block all connections");
    }

    // Validate rate limiting
    if (config.security?.rateLimit?.enabled) {
        if (!config.security.rateLimit.requestsPerMinute || config.security.rateLimit.requestsPerMinute < 1) {
            throw new Error("Invalid rate limit configuration");
        }
    }
}

/**
 * Get a specific service configuration (legacy - untyped)
 */
export function getServiceConfig(
    config: TypedResourcesConfig,
    category: keyof NonNullable<TypedResourcesConfig["services"]>,
    serviceId: string,
): unknown {
    return config.services?.[category]?.[serviceId];
}

/**
 * Get a specific service configuration with full type safety
 */
export function getTypedServiceConfig<TId extends ResourceId>(
    config: TypedResourcesConfig,
    resourceId: TId,
): GetConfig<TId> | undefined {
    const category = getResourceCategory(resourceId);
    const categoryServices = config.services?.[category];
    if (!categoryServices) {
        return undefined;
    }
    return (categoryServices as any)[resourceId] as GetConfig<TId> | undefined;
}

/**
 * Helper function to get resource category from resource ID
 */
function getResourceCategory(resourceId: ResourceId): keyof NonNullable<TypedResourcesConfig["services"]> {
    // Use static import for proper category mapping
    if (isResourceInCategory(resourceId, ResourceCategory.AI)) {
        return "ai";
    }
    if (isResourceInCategory(resourceId, ResourceCategory.Automation)) {
        return "automation";
    }
    if (isResourceInCategory(resourceId, ResourceCategory.Agents)) {
        return "agents";
    }
    if (isResourceInCategory(resourceId, ResourceCategory.Storage)) {
        return "storage";
    }

    // This should never happen if typeRegistry is properly maintained
    throw new Error(`Unknown resource ID: ${resourceId}. Ensure it's defined in typeRegistry.ts`);
}

/**
 * Check if a host is allowed based on security configuration
 */
export function isHostAllowed(config: TypedResourcesConfig, host: string): boolean {
    const allowedHosts = config.security?.network?.allowedHosts || ["localhost", "127.0.0.1"];
    const allowCustom = config.security?.network?.allowCustomHosts || false;

    if (allowCustom) {
        return true;
    }

    return allowedHosts.includes(host);
}

/**
 * Get environment variable name for a service key
 */
export function getEnvVarName(config: TypedResourcesConfig, serviceId: string, keyName: string): string {
    const prefix = config.security?.authentication?.keyPrefix || "VROOLI_RESOURCE_";
    return `${prefix}${serviceId.toUpperCase()}_${keyName.toUpperCase()}`;
}
