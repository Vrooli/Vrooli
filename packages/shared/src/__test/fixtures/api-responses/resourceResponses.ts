/* c8 ignore start */
/**
 * Resource API Response Fixtures
 * 
 * Comprehensive fixtures for resource management endpoints including
 * resource creation, version control, and access management.
 */

import type {
    Resource,
    ResourceCreateInput,
    ResourceUpdateInput,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_DESCRIPTION_LENGTH = 2000;
const MIN_NAME_LENGTH = 2;

/**
 * Resource API response factory
 */
export class ResourceResponseFactory extends BaseAPIResponseFactory<
    Resource,
    ResourceCreateInput,
    ResourceUpdateInput
> {
    protected readonly entityName = "resource";

    /**
     * Create mock resource data
     */
    createMockData(options?: MockDataOptions): Resource {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const resourceId = options?.overrides?.id || generatePK().toString();

        const baseResource: Resource = {
            __typename: "Resource",
            id: resourceId,
            created_at: now,
            updated_at: now,
            isInternal: false,
            isPrivate: false,
            usedBy: [],
            usedByCount: 0,
            versions: [],
            versionsCount: 0,
            bookmarks: 0,
            reportsReceivedCount: 0,
            you: {
                canDelete: false,
                canReport: true,
                canUpdate: false,
                isBookmarked: false,
                isViewed: false,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            // Create a comprehensive resource with versions and translations
            return {
                ...baseResource,
                isInternal: false,
                isPrivate: false,
                usedByCount: 5,
                versionsCount: 3,
                bookmarks: 12,
                versions: [{
                    __typename: "ResourceVersion",
                    id: generatePK().toString(),
                    created_at: now,
                    updated_at: now,
                    isLatest: true,
                    isPrivate: false,
                    versionIndex: 0,
                    versionLabel: "1.0.0",
                    translations: [{
                        __typename: "ResourceVersionTranslation",
                        id: generatePK().toString(),
                        language: "en",
                        name: "Sample Resource",
                        description: "A comprehensive example resource for testing",
                    }],
                    you: {
                        canDelete: true,
                        canReport: false,
                        canUpdate: true,
                        isBookmarked: false,
                        isViewed: true,
                    },
                }],
                you: {
                    canDelete: true,
                    canReport: false,
                    canUpdate: true,
                    isBookmarked: false,
                    isViewed: true,
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseResource,
            ...options?.overrides,
        };
    }

    /**
     * Create resource from input
     */
    createFromInput(input: ResourceCreateInput): Resource {
        const now = new Date().toISOString();
        const resourceId = generatePK().toString();

        return {
            __typename: "Resource",
            id: resourceId,
            created_at: now,
            updated_at: now,
            isInternal: input.isInternal || false,
            isPrivate: input.isPrivate || false,
            usedBy: [],
            usedByCount: 0,
            versions: input.versionsCreate?.map((version, index) => ({
                __typename: "ResourceVersion" as const,
                id: generatePK().toString(),
                created_at: now,
                updated_at: now,
                isLatest: index === 0,
                isPrivate: version.isPrivate || false,
                versionIndex: index,
                versionLabel: version.versionLabel || `1.${index}.0`,
                translations: version.translationsCreate?.map(t => ({
                    __typename: "ResourceVersionTranslation" as const,
                    id: generatePK().toString(),
                    language: t.language,
                    name: t.name || "",
                    description: t.description || null,
                })) || [],
                you: {
                    canDelete: true,
                    canReport: false,
                    canUpdate: true,
                    isBookmarked: false,
                    isViewed: false,
                },
            })) || [],
            versionsCount: input.versionsCreate?.length || 0,
            bookmarks: 0,
            reportsReceivedCount: 0,
            you: {
                canDelete: true,
                canReport: false,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
            },
        };
    }

    /**
     * Update resource from input
     */
    updateFromInput(existing: Resource, input: ResourceUpdateInput): Resource {
        const updates: Partial<Resource> = {
            updated_at: new Date().toISOString(),
        };

        if (input.isInternal !== undefined) updates.isInternal = input.isInternal;
        if (input.isPrivate !== undefined) updates.isPrivate = input.isPrivate;

        // Handle version updates
        if (input.versionsUpdate) {
            updates.versions = existing.versions?.map(version => {
                const update = input.versionsUpdate?.find(u => u.id === version.id);
                if (update) {
                    return {
                        ...version,
                        updated_at: new Date().toISOString(),
                        isPrivate: update.isPrivate ?? version.isPrivate,
                        versionLabel: update.versionLabel ?? version.versionLabel,
                    };
                }
                return version;
            });
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ResourceCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.versionsCreate || input.versionsCreate.length === 0) {
            errors.versions = "At least one version is required";
        }

        // Validate first version
        if (input.versionsCreate && input.versionsCreate.length > 0) {
            const firstVersion = input.versionsCreate[0];
            if (!firstVersion.translationsCreate || firstVersion.translationsCreate.length === 0) {
                errors["versions.0.translations"] = "At least one translation is required";
            } else {
                const firstTranslation = firstVersion.translationsCreate[0];
                if (!firstTranslation.name) {
                    errors["versions.0.translations.0.name"] = "Name is required";
                } else if (firstTranslation.name.length < MIN_NAME_LENGTH) {
                    errors["versions.0.translations.0.name"] = "Name must be at least 2 characters";
                }

                if (firstTranslation.description && firstTranslation.description.length > MAX_DESCRIPTION_LENGTH) {
                    errors["versions.0.translations.0.description"] = "Description must be 2000 characters or less";
                }
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ResourceUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        // Validate version updates if provided
        if (input.versionsUpdate) {
            input.versionsUpdate.forEach((version, index) => {
                if (version.versionLabel && version.versionLabel.length < 1) {
                    errors[`versions.${index}.versionLabel`] = "Version label cannot be empty";
                }
            });
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create resource with multiple versions
     */
    createResourceWithVersions(versionCount = 3): Resource {
        const resource = this.createMockData({ scenario: "complete" });
        const now = new Date().toISOString();
        const versions = [];

        for (let i = 0; i < versionCount; i++) {
            versions.push({
                __typename: "ResourceVersion" as const,
                id: generatePK().toString(),
                created_at: now,
                updated_at: now,
                isLatest: i === 0,
                isPrivate: false,
                versionIndex: i,
                versionLabel: `1.${i}.0`,
                translations: [{
                    __typename: "ResourceVersionTranslation" as const,
                    id: generatePK().toString(),
                    language: "en",
                    name: `Resource Version ${i + 1}`,
                    description: `Version ${i + 1} of the resource with improvements`,
                }],
                you: {
                    canDelete: i === 0, // Can only delete latest
                    canReport: false,
                    canUpdate: i === 0, // Can only update latest
                    isBookmarked: false,
                    isViewed: i === 0,
                },
            });
        }

        return {
            ...resource,
            versions,
            versionsCount: versionCount,
        };
    }

    /**
     * Create private resource
     */
    createPrivateResource(): Resource {
        return this.createMockData({
            scenario: "complete",
            overrides: {
                isPrivate: true,
                you: {
                    canDelete: true,
                    canReport: false,
                    canUpdate: true,
                    isBookmarked: false,
                    isViewed: true,
                },
            },
        });
    }

    /**
     * Create internal resource
     */
    createInternalResource(): Resource {
        return this.createMockData({
            scenario: "complete",
            overrides: {
                isInternal: true,
                isPrivate: true,
                you: {
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isViewed: true,
                },
            },
        });
    }
}

/**
 * Pre-configured resource response scenarios
 */
export const resourceResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ResourceCreateInput>) => {
        const factory = new ResourceResponseFactory();
        const defaultInput: ResourceCreateInput = {
            isInternal: false,
            isPrivate: false,
            versionsCreate: [{
                isPrivate: false,
                versionLabel: "1.0.0",
                translationsCreate: [{
                    language: "en",
                    name: "Test Resource",
                    description: "A test resource for development",
                }],
            }],
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (resource?: Resource) => {
        const factory = new ResourceResponseFactory();
        return factory.createSuccessResponse(
            resource || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ResourceResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Resource, updates?: Partial<ResourceUpdateInput>) => {
        const factory = new ResourceResponseFactory();
        const resource = existing || factory.createMockData({ scenario: "complete" });
        const input: ResourceUpdateInput = {
            id: resource.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(resource, input),
        );
    },

    withVersionsSuccess: (versionCount?: number) => {
        const factory = new ResourceResponseFactory();
        return factory.createSuccessResponse(
            factory.createResourceWithVersions(versionCount),
        );
    },

    privateResourceSuccess: () => {
        const factory = new ResourceResponseFactory();
        return factory.createSuccessResponse(
            factory.createPrivateResource(),
        );
    },

    internalResourceSuccess: () => {
        const factory = new ResourceResponseFactory();
        return factory.createSuccessResponse(
            factory.createInternalResource(),
        );
    },

    listSuccess: (resources?: Resource[]) => {
        const factory = new ResourceResponseFactory();
        return factory.createPaginatedResponse(
            resources || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: resources?.length || DEFAULT_COUNT },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ResourceResponseFactory();
        return factory.createValidationErrorResponse({
            versions: "At least one version is required",
            "versions.0.translations.0.name": "Name is required",
        });
    },

    notFoundError: (resourceId?: string) => {
        const factory = new ResourceResponseFactory();
        return factory.createNotFoundErrorResponse(
            resourceId || "non-existent-resource",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ResourceResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["resource:update"],
        );
    },

    versionLimitError: () => {
        const factory = new ResourceResponseFactory();
        const VERSION_LIMIT = 50;
        return factory.createBusinessErrorResponse("limit", {
            resource: "versions",
            limit: VERSION_LIMIT,
            current: VERSION_LIMIT,
            message: "Resource has reached the maximum number of versions",
        });
    },

    resourceInUseError: () => {
        const factory = new ResourceResponseFactory();
        return factory.createBusinessErrorResponse("state", {
            reason: "Resource is currently in use",
            message: "Cannot delete resource that is being used by active routines",
            usedByCount: 3,
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ResourceResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ResourceResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ResourceResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const resourceResponseFactory = new ResourceResponseFactory();
