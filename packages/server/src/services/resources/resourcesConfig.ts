import { existsSync, readFileSync } from "fs";
import { resolve } from "path";
import { logger } from "../../events/logger.js";
import type {
    AgentResourceId,
    AIResourceId,
    AutomationResourceId,
    ExecutionResourceId,
    GetConfig,
    ResourceConfigMap,
    ResourceId,
    StorageResourceId,
} from "./typeRegistry.js";
import { isResourceInCategory } from "./typeRegistry.js";
import { ResourceCategory } from "./types.js";
import type { ExecutionConfig } from "../../tasks/sandbox/executionConfig.js";

/**
 * Security configuration interface for resources
 */
export interface ResourceSecurityConfig {
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
}

/**
 * Unified Service Configuration interface matching service.json schema
 */
export interface ServiceConfig {
    $schema?: string;
    version: string;
    service?: {
        name: string;
        displayName?: string;
        description?: string;
        version: string;
        type: "application" | "api" | "worker" | "cron" | "daemon" | "library" | "tool" | "platform";
        tags?: string[];
        maintainers?: Array<{
            name: string;
            email?: string;
            url?: string;
        }>;
        repository?: {
            type: "git" | "svn" | "mercurial" | "perforce";
            url: string;
            directory?: string;
        };
        license?: string;
        homepage?: string;
        documentation?: string;
        support?: {
            email?: string;
            url?: string;
            issues?: string;
            forum?: string;
            chat?: string;
        };
    };
    resources?: {
        ai?: Partial<Pick<ResourceConfigMap, AIResourceId>>;
        automation?: Partial<Pick<ResourceConfigMap, AutomationResourceId>>;
        agents?: Partial<Pick<ResourceConfigMap, AgentResourceId>>;
        storage?: Partial<Pick<ResourceConfigMap, StorageResourceId>>;
        execution?: Partial<Pick<ResourceConfigMap, ExecutionResourceId>>;
    };
    execution?: ExecutionConfig;
    scenarios?: Record<string, any>;
    serve?: Record<string, any>;
    inheritance?: {
        extends?: string | string[];
        overrides?: {
            resources?: boolean;
            execution?: boolean;
            scenarios?: boolean;
            serve?: boolean;
        };
        merge?: {
            strategy?: "replace" | "merge" | "append" | "prepend";
            arrays?: "replace" | "concat" | "union";
        };
    };
    lifecycle?: Record<string, any>;
    documentation?: Record<string, any>;
    /** Security configuration for resources */
    security?: ResourceSecurityConfig;
}


/**
 * Default service configuration
 */
const DEFAULT_SERVICE_CONFIG: ServiceConfig = {
    version: "1.0.0",
    service: {
        name: "vrooli",
        version: "1.0.0",
        type: "platform",
    },
    resources: {
        ai: {},
        automation: {},
        agents: {},
        storage: {},
        execution: {},
    },
};


/**
 * Load unified service configuration from service.json
 */
export async function loadServiceConfig(customPath?: string): Promise<ServiceConfig> {
    const projectRoot = process.env.PROJECT_DIR || process.cwd();
    const vrooliDir = resolve(projectRoot, ".vrooli");

    // Configuration file path
    const configPath = customPath || resolve(vrooliDir, "service.json");

    let config: ServiceConfig = JSON.parse(JSON.stringify(DEFAULT_SERVICE_CONFIG));
    let configLoaded = false;

    // Try to load service configuration
    if (existsSync(configPath)) {
        try {
            const fileContent = readFileSync(configPath, "utf8");
            const fileConfig = JSON.parse(fileContent);

            // Merge with default config
            config = mergeConfigs(config, fileConfig);
            configLoaded = true;

            logger.info(`[ServiceConfig] Loaded configuration from: ${configPath}`);
        } catch (error) {
            logger.error(`[ServiceConfig] Failed to load config from ${configPath}`, error);
        }
    }

    if (!configLoaded) {
        logger.info("[ServiceConfig] No service.json found, using defaults");
    }

    // Apply environment variable substitutions
    config = substituteEnvironmentVariables(config);

    // Validate configuration
    validateServiceConfig(config);

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
 * Validate service configuration
 */
function validateServiceConfig(config: ServiceConfig): void {
    // Check version format
    if (!config.version.match(/^\d+\.\d+\.\d+$/)) {
        throw new Error(`Invalid version format: ${config.version}`);
    }

    // Validate security settings
    if (config.security?.network?.allowedHosts && config.security.network.allowedHosts.length === 0) {
        logger.warn("[ServiceConfig] No allowed hosts specified, this may block all connections");
    }

    // Validate rate limiting
    if (config.security?.rateLimit?.enabled) {
        if (!config.security.rateLimit.requestsPerMinute || config.security.rateLimit.requestsPerMinute < 1) {
            throw new Error("Invalid rate limit configuration");
        }
    }
}

/**
 * Get a specific resource configuration from service config
 */
export function getResourceConfig(
    config: ServiceConfig,
    category: keyof NonNullable<ServiceConfig["resources"]>,
    resourceId: string,
): unknown {
    // Get from resources section
    return config.resources?.[category]?.[resourceId];
}

/**
 * Get a specific resource configuration with full type safety
 */
export function getTypedResourceConfig<TId extends ResourceId>(
    config: ServiceConfig,
    resourceId: TId,
): GetConfig<TId> | undefined {
    const category = getResourceCategory(resourceId);
    
    // Get from resources section
    const categoryResources = config.resources?.[category];
    if (categoryResources) {
        return (categoryResources as any)[resourceId] as GetConfig<TId> | undefined;
    }
    
    return undefined;
}

/**
 * Helper function to get resource category from resource ID
 */
function getResourceCategory(resourceId: ResourceId): keyof NonNullable<ServiceConfig["resources"]> {
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
    if (isResourceInCategory(resourceId, ResourceCategory.Execution)) {
        return "execution";
    }

    // This should never happen if typeRegistry is properly maintained
    throw new Error(`Unknown resource ID: ${resourceId}. Ensure it's defined in typeRegistry.ts`);
}

/**
 * Check if a host is allowed based on security configuration
 */
export function isHostAllowed(config: ServiceConfig, host: string): boolean {
    const allowedHosts = config.security?.network?.allowedHosts || ["localhost", "127.0.0.1"];
    const allowCustom = config.security?.network?.allowCustomHosts || false;

    if (allowCustom) {
        return true;
    }

    return allowedHosts.includes(host);
}

/**
 * Get environment variable name for a resource key
 */
export function getEnvVarName(config: ServiceConfig, resourceId: string, keyName: string): string {
    const prefix = config.security?.authentication?.keyPrefix || "VROOLI_RESOURCE_";
    return `${prefix}${resourceId.toUpperCase()}_${keyName.toUpperCase()}`;
}
