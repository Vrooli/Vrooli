import { type BaseConfigObject } from "../../../shape/configs/base.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";

/**
 * Validation result for config objects
 */
export interface ValidationResult<T = unknown> {
    isValid: boolean;
    errors?: string[];
    data?: T;
}

/**
 * Format types for config transformation
 */
export type ApiConfigFormat = Record<string, unknown>; // Would be more specific in real implementation
export type DbConfigFormat = string; // JSON string for database storage

/**
 * Enhanced factory interface for config fixtures
 * Implements the ideal architecture described in the README
 */
export interface ConfigFixtureFactory<TConfig extends BaseConfigObject> {
    // Core fixtures
    minimal: TConfig;
    complete: TConfig;
    withDefaults: TConfig;

    // Variant collections
    variants: {
        [key: string]: TConfig;
    };

    // Invalid configurations for validation testing
    invalid: {
        missingVersion?: Partial<TConfig>;
        invalidVersion?: TConfig;
        malformedStructure?: unknown;
        invalidTypes?: Partial<TConfig>;
        [key: string]: unknown; // Allow additional invalid scenarios
    };

    // Factory methods
    create: (overrides?: Partial<TConfig>) => TConfig;
    createVariant: (variant: keyof ConfigFixtureFactory<TConfig>["variants"], overrides?: Partial<TConfig>) => TConfig;

    // Validation methods
    validate: (config: unknown) => ValidationResult<TConfig>;
    isValid: (config: unknown) => config is TConfig;

    // Composition helpers
    merge: (base: TConfig, override: Partial<TConfig>) => TConfig;
    applyDefaults: (partialConfig: Partial<TConfig>) => TConfig;

    // Integration helpers
    toApiFormat: () => ApiConfigFormat;
    toDbFormat: () => DbConfigFormat;
    fromJson: (json: string) => TConfig;
    toJson: (config: TConfig) => string;
}

/**
 * Base factory class that implements common patterns
 */
export abstract class BaseConfigFactory<TConfig extends BaseConfigObject> implements ConfigFixtureFactory<TConfig> {
    abstract minimal: TConfig;
    abstract complete: TConfig;
    abstract withDefaults: TConfig;
    abstract variants: { [key: string]: TConfig };
    abstract invalid: {
        missingVersion?: Partial<TConfig>;
        invalidVersion?: TConfig;
        malformedStructure?: unknown;
        invalidTypes?: Partial<TConfig>;
        [key: string]: unknown;
    };

    /**
     * Create a config with optional overrides
     */
    create(overrides: Partial<TConfig> = {}): TConfig {
        return this.merge(this.minimal, overrides);
    }

    /**
     * Create a specific variant with optional overrides
     */
    createVariant(variant: keyof ConfigFixtureFactory<TConfig>["variants"], overrides: Partial<TConfig> = {}): TConfig {
        const base = this.variants[variant as string];
        if (!base) {
            throw new Error(`Unknown variant: ${String(variant)}`);
        }
        return this.merge(base, overrides);
    }

    /**
     * Validate a config object
     * Override this method in specific factories for custom validation
     */
    validate(config: unknown): ValidationResult<TConfig> {
        // Basic validation - check for required __version field
        if (!config || typeof config !== "object") {
            return {
                isValid: false,
                errors: ["Config must be an object"],
            };
        }

        const configObj = config as Record<string, unknown>;

        if (!configObj.__version) {
            return {
                isValid: false,
                errors: ["Missing required __version field"],
            };
        }

        if (configObj.__version !== LATEST_CONFIG_VERSION) {
            return {
                isValid: false,
                errors: [`Invalid version: ${configObj.__version}`],
            };
        }

        return {
            isValid: true,
            data: config as TConfig,
        };
    }

    /**
     * Type guard to check if value is a valid config
     */
    isValid(config: unknown): config is TConfig {
        return this.validate(config).isValid;
    }

    /**
     * Deep merge two config objects
     */
    merge(base: TConfig, override: Partial<TConfig>): TConfig {
        return deepMerge(base, override);
    }

    /**
     * Apply defaults to a partial config
     */
    applyDefaults(partialConfig: Partial<TConfig>): TConfig {
        return this.merge(this.withDefaults, partialConfig);
    }

