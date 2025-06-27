/* c8 ignore start */
/**
 * Resource Version API Response Fixtures
 * 
 * Comprehensive API response fixtures for resource version endpoints, including
 * success responses, error scenarios, and MSW handlers for testing versioning workflows.
 */

import type { 
    ResourceVersion, 
    ResourceVersionCreateInput, 
    ResourceVersionUpdateInput,
    ResourceUsedFor,
    ResourceVersionTranslation,
    Resource,
    ResourceList,
} from "../../../api/types.js";
import { 
    ResourceUsedFor as ResourceUsedForEnum, 
} from "../../../run/enums.js";
import type { MockDataOptions } from "./types.js";
import { BaseAPIResponseFactory } from "./base.js";

// Constants for realistic data generation
const VERSION_LABELS = ["1.0.0", "1.1.0", "2.0.0", "2.1.0", "3.0.0-alpha", "3.0.0-beta", "3.0.0"] as const;
const DEFAULT_TITLE = "Resource Version";
const DEFAULT_DESCRIPTION = "A resource version for testing purposes";
const API_ENDPOINT = "https://api.example.com/v1";
const DOC_ENDPOINT = "https://docs.example.com";
const MINUTES_IN_MS = 60 * 1000;

/**
 * Factory for generating ResourceVersion API responses
 */
export class ResourceVersionAPIResponseFactory extends BaseAPIResponseFactory<
    ResourceVersion,
    ResourceVersionCreateInput,
    ResourceVersionUpdateInput
