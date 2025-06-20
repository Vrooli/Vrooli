import { type BaseConfigObject, ResourceUsedFor } from "../../../shape/configs/base.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";

/**
 * Common resource configurations used across different config types
 */
export const commonResources = {
    documentation: {
        link: "https://docs.vrooli.com",
        usedFor: ResourceUsedFor.OfficialWebsite,
        translations: [{
            language: "en",
            name: "Documentation",
            description: "Official documentation",
        }],
    },
    tutorial: {
        link: "https://vrooli.com/tutorial",
        usedFor: ResourceUsedFor.Tutorial,
        translations: [{
            language: "en",
            name: "Getting Started",
            description: "Interactive tutorial",
        }],
    },
    github: {
        link: "https://github.com/Vrooli/Vrooli",
        usedFor: ResourceUsedFor.Developer,
        translations: [{
            language: "en",
            name: "Source Code",
            description: "GitHub repository",
        }],
    },
    support: {
        link: "https://vrooli.com/support",
        usedFor: ResourceUsedFor.Community,
        translations: [{
            language: "en",
            name: "Help Center",
            description: "Get help and support",
        }],
    },
};

/**
 * Common translation patterns
 */
export const translationPatterns = {
    /**
     * Create translations for common languages
     */
    multiLanguage: (baseName: string, baseDescription = "") => [
        {
            language: "en",
            name: baseName,
            description: baseDescription,
        },
        {
            language: "es",
            name: `${baseName} (Spanish)`,
            description: baseDescription ? `${baseDescription} (Spanish)` : "",
        },
        {
            language: "fr",
            name: `${baseName} (French)`,
            description: baseDescription ? `${baseDescription} (French)` : "",
        },
    ],

    /**
     * Create a single English translation
     */
    english: (name: string, description = "") => [{
        language: "en",
        name,
        description,
    }],
};

/**
 * Common permission patterns
 */
export const permissionPatterns = {
    public: {
        read: true,
        write: false,
        delete: false,
    },
    authenticated: {
        read: true,
        write: true,
        delete: false,
    },
    owner: {
        read: true,
        write: true,
        delete: true,
    },
    admin: {
        read: true,
        write: true,
        delete: true,
        admin: true,
    },
};

/**
 * Common limit configurations
 */
export const limitPatterns = {
    free: {
        maxRequests: 100,
        maxTokens: 1000,
        maxStorage: 1024 * 1024 * 100, // 100MB
        maxCompute: 60, // 60 seconds
    },
    premium: {
        maxRequests: 1000,
        maxTokens: 10000,
        maxStorage: 1024 * 1024 * 1024, // 1GB
        maxCompute: 300, // 5 minutes
    },
    enterprise: {
        maxRequests: 10000,
        maxTokens: 100000,
        maxStorage: 1024 * 1024 * 1024 * 10, // 10GB
        maxCompute: 3600, // 1 hour
    },
};

/**
 * Create a base config with version
 */
export function createBaseConfig<T extends BaseConfigObject>(overrides: Omit<T, "__version"> = {} as Omit<T, "__version">): T {
    return {
        __version: LATEST_CONFIG_VERSION,
        ...overrides,
    } as T;
}

/**
 * Create an invalid config for testing
 */
export function createInvalidConfig<T extends BaseConfigObject>(
    type: "missingVersion" | "invalidVersion" | "wrongType",
    baseConfig: Partial<T> = {},
): any {
    switch (type) {
        case "missingVersion":
            const { __version, ...withoutVersion } = { ...baseConfig, __version: undefined };
            return withoutVersion;
        case "invalidVersion":
            return { ...baseConfig, __version: "0.0.1" };
        case "wrongType":
            return "not an object";
        default:
            return null;
    }
}

/**
 * Create config with resources
 */