    /**
     * Transform to API format
     * Override in specific factories if needed
     */
    toApiFormat(): ApiConfigFormat {
        return this.complete;
    }

    /**
     * Transform to database format
     * Override in specific factories if needed
     */
    toDbFormat(): DbConfigFormat {
        return JSON.stringify(this.complete);
    }

    /**
     * Parse config from JSON string
     */
    fromJson(json: string): TConfig {
        try {
            const parsed = JSON.parse(json);
            const validation = this.validate(parsed);
            if (!validation.isValid) {
                throw new Error(`Invalid config: ${validation.errors?.join(", ")}`);
            }
            if (!validation.data) {
                throw new Error("Validation succeeded but no data returned");
            }
            return validation.data;
        } catch (error) {
            throw new Error(`Failed to parse config JSON: ${error instanceof Error ? error.message : "Unknown error"}`);
        }
    }

    /**
     * Convert config to JSON string
     */
    toJson(config: TConfig): string {
        return JSON.stringify(config, null, 2);
    }
}

/**
 * Deep merge utility function
 * Handles nested objects and arrays
 */
export function deepMerge<T extends Record<string, unknown>>(target: T, source: Partial<T>): T {
    const result = { ...target };

    for (const key in source) {
        if (Object.prototype.hasOwnProperty.call(source, key)) {
            const sourceValue = source[key];
            const targetValue = result[key];

            if (sourceValue === undefined) {
                continue;
            }

            if (sourceValue === null) {
                result[key] = null as T[Extract<keyof T, string>];
            } else if (Array.isArray(sourceValue)) {
                result[key] = [...sourceValue] as T[Extract<keyof T, string>];
            } else if (
                typeof sourceValue === "object" &&
                sourceValue !== null &&
                sourceValue.constructor === Object
            ) {
                if (targetValue && typeof targetValue === "object" && !Array.isArray(targetValue)) {
                    result[key] = deepMerge(targetValue as Record<string, unknown>, sourceValue as Record<string, unknown>) as T[Extract<keyof T, string>];
                } else {
                    result[key] = deepMerge({}, sourceValue as Record<string, unknown>) as T[Extract<keyof T, string>];
                }
            } else {
                result[key] = sourceValue as T[Extract<keyof T, string>];
            }
        }
    }

    return result;
}

/**
 * Validation helper functions
 */
export const validationHelpers = {
    /**
     * Check if a value is a valid enum value
     */
    isValidEnum<T extends Record<string, string>>(value: unknown, enumObj: T): boolean {
        return Object.values(enumObj).includes(value as string);
    },

    /**
     * Check if a value is within a numeric range
     */
    isInRange(value: number, min: number, max: number): boolean {
        return value >= min && value <= max;
    },

    /**
     * Check if a string matches a pattern
     */
    matchesPattern(value: string, pattern: RegExp): boolean {
        return pattern.test(value);
    },

    /**
     * Check if an array has valid length
     */
    hasValidLength<T>(array: T[], min: number, max?: number): boolean {
        return array.length >= min && (max === undefined || array.length <= max);
    },
};

/**
 * Composition helper functions
 */
export const compositionHelpers = {
    /**
     * Create a config by extending a base with specific fields
     */
    extend<T extends BaseConfigObject>(base: T, fields: Partial<T>): T {
        return { ...base, ...fields };
    },

    /**
     * Create a config by picking specific fields from a source
     */
    pick<T extends BaseConfigObject, K extends keyof T>(source: T, keys: K[]): Pick<T, K> & { __version: string } {
        const result = { __version: source.__version } as Pick<T, K> & { __version: string };
        for (const key of keys) {
            if (key in source) {
                (result as Record<string, unknown>)[key] = source[key];
            }
        }
        return result;
    },

    /**
     * Create a config by omitting specific fields from a source
     */
    omit<T extends BaseConfigObject, K extends keyof T>(source: T, keys: K[]): Omit<T, K> & { __version: string } {
        const result = { ...source };
        for (const key of keys) {
            delete (result as Record<string, unknown>)[key];
        }
        return result as Omit<T, K> & { __version: string };
    },
};