> {
    protected readonly entityName = "resource-version";

    /**
     * Generate a realistic version label
     */
    private generateVersionLabel(): string {
        return VERSION_LABELS[Math.floor(Math.random() * VERSION_LABELS.length)];
    }

    /**
     * Generate a realistic resource link based on usage type
     */
    private generateResourceLink(usedFor: ResourceUsedFor, id: string): string {
        switch (usedFor) {
            case ResourceUsedForEnum.Api:
                return `${API_ENDPOINT}/resource/${id}`;
            case ResourceUsedForEnum.Display:
                return `${DOC_ENDPOINT}/resource/${id}`;
            case ResourceUsedForEnum.Function:
                return `https://function.example.com/${id}`;
            case ResourceUsedForEnum.Plugin:
                return `https://plugin.example.com/${id}`;
            default:
                return `https://resource.example.com/${id}`;
        }
    }

    /**
     * Create mock resource (root resource)
     */
    private createMockResource(id?: string): Resource {
        const resourceId = id || this.generateId();
        const now = new Date().toISOString();
        
        return {
            __typename: "Resource",
            id: resourceId,
            createdAt: now,
            updatedAt: now,
            isInternal: false,
            isPrivate: false,
            usedBy: [],
            usedByCount: 0,
            versions: [],
            versionsCount: 1,
            you: {
                __typename: "ResourceYou",
                canDelete: true,
                canUpdate: true,
                canReport: false,
                isBookmarked: false,
                isReacted: false,
                reaction: null,
            },
        };
    }

    /**
     * Create mock resource list
     */
    private createMockResourceList(id?: string): ResourceList {
        const listId = id || this.generateId();
        const now = new Date().toISOString();
        
        return {
            __typename: "ResourceList",
            id: listId,
            createdAt: now,
            updatedAt: now,
            listFor: {
                __typename: "ApiVersion",
                id: `api_${listId}`,
                versionLabel: "1.0.0",
                createdAt: now,
                updatedAt: now,
            },
            resources: [],
            translations: [],
        };
    }

    /**
     * Create mock resource version data
     */
    createMockData(options?: MockDataOptions): ResourceVersion {
        const now = new Date().toISOString();
        const id = this.generateId();
        const usedFor = options?.overrides?.usedFor as ResourceUsedFor || ResourceUsedForEnum.Display;
        const versionLabel = options?.overrides?.versionLabel as string || this.generateVersionLabel();
        const isLatest = options?.overrides?.isLatest as boolean ?? true;
        const isPrivate = options?.overrides?.isPrivate as boolean ?? false;
        
        // Create default translation
        const title = options?.overrides?.title as string || `${DEFAULT_TITLE} ${versionLabel}`;
        const description = options?.overrides?.description as string || `${DEFAULT_DESCRIPTION} (${usedFor})`;
        
        const defaultTranslation: ResourceVersionTranslation = {
            __typename: "ResourceVersionTranslation",
            id: `trans_${id}`,
            language: "en",
            title,
            description,
        };

        const baseResourceVersion: ResourceVersion = {
            __typename: "ResourceVersion",
            id,
            createdAt: now,
            updatedAt: now,
            commentsCount: 0,
            directoryListingsCount: 0,
            forksCount: 0,
            isLatest,
            isPrivate,
            reportsCount: 0,
            versionIndex: parseInt(versionLabel.split(".")[0]) || 1,
            versionLabel,
            comments: [],
            translations: [defaultTranslation],
            translationsCount: 1,
            you: {
                __typename: "ResourceVersionYou",
                canComment: true,
                canDelete: true,
                canReport: false,
                canUpdate: true,
                canUse: true,
                canRead: true,
                isBookmarked: false,
                isReacted: false,
                reaction: null,
            },
        };

        // Add optional relationships based on scenario
        if (options?.withRelations !== false) {
            // Add root resource
            if (options?.includeOptional || options?.scenario === "complete") {
                baseResourceVersion.root = this.createMockResource();
            }
            
            // Add resource list for API resources
            if (usedFor === ResourceUsedForEnum.Api && options?.scenario !== "minimal") {
                baseResourceVersion.resourceList = this.createMockResourceList();
            }
        }

        // Apply scenario-specific overrides
        if (options?.scenario) {
            switch (options.scenario) {
                case "minimal":
                    baseResourceVersion.translations = [];
                    baseResourceVersion.translationsCount = 0;
                    baseResourceVersion.you.canDelete = false;
                    baseResourceVersion.you.canUpdate = false;
                    break;
                case "complete":
                    baseResourceVersion.commentsCount = 5;
                    baseResourceVersion.forksCount = 3;
                    baseResourceVersion.reportsCount = 1;
                    baseResourceVersion.directoryListingsCount = 2;
                    baseResourceVersion.isBookmarked = true;
                    break;
                case "edge-case":
                    baseResourceVersion.isLatest = false;
                    baseResourceVersion.isPrivate = true;
                    baseResourceVersion.versionLabel = "0.0.1-alpha";
                    baseResourceVersion.you.canUse = false;
                    break;
            }
        }

        // Apply explicit overrides
        if (options?.overrides) {
            Object.assign(baseResourceVersion, options.overrides);
        }

        return baseResourceVersion;
    }

    /**
     * Create entity from create input
     */
    createFromInput(input: ResourceVersionCreateInput): ResourceVersion {
        const resourceVersion = this.createMockData();
        
        // Update based on input
        if (input.id) resourceVersion.id = input.id;
        if (input.versionLabel) resourceVersion.versionLabel = input.versionLabel;
        if (input.isPrivate !== undefined) resourceVersion.isPrivate = input.isPrivate;
        
        // Handle translations
        if (input.translationsCreate && input.translationsCreate.length > 0) {
            resourceVersion.translations = input.translationsCreate.map(trans => ({
                __typename: "ResourceVersionTranslation" as const,
                id: trans.id,
                language: trans.language,
                title: trans.title || DEFAULT_TITLE,
                description: trans.description || DEFAULT_DESCRIPTION,
            }));
            resourceVersion.translationsCount = input.translationsCreate.length;
        }
        
        return resourceVersion;
    }

    /**
     * Update entity from update input
     */
    updateFromInput(existing: ResourceVersion, input: ResourceVersionUpdateInput): ResourceVersion {
        const updated = { ...existing };
        updated.updatedAt = new Date().toISOString();
        
        if (input.versionLabel !== undefined) updated.versionLabel = input.versionLabel;
        if (input.isPrivate !== undefined) updated.isPrivate = input.isPrivate;
        if (input.isLatest !== undefined) updated.isLatest = input.isLatest;
        
        // Handle translation updates
        if (input.translationsUpdate) {
            updated.translations = input.translationsUpdate.map(trans => ({
                __typename: "ResourceVersionTranslation" as const,
                id: trans.id,
                language: trans.language,
                title: trans.title || DEFAULT_TITLE,
                description: trans.description || DEFAULT_DESCRIPTION,
            }));
            updated.translationsCount = updated.translations.length;
        }
        
        return updated;
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ResourceVersionCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};
        
        if (!input.versionLabel?.trim()) {
            errors.versionLabel = "Version label is required";
        } else if (!/^\d+\.\d+\.\d+(\-[a-zA-Z0-9]+)?$/.test(input.versionLabel)) {
            errors.versionLabel = "Version label must follow semantic versioning (e.g., 1.0.0)";
        }
        
        if (!input.rootConnect) {
            errors.rootConnect = "Root resource connection is required";
        }
        
        if (!input.translationsCreate || input.translationsCreate.length === 0) {
            errors.translations = "At least one translation is required";
        } else {
            const englishTranslation = input.translationsCreate.find(t => t.language === "en");
            if (!englishTranslation || !englishTranslation.title?.trim()) {
                errors.translations = "English translation with title is required";
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
    async validateUpdateInput(input: ResourceVersionUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};
        
        if (!input.id) {
            errors.id = "Resource version ID is required for updates";
        }
        
        if (input.versionLabel && !/^\d+\.\d+\.\d+(\-[a-zA-Z0-9]+)?$/.test(input.versionLabel)) {
            errors.versionLabel = "Version label must follow semantic versioning (e.g., 1.0.0)";
        }
        
        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create resource versions for all usage types
     */
    createAllUsageTypeVersions(): ResourceVersion[] {
        return Object.values(ResourceUsedForEnum).map(usedFor => 
            this.createMockData({
                overrides: {
                    usedFor,
                    title: `${usedFor} Resource`,
                    description: `A resource version designed for ${usedFor.toLowerCase()} usage`,
                },
                withRelations: true,
            }),
        );
    }

    /**
     * Create version history for a resource
     */
    createVersionHistory(resourceId: string, versionCount = 5): ResourceVersion[] {
        const versions: ResourceVersion[] = [];
        
        for (let i = 0; i < versionCount; i++) {
            const isLatest = i === versionCount - 1;
            const versionIndex = i + 1;
            const versionLabel = `1.${i}.0`;
            
            versions.push(this.createMockData({
                overrides: {
                    versionIndex,
                    versionLabel,
                    isLatest,
                    root: { id: resourceId },
                },
                withRelations: true,
                scenario: isLatest ? "complete" : "minimal",
            }));
        }
        
        return versions;
    }

    /**
     * Create business error for version conflicts
     */
    createVersionConflictError(existingVersion: string, requestedVersion: string): any {
        return this.createBusinessErrorResponse("conflict", {
            reason: "Version already exists",
            existingVersion,
            requestedVersion,
            suggestion: "Use a different version number or update the existing version",
        });
    }

    /**
     * Create business error for invalid version transitions
     */
    createInvalidVersionTransitionError(fromVersion: string, toVersion: string): any {
        return this.createBusinessErrorResponse("workflow", {
            reason: "Invalid version transition",
            fromVersion,
            toVersion,
            validNextVersions: this.getValidNextVersions(fromVersion),
        });
    }

    /**
     * Get valid next versions for semantic versioning
     */
    private getValidNextVersions(currentVersion: string): string[] {
        const [major, minor, patch] = currentVersion.split(".").map(Number);
        if (isNaN(major) || isNaN(minor) || isNaN(patch)) return [];
        
        return [
            `${major}.${minor}.${patch + 1}`, // Patch increment
            `${major}.${minor + 1}.0`,        // Minor increment
            `${major + 1}.0.0`,               // Major increment
        ];
    }

    /**
     * Create business error for deprecated version access
     */
    createDeprecatedVersionError(version: string, replacementVersion?: string): any {
        return this.createBusinessErrorResponse("policy", {
            reason: "Version is deprecated and no longer accessible",
            deprecatedVersion: version,
            replacementVersion: replacementVersion || "latest",
            deprecationDate: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days ago
        });
    }
}

/**
 * Pre-configured response scenarios for common use cases
 */
export const resourceVersionResponseScenarios = {
    // Success scenarios
    createSuccess: (resourceVersion?: ResourceVersion) => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createSuccessResponse(
            resourceVersion || factory.createMockData(),
        );
    },

    listSuccess: (resourceVersions?: ResourceVersion[], pagination?: { page: number; pageSize: number; totalCount: number }) => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createPaginatedResponse(
            resourceVersions || factory.createAllUsageTypeVersions(),
            pagination || { page: 1, pageSize: 20, totalCount: 50 },
        );
    },

    apiResourceVersion: () => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: { usedFor: ResourceUsedForEnum.Api },
                scenario: "complete",
                withRelations: true,
            }),
        );
    },

    latestVersion: () => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: { 
                    isLatest: true,
                    versionLabel: "2.1.0",
                },
                scenario: "complete",
            }),
        );
    },

    versionHistory: (resourceId?: string) => {
        const factory = new ResourceVersionAPIResponseFactory();
        const versions = factory.createVersionHistory(resourceId || factory.generateId());
        return factory.createPaginatedResponse(
            versions,
            { page: 1, pageSize: versions.length, totalCount: versions.length },
        );
    },

    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                versionLabel: "Version label must follow semantic versioning",
                rootConnect: "Root resource connection is required",
                translations: "At least one translation is required",
            },
        );
    },

    notFoundError: (resourceVersionId?: string) => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createNotFoundErrorResponse(
            resourceVersionId || "non-existent-version-id",
            "resource version",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["resource:write"],
        );
    },

    versionConflictError: (existingVersion?: string, requestedVersion?: string) => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createVersionConflictError(
            existingVersion || "1.0.0",
            requestedVersion || "1.0.0",
        );
    },

    invalidVersionTransitionError: (fromVersion?: string, toVersion?: string) => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createInvalidVersionTransitionError(
            fromVersion || "1.0.0",
            toVersion || "0.9.0",
        );
    },

    deprecatedVersionError: (version?: string, replacement?: string) => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createDeprecatedVersionError(version || "0.5.0", replacement);
    },

    serverError: () => {
        const factory = new ResourceVersionAPIResponseFactory();
        return factory.createServerErrorResponse("resource-version-service", "create");
    },

    rateLimitError: () => {
        const factory = new ResourceVersionAPIResponseFactory();
        const resetTime = new Date(Date.now() + 5 * MINUTES_IN_MS); // 5 minutes from now
        return factory.createRateLimitErrorResponse(50, 0, resetTime);
    },
};

// Export factory instance for direct use
export const resourceVersionAPIResponseFactory = new ResourceVersionAPIResponseFactory();

