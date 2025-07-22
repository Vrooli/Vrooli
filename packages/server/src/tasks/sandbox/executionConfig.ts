import { CodeLanguage } from "@vrooli/shared";
import { readFileSync, existsSync } from "fs";
import { resolve } from "path";
import { logger } from "../../events/logger.js";

// Security limit constants
const MAX_TIMEOUT_MS = 600000; // 10 minutes
const MAX_MEMORY_LIMIT_MB = 4096; // 4GB  
const MAX_OUTPUT_BUFFER_KB = 10240; // 10MB

/**
 * Execution configuration interface matching the JSON schema
 */
export interface ExecutionConfig {
    enabled: boolean;
    allowedLanguages: CodeLanguage[];
    security: {
        allowedPaths: string[];
        deniedPaths: string[];
        maxTimeoutMs: number;
        maxMemoryLimitMb: number;
        maxOutputBufferKb: number;
        allowNetworkAccess: boolean;
        allowFileSystemWrites: boolean;
    };
    defaults: {
        timeoutMs: number;
        workingDirectory: string;
        memoryLimitMb: number;
    };
    runtime: {
        executables: {
            node: string;
            python3: string;
            bash: string;
        };
        environment: Record<string, string>;
    };
    logging: {
        enabled: boolean;
        level: "error" | "warn" | "info" | "debug";
        auditExecution: boolean;
    };
}

/**
 * Raw configuration from JSON file (before validation)
 */
interface RawExecutionConfig {
    enabled?: boolean;
    allowedLanguages?: string[];
    security?: {
        allowedPaths?: string[];
        deniedPaths?: string[];
        maxTimeoutMs?: number;
        maxMemoryLimitMb?: number;
        maxOutputBufferKb?: number;
        allowNetworkAccess?: boolean;
        allowFileSystemWrites?: boolean;
    };
    defaults?: {
        timeoutMs?: number;
        workingDirectory?: string;
        memoryLimitMb?: number;
    };
    runtime?: {
        executables?: {
            node?: string;
            python3?: string;
            bash?: string;
        };
        environment?: Record<string, string>;
    };
    logging?: {
        enabled?: boolean;
        level?: string;
        auditExecution?: boolean;
    };
    _documentation?: unknown; // Ignored field
}

/**
 * Default configuration values
 */
const DEFAULT_CONFIG: ExecutionConfig = {
    enabled: false,
    allowedLanguages: [CodeLanguage.Javascript],
    security: {
        allowedPaths: ["./"],
        deniedPaths: ["./node_modules/", "./.git/", "./data/", "./backups/"],
        maxTimeoutMs: 300000, // 5 minutes
        maxMemoryLimitMb: 1024, // 1GB
        maxOutputBufferKb: 2048, // 2MB
        allowNetworkAccess: false,
        allowFileSystemWrites: false,
    },
    defaults: {
        timeoutMs: 30000, // 30 seconds
        workingDirectory: "./",
        memoryLimitMb: 512, // 512MB
    },
    runtime: {
        executables: {
            node: "node",
            python3: "python3",
            bash: "bash",
        },
        environment: {
            NODE_ENV: "development",
        },
    },
    logging: {
        enabled: true,
        level: "info",
        auditExecution: true,
    },
};

/**
 * Map string language names to CodeLanguage enum
 */
function mapLanguageString(lang: string): CodeLanguage | null {
    switch (lang.toLowerCase()) {
        case "javascript":
        case "js":
            return CodeLanguage.Javascript;
        case "shell":
        case "bash":
            return CodeLanguage.Shell;
        case "python":
        case "py":
            return CodeLanguage.Python;
        default:
            return null;
    }
}

/**
 * Validate and normalize configuration
 */
