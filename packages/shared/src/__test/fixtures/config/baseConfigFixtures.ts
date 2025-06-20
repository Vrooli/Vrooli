import { type BaseConfigObject, ResourceUsedFor } from "../../../shape/configs/base.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";

/**
 * Standard test fixture structure for config validation tests.
 * This ensures consistency across all config types.
 */
export interface ConfigTestFixtures<T extends BaseConfigObject = BaseConfigObject> {
    minimal: T;           // Minimal valid config
    complete: T;          // Fully populated config
    withDefaults: T;      // Config with defaults applied
    invalid: {
        missingVersion?: Partial<T>;
        invalidVersion?: T;
        malformedStructure?: any;
        invalidTypes?: Partial<T>;
    };
    variants: Record<string, T>;  // Different valid configurations
}

/**
 * Base config fixtures that can be extended by specific config types
 */
export const baseConfigFixtures: ConfigTestFixtures<BaseConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        resources: [
            {
                link: "https://example.com/docs",
                usedFor: ResourceUsedFor.OfficialWebsite,
                translations: [{
                    language: "en",
                    name: "Documentation",
                    description: "Official documentation",
                }],
            },
            {
                link: "https://github.com/example/repo",
                usedFor: ResourceUsedFor.Developer,
                translations: [{
                    language: "en",
                    name: "Source Code",
                    description: "GitHub repository",
                }],
            },
        ],
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        resources: [],
    },

    invalid: {
        missingVersion: {
            // Missing required __version field
            resources: [],
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            resources: [],
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            resources: "not an array", // Should be array
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            resources: [
                {
                    link: "123",
                    usedFor: ResourceUsedFor.Learning,
                    translations: [],
                },
            ],
        },
    },

    variants: {
        singleResource: {
            __version: LATEST_CONFIG_VERSION,
            resources: [{
                link: "https://example.com",
                usedFor: ResourceUsedFor.Learning,
                translations: [{
                    language: "en",
                    name: "Learning Resource",
                }],
            }],
        },

        multiLanguage: {
            __version: LATEST_CONFIG_VERSION,
            resources: [{
                link: "https://example.com/tutorial",
                usedFor: ResourceUsedFor.Tutorial,
                translations: [
                    {
                        language: "en",
                        name: "Tutorial",
                        description: "Step-by-step guide",
                    },
                    {
                        language: "es",
                        name: "Tutorial",
                        description: "Guía paso a paso",
                    },
                    {
                        language: "fr",
                        name: "Tutoriel",
                        description: "Guide étape par étape",
                    },
                ],
            }],
        },

        allResourceTypes: {
            __version: LATEST_CONFIG_VERSION,
            resources: Object.values(ResourceUsedFor).map((usedFor) => ({
                link: `https://example.com/${usedFor.toLowerCase()}`,
                usedFor,
                translations: [{
                    language: "en",
                    name: `${usedFor} Resource`,
                    description: `Resource for ${usedFor}`,
                }],
            })),
        },
    },
};

/**
 * Utility function to merge partial config with base defaults
 */
export function mergeWithBaseDefaults<T extends BaseConfigObject>(
    partial: Partial<T>,
    base: BaseConfigObject = baseConfigFixtures.minimal,
): T {
    return {
        ...base,
        ...partial,
        resources: partial.resources ?? base.resources,
    } as T;
}

/**
 * Create a config fixture with a specific resource type
 */
export function createConfigWithResource(
    usedFor: ResourceUsedFor,
    link = "https://example.com",
    name = "Test Resource",
): BaseConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        resources: [{
            link,
            usedFor,
            translations: [{
                language: "en",
                name,
            }],
        }],
    };
}