export function withResources<T extends BaseConfigObject>(
    config: T,
    resources: Array<keyof typeof commonResources | typeof commonResources[keyof typeof commonResources]>,
): T {
    const resourceList = resources.map(r => {
        if (typeof r === "string" && r in commonResources) {
            return commonResources[r as keyof typeof commonResources];
        }
        return r;
    });

    return {
        ...config,
        resources: resourceList,
    } as T;
}

/**
 * Create timestamp fields
 */
export function withTimestamps(date: Date = new Date()) {
    return {
        createdAt: date.toISOString(),
        updatedAt: date.toISOString(),
    };
}

/**
 * Create schedule fields
 */
export function withSchedule(
    startTime: Date = new Date(),
    endTime?: Date,
    timezone = "UTC",
) {
    return {
        startTime: startTime.toISOString(),
        endTime: endTime?.toISOString() || null,
        timezone,
    };
}

/**
 * Merge configs with validation
 */
export function mergeWithValidation<T extends BaseConfigObject>(
    base: T,
    override: Partial<T>,
    validate?: (config: T) => boolean,
): T {
    const merged = {
        ...base,
        ...override,
        __version: base.__version, // Preserve version
    } as T;

    if (validate && !validate(merged)) {
        throw new Error("Merged config failed validation");
    }

    return merged;
}

/**
 * Create a factory method for a specific config type
 */
export function createConfigFactory<T extends BaseConfigObject>(
    minimal: T,
    defaults: Partial<T> = {},
) {
    return {
        create: (overrides: Partial<T> = {}): T => ({
            ...minimal,
            ...defaults,
            ...overrides,
            __version: LATEST_CONFIG_VERSION,
        } as T),

        createMany: (count: number, overrides: Partial<T> = {}): T[] => {
            return Array.from({ length: count }, (_, i) => ({
                ...minimal,
                ...defaults,
                ...overrides,
                __version: LATEST_CONFIG_VERSION,
                // Add index to any id field if it exists
                ...(("id" in minimal) ? { id: `${(minimal as any).id}_${i}` } : {}),
            } as T));
        },
    };
}

/**
 * Generate test scenarios for a config type
 */
export function generateTestScenarios<T extends BaseConfigObject>(
    baseConfig: T,
    scenarios: Record<string, Partial<T>>,
): Record<string, T> {
    const result: Record<string, T> = {};

    for (const [name, overrides] of Object.entries(scenarios)) {
        result[name] = {
            ...baseConfig,
            ...overrides,
        } as T;
    }

    return result;
}

/**
 * Create edge case configs for testing boundaries
 */
export const edgeCaseGenerators = {
    /**
     * Create config with maximum allowed values
     */
    maxValues: <T extends Record<string, any>>(fields: Record<keyof T, any>): Partial<T> => {
        const result: Partial<T> = {};
        for (const [key, value] of Object.entries(fields)) {
            if (typeof value === "number") {
                (result as any)[key] = Number.MAX_SAFE_INTEGER;
            } else if (typeof value === "string") {
                (result as any)[key] = "x".repeat(1000); // Long string
            } else if (Array.isArray(value)) {
                (result as any)[key] = Array(100).fill(value[0] || {}); // Large array
            } else {
                (result as any)[key] = value;
            }
        }
        return result;
    },

    /**
     * Create config with minimum allowed values
     */
    minValues: <T extends Record<string, any>>(fields: Record<keyof T, any>): Partial<T> => {
        const result: Partial<T> = {};
        for (const [key, value] of Object.entries(fields)) {
            if (typeof value === "number") {
                (result as any)[key] = 0;
            } else if (typeof value === "string") {
                (result as any)[key] = "";
            } else if (Array.isArray(value)) {
                (result as any)[key] = [];
            } else {
                (result as any)[key] = value;
            }
        }
        return result;
    },

    /**
     * Create config with null/undefined values where allowed
     */
    nullValues: <T extends Record<string, any>>(optionalFields: (keyof T)[]): Partial<T> => {
        const result: Partial<T> = {};
        for (const field of optionalFields) {
            (result as any)[field] = null;
        }
        return result;
    },
};
