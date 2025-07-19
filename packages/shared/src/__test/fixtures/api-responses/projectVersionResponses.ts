/* c8 ignore start */
/**
 * ProjectVersion API Response Fixtures
 * 
 * Comprehensive fixtures for project version management endpoints including
 * project creation, version control, and collaboration features.
 */

import type {
    Project,
    ProjectVersion,
    ProjectVersionCreateInput,
    ProjectVersionUpdateInput,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import { teamResponseFactory } from "./teamResponses.js";
import type { MockDataOptions } from "./types.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_DESCRIPTION_LENGTH = 2000;
const MIN_NAME_LENGTH = 2;
const MAX_COMPLEXITY = 10;
const MIN_COMPLEXITY = 1;

/**
 * ProjectVersion API response factory
 */
export class ProjectVersionResponseFactory extends BaseAPIResponseFactory<
    ProjectVersion,
    ProjectVersionCreateInput,
    ProjectVersionUpdateInput
> {
    protected readonly entityName = "project version";

    /**
     * Create mock project version data
     */
    createMockData(options?: MockDataOptions): ProjectVersion {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const versionId = options?.overrides?.id || generatePK().toString();
        const projectId = generatePK().toString();

        const baseProjectVersion: ProjectVersion = {
            __typename: "ProjectVersion",
            id: versionId,
            createdAt: now,
            updatedAt: now,
            versionLabel: "1.0.0",
            versionNotes: null,
            isComplete: false,
            isDeleted: false,
            isLatest: true,
            isPrivate: false,
            completedAt: null,
            complexity: 5,
            translations: [{
                __typename: "ProjectVersionTranslation",
                id: generatePK().toString(),
                language: "en",
                name: "Test Project",
                description: "A sample project for testing",
            }],
            translationsCount: 1,
            directories: [],
            directoriesCount: 0,
            root: {
                __typename: "Project",
                id: projectId,
                createdAt: now,
                updatedAt: now,
                handle: `project_${projectId.slice(0, 8)}`,
                isDeleted: false,
                isPrivate: false,
                permissions: JSON.stringify({ canRead: true }),
                completedAt: null,
                score: 10,
                bookmarks: 0,
                views: 0,
                owner: userResponseFactory.createMockData(),
                hasCompleteVersion: false,
                tags: [],
                tagsCount: 0,
                transfersCount: 0,
                versionsCount: 1,
                you: {
                    canComment: true,
                    canCopy: true,
                    canDelete: true,
                    canUpdate: true,
                    canBookmark: true,
                    canReport: false,
                    canTransfer: true,
                    canReact: true,
                    isBookmarked: false,
                    isViewed: false,
                    reaction: null,
                },
            } as Project,
            pullRequest: null,
            suggestedNextByProject: [],
            suggestedNextByProjectCount: 0,
            you: {
                canComment: true,
                canCopy: true,
                canDelete: true,
                canUpdate: true,
                canBookmark: true,
                canReport: false,
                canRead: true,
                canReact: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseProjectVersion,
                versionLabel: "2.1.0",
                versionNotes: "Major update with new features and improvements",
                isComplete: false,
                complexity: 8,
                translations: [{
                    __typename: "ProjectVersionTranslation",
                    id: generatePK().toString(),
                    language: "en",
                    name: "Advanced Project System",
                    description: "A comprehensive project management system with advanced features for team collaboration and task automation",
                }],
                directories: [{
                    __typename: "ProjectVersionDirectory",
                    id: generatePK().toString(),
                    createdAt: now,
                    updatedAt: now,
                    isRoot: true,
                    childOrder: [],
                    translations: [{
                        __typename: "ProjectVersionDirectoryTranslation",
                        id: generatePK().toString(),
                        language: "en",
                        name: "Root Directory",
                        description: "Main project directory",
                    }],
                }],
                directoriesCount: 1,
                root: {
                    ...baseProjectVersion.root,
                    score: 85,
                    bookmarks: 23,
                    views: 150,
                    hasCompleteVersion: false,
                    tagsCount: 3,
                    tags: [{
                        __typename: "Tag",
                        id: generatePK().toString(),
                        createdAt: now,
                        updatedAt: now,
                        tag: "project-management",
                    }, {
                        __typename: "Tag",
                        id: generatePK().toString(),
                        createdAt: now,
                        updatedAt: now,
                        tag: "automation",
                    }, {
                        __typename: "Tag",
                        id: generatePK().toString(),
                        createdAt: now,
                        updatedAt: now,
                        tag: "collaboration",
                    }],
                    you: {
                        ...baseProjectVersion.root.you,
                        isBookmarked: true,
                        isViewed: true,
                    },
                },
                you: {
                    ...baseProjectVersion.you,
                    isBookmarked: true,
                    isViewed: true,
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseProjectVersion,
            ...options?.overrides,
        };
    }

    /**
     * Create project version from input
     */
    createFromInput(input: ProjectVersionCreateInput): ProjectVersion {
        const now = new Date().toISOString();
        const versionId = generatePK().toString();

        return {
            __typename: "ProjectVersion",
            id: versionId,
            createdAt: now,
            updatedAt: now,
            versionLabel: input.versionLabel || "1.0.0",
            versionNotes: input.versionNotes || null,
            isComplete: false,
            isDeleted: false,
            isLatest: true,
            isPrivate: input.isPrivate || false,
            completedAt: null,
            complexity: input.complexity || 5,
            translations: input.translationsCreate?.map(t => ({
                __typename: "ProjectVersionTranslation" as const,
                id: generatePK().toString(),
                language: t.language,
                name: t.name || "",
                description: t.description || null,
            })) || [],
            translationsCount: input.translationsCreate?.length || 0,
            directories: input.directoriesCreate?.map((dir, index) => ({
                __typename: "ProjectVersionDirectory" as const,
                id: generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                isRoot: index === 0,
                childOrder: dir.childOrder || [],
                translations: dir.translationsCreate?.map(t => ({
                    __typename: "ProjectVersionDirectoryTranslation" as const,
                    id: generatePK().toString(),
                    language: t.language,
                    name: t.name || "",
                    description: t.description || null,
                })) || [],
            })) || [],
            directoriesCount: input.directoriesCreate?.length || 0,
            root: {
                __typename: "Project",
                id: input.rootConnect || generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                handle: `project_${versionId.slice(0, 8)}`,
                isDeleted: false,
                isPrivate: input.isPrivate || false,
                permissions: JSON.stringify({ canRead: true }),
                completedAt: null,
                score: 0,
                bookmarks: 0,
                views: 0,
                owner: userResponseFactory.createMockData(),
                hasCompleteVersion: false,
                tags: [],
                tagsCount: 0,
                transfersCount: 0,
                versionsCount: 1,
                you: {
                    canComment: true,
                    canCopy: true,
                    canDelete: true,
                    canUpdate: true,
                    canBookmark: true,
                    canReport: false,
                    canTransfer: true,
                    canReact: true,
                    isBookmarked: false,
                    isViewed: false,
                    reaction: null,
                },
            } as Project,
            pullRequest: null,
            suggestedNextByProject: [],
            suggestedNextByProjectCount: 0,
            you: {
                canComment: true,
                canCopy: true,
                canDelete: true,
                canUpdate: true,
                canBookmark: true,
                canReport: false,
                canRead: true,
                canReact: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        };
    }

    /**
     * Update project version from input
     */
    updateFromInput(existing: ProjectVersion, input: ProjectVersionUpdateInput): ProjectVersion {
        const updates: Partial<ProjectVersion> = {
            updatedAt: new Date().toISOString(),
        };

        if (input.versionLabel !== undefined) updates.versionLabel = input.versionLabel;
        if (input.versionNotes !== undefined) updates.versionNotes = input.versionNotes;
        if (input.isComplete !== undefined) {
            updates.isComplete = input.isComplete;
            if (input.isComplete) {
                updates.completedAt = new Date().toISOString();
            }
        }
        if (input.isPrivate !== undefined) updates.isPrivate = input.isPrivate;
        if (input.complexity !== undefined) updates.complexity = input.complexity;

        // Handle translation updates
        if (input.translationsUpdate) {
            updates.translations = existing.translations?.map(t => {
                const update = input.translationsUpdate?.find(u => u.id === t.id);
                return update ? { ...t, ...update } : t;
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
    async validateCreateInput(input: ProjectVersionCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.translationsCreate || input.translationsCreate.length === 0) {
            errors.translations = "At least one translation is required";
        } else {
            const firstTranslation = input.translationsCreate[0];
            if (!firstTranslation.name) {
                errors["translations.0.name"] = "Name is required";
            } else if (firstTranslation.name.length < MIN_NAME_LENGTH) {
                errors["translations.0.name"] = "Name must be at least 2 characters";
            }

            if (firstTranslation.description && firstTranslation.description.length > MAX_DESCRIPTION_LENGTH) {
                errors["translations.0.description"] = "Description must be 2000 characters or less";
            }
        }

        if (input.complexity !== undefined) {
            if (input.complexity < MIN_COMPLEXITY || input.complexity > MAX_COMPLEXITY) {
                errors.complexity = "Complexity must be between 1 and 10";
            }
        }

        if (!input.rootConnect) {
            errors.root = "Root project is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ProjectVersionUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.versionLabel !== undefined && input.versionLabel.length < 1) {
            errors.versionLabel = "Version label cannot be empty";
        }

        if (input.complexity !== undefined) {
            if (input.complexity < MIN_COMPLEXITY || input.complexity > MAX_COMPLEXITY) {
                errors.complexity = "Complexity must be between 1 and 10";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create team-owned project version
     */
    createTeamProjectVersion(): ProjectVersion {
        const projectVersion = this.createMockData({ scenario: "complete" });
        const team = teamResponseFactory.createMockData({ scenario: "complete" });

        return {
            ...projectVersion,
            root: {
                ...projectVersion.root,
                owner: team,
                you: {
                    ...projectVersion.root.you,
                    canDelete: false, // Team members can't delete team projects
                    canTransfer: false,
                },
            },
        };
    }

    /**
     * Create completed project version
     */
    createCompletedProjectVersion(): ProjectVersion {
        const now = new Date().toISOString();
        return this.createMockData({
            scenario: "complete",
            overrides: {
                isComplete: true,
                completedAt: now,
                root: {
                    ...this.createMockData().root,
                    completedAt: now,
                    hasCompleteVersion: true,
                },
            },
        });
    }

    /**
     * Create project version with multiple directories
     */
    createProjectVersionWithDirectories(dirCount = 3): ProjectVersion {
        const projectVersion = this.createMockData({ scenario: "complete" });
        const now = new Date().toISOString();
        const directories = [];

        for (let i = 0; i < dirCount; i++) {
            directories.push({
                __typename: "ProjectVersionDirectory" as const,
                id: generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                isRoot: i === 0,
                childOrder: [],
                translations: [{
                    __typename: "ProjectVersionDirectoryTranslation" as const,
                    id: generatePK().toString(),
                    language: "en",
                    name: `Directory ${i + 1}`,
                    description: `Project directory ${i + 1} for organizing content`,
                }],
            });
        }

        return {
            ...projectVersion,
            directories,
            directoriesCount: dirCount,
        };
    }
}

/**
 * Pre-configured project version response scenarios
 */
export const projectVersionResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ProjectVersionCreateInput>) => {
        const factory = new ProjectVersionResponseFactory();
        const defaultInput: ProjectVersionCreateInput = {
            versionLabel: "1.0.0",
            isPrivate: false,
            complexity: 5,
            rootConnect: generatePK().toString(),
            translationsCreate: [{
                language: "en",
                name: "New Project",
                description: "A new project for testing",
            }],
            directoriesCreate: [{
                isRoot: true,
                childOrder: [],
                translationsCreate: [{
                    language: "en",
                    name: "Root",
                    description: "Root directory",
                }],
            }],
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (projectVersion?: ProjectVersion) => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createSuccessResponse(
            projectVersion || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: ProjectVersion, updates?: Partial<ProjectVersionUpdateInput>) => {
        const factory = new ProjectVersionResponseFactory();
        const projectVersion = existing || factory.createMockData({ scenario: "complete" });
        const input: ProjectVersionUpdateInput = {
            id: projectVersion.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(projectVersion, input),
        );
    },

    teamProjectSuccess: () => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createSuccessResponse(
            factory.createTeamProjectVersion(),
        );
    },

    completedProjectSuccess: () => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createSuccessResponse(
            factory.createCompletedProjectVersion(),
        );
    },

    withDirectoriesSuccess: (dirCount?: number) => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createSuccessResponse(
            factory.createProjectVersionWithDirectories(dirCount),
        );
    },

    listSuccess: (projectVersions?: ProjectVersion[]) => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createPaginatedResponse(
            projectVersions || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: projectVersions?.length || DEFAULT_COUNT },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createValidationErrorResponse({
            "translations.0.name": "Name is required",
            root: "Root project is required",
            complexity: "Complexity must be between 1 and 10",
        });
    },

    notFoundError: (projectVersionId?: string) => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createNotFoundErrorResponse(
            projectVersionId || "non-existent-project-version",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["project:update"],
        );
    },

    projectInUseError: () => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createBusinessErrorResponse("state", {
            reason: "Project is currently in use",
            message: "Cannot delete project that has active runs or dependencies",
            activeRuns: 2,
        });
    },

    complexityLimitError: () => {
        const factory = new ProjectVersionResponseFactory();
        return factory.createBusinessErrorResponse("limit", {
            resource: "complexity",
            limit: MAX_COMPLEXITY,
            current: 11,
            message: "Project complexity exceeds maximum allowed value",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ProjectVersionResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ProjectVersionResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ProjectVersionResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const projectVersionResponseFactory = new ProjectVersionResponseFactory();