function validateConfig(raw: RawExecutionConfig): ExecutionConfig {
    const config: ExecutionConfig = JSON.parse(JSON.stringify(DEFAULT_CONFIG));

    // Enabled flag
    if (typeof raw.enabled === "boolean") {
        config.enabled = raw.enabled;
    }

    // Allowed languages
    if (Array.isArray(raw.allowedLanguages)) {
        const mappedLanguages: CodeLanguage[] = [];
        for (const lang of raw.allowedLanguages) {
            if (typeof lang === "string") {
                const mapped = mapLanguageString(lang);
                if (mapped && !mappedLanguages.includes(mapped)) {
                    mappedLanguages.push(mapped);
                }
            }
        }
        if (mappedLanguages.length > 0) {
            config.allowedLanguages = mappedLanguages;
        }
    }

    // Security settings
    if (raw.security) {
        if (Array.isArray(raw.security.allowedPaths)) {
            config.security.allowedPaths = raw.security.allowedPaths.filter(p => typeof p === "string");
        }
        if (Array.isArray(raw.security.deniedPaths)) {
            config.security.deniedPaths = raw.security.deniedPaths.filter(p => typeof p === "string");
        }
        if (typeof raw.security.maxTimeoutMs === "number" && raw.security.maxTimeoutMs > 0) {
            config.security.maxTimeoutMs = Math.min(raw.security.maxTimeoutMs, MAX_TIMEOUT_MS);
        }
        if (typeof raw.security.maxMemoryLimitMb === "number" && raw.security.maxMemoryLimitMb > 0) {
            config.security.maxMemoryLimitMb = Math.min(raw.security.maxMemoryLimitMb, MAX_MEMORY_LIMIT_MB);
        }
        if (typeof raw.security.maxOutputBufferKb === "number" && raw.security.maxOutputBufferKb > 0) {
            config.security.maxOutputBufferKb = Math.min(raw.security.maxOutputBufferKb, MAX_OUTPUT_BUFFER_KB);
        }
        if (typeof raw.security.allowNetworkAccess === "boolean") {
            config.security.allowNetworkAccess = raw.security.allowNetworkAccess;
        }
        if (typeof raw.security.allowFileSystemWrites === "boolean") {
            config.security.allowFileSystemWrites = raw.security.allowFileSystemWrites;
        }
    }

    // Default settings
    if (raw.defaults) {
        if (typeof raw.defaults.timeoutMs === "number" && raw.defaults.timeoutMs > 0) {
            config.defaults.timeoutMs = Math.min(raw.defaults.timeoutMs, config.security.maxTimeoutMs);
        }
        if (typeof raw.defaults.workingDirectory === "string") {
            config.defaults.workingDirectory = raw.defaults.workingDirectory;
        }
        if (typeof raw.defaults.memoryLimitMb === "number" && raw.defaults.memoryLimitMb > 0) {
            config.defaults.memoryLimitMb = Math.min(raw.defaults.memoryLimitMb, config.security.maxMemoryLimitMb);
        }
    }

    // Runtime settings
    if (raw.runtime) {
        if (raw.runtime.executables) {
            if (typeof raw.runtime.executables.node === "string") {
                config.runtime.executables.node = raw.runtime.executables.node;
            }
            if (typeof raw.runtime.executables.python3 === "string") {
                config.runtime.executables.python3 = raw.runtime.executables.python3;
            }
            if (typeof raw.runtime.executables.bash === "string") {
                config.runtime.executables.bash = raw.runtime.executables.bash;
            }
        }
        if (raw.runtime.environment && typeof raw.runtime.environment === "object") {
            config.runtime.environment = { ...config.runtime.environment, ...raw.runtime.environment };
        }
    }

    // Logging settings
    if (raw.logging) {
        if (typeof raw.logging.enabled === "boolean") {
            config.logging.enabled = raw.logging.enabled;
        }
        if (typeof raw.logging.level === "string" && 
            ["error", "warn", "info", "debug"].includes(raw.logging.level)) {
            config.logging.level = raw.logging.level as ExecutionConfig["logging"]["level"];
        }
        if (typeof raw.logging.auditExecution === "boolean") {
            config.logging.auditExecution = raw.logging.auditExecution;
        }
    }

    return config;
}

/**
 * Load configuration from file
 */
function loadConfigFromFile(filePath: string): RawExecutionConfig | null {
    try {
        if (!existsSync(filePath)) {
            return null;
        }
        const content = readFileSync(filePath, "utf-8");
        return JSON.parse(content) as RawExecutionConfig;
    } catch (error) {
        logger.warn(`Failed to load execution config from ${filePath}`, { 
            error: error instanceof Error ? error.message : String(error),
            trace: "loadConfigFromFile",
        });
        return null;
    }
}

/**
 * Load execution configuration with proper precedence:
 * 1. .vrooli/execution.local.json (local overrides)
 * 2. .vrooli/execution.json (shared base config)
 * 3. Environment variables (backward compatibility)
 * 4. Built-in defaults
 */
export function loadExecutionConfig(projectRoot?: string): ExecutionConfig {
    const root = projectRoot || process.env.PROJECT_DIR || process.cwd();
    
    // Load local config (highest precedence)
    const localConfigPath = resolve(root, ".vrooli/execution.local.json");
    const localConfig = loadConfigFromFile(localConfigPath);
    
    // Load shared config (fallback)
    const sharedConfigPath = resolve(root, ".vrooli/execution.json");
    const sharedConfig = loadConfigFromFile(sharedConfigPath);
    
    // Merge configurations with precedence
    const mergedRaw: RawExecutionConfig = {
        ...sharedConfig,
        ...localConfig,
    };
    
    // Apply environment variable overrides for backward compatibility
    const envEnabled = process.env.VROOLI_ENABLE_LOCAL_EXECUTION === "true";
    const isDevOrTest = process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";
    
    // If environment variables suggest enabling, but no config files exist, provide warning
    if (envEnabled && isDevOrTest && !localConfig && !sharedConfig) {
        logger.warn("Local execution enabled via environment variable but no configuration file found", {
            suggestion: "Create .vrooli/execution.local.json with enabled: true",
            trace: "loadExecutionConfig-env-fallback",
        });
        mergedRaw.enabled = true;
    }
    
    // Validate and normalize
    const config = validateConfig(mergedRaw);
    
    if (config.logging.enabled) {
        logger.info("Loaded execution configuration", {
            enabled: config.enabled,
            allowedLanguages: config.allowedLanguages,
            configSource: localConfig ? "local" : (sharedConfig ? "shared" : "default"),
            trace: "loadExecutionConfig-success",
        });
    }
    
    return config;
}

/**
 * Generate helpful error message when local execution is disabled
 */
export function generateConfigurationErrorMessage(): string {
    const projectRoot = process.cwd();
    const localConfigPath = resolve(projectRoot, ".vrooli/execution.local.json");
    const exampleConfigPath = resolve(projectRoot, ".vrooli/execution.example.json");
    
    let message = "Local code execution is disabled. To enable:\n\n";
    
    if (existsSync(exampleConfigPath)) {
        message += `1. Copy ${exampleConfigPath} to ${localConfigPath}\n`;
        message += `2. Edit ${localConfigPath} and set "enabled": true\n`;
        message += "3. Add required languages to the \"allowedLanguages\" array\n";
        message += "4. Configure \"allowedPaths\" for your security needs\n\n";
    } else {
        message += `1. Create ${localConfigPath}\n`;
        message += "2. Set minimum configuration: {\"enabled\": true, \"allowedLanguages\": [\"shell\"]}\n\n";
    }
    
    message += "For security, local execution requires both:\n";
    message += "- Configuration file with enabled: true\n";
    message += "- Development or test environment (NODE_ENV)\n\n";
    message += "See .vrooli/execution.example.json for full configuration options.";
    
    return message;
}

/**
 * Singleton configuration instance
 */
let cachedConfig: ExecutionConfig | null = null;

/**
 * Get the current execution configuration (cached)
 */
export function getExecutionConfig(): ExecutionConfig {
    if (!cachedConfig) {
        cachedConfig = loadExecutionConfig();
    }
    return cachedConfig;
}

/**
 * Reload configuration (for testing or config changes)
 */
export function reloadExecutionConfig(): ExecutionConfig {
    cachedConfig = null;
    return getExecutionConfig();
}

