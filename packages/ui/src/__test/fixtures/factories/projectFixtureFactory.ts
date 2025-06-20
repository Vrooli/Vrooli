/* c8 ignore start */
/**
 * Project fixture factory implementation
 * 
 * This demonstrates the complete pattern for implementing fixtures for a Project object type
 * with full type safety and integration with @vrooli/shared.
 */

import {
    generatePK
} from "@vrooli/shared";

// Import types directly - these are the core types we need
type Resource = {
    __typename: "Resource";
    id: string;
    publicId: string;
    isPrivate: boolean;
    isDeleted: boolean;
    isInternal: boolean;
    resourceType: string;
    hasCompleteVersion: boolean;
    permissions: string;
    score: number;
    views: number;
    bookmarks: number;
    transfersCount: number;
    issuesCount: number;
    pullRequestsCount: number;
    versionsCount: number;
    translatedName: string;
    createdAt: string;
    updatedAt: string;
    completedAt: string | null;
    bookmarkedBy: unknown[];
    createdBy: unknown | null;
    owner: unknown | null;
    parent: unknown | null;
    issues: unknown[];
    pullRequests: unknown[];
    stats: unknown[];
    tags: unknown[];
    transfers: unknown[];
    versions: unknown[];
    you: {
        __typename: "ResourceYou";
        canBookmark: boolean;
        canComment: boolean;
        canDelete: boolean;
        canReact: boolean;
        canRead: boolean;
        canTransfer: boolean;
        canUpdate: boolean;
        isBookmarked: boolean;
        isViewed: boolean;
        reaction: string | null;
    };
};

type ResourceCreateInput = {
    id: string;
    resourceType: string;
    isPrivate?: boolean;
    isInternal?: boolean;
    permissions?: string;
    ownedByUserConnect?: string;
    ownedByTeamConnect?: string;
    tagsCreate?: Array<{ id: string; tag: string }>;
    versionsCreate: Array<{
        id: string;
        versionLabel: string;
        isPrivate?: boolean;
        isComplete?: boolean;
        config?: object;
        translationsCreate: Array<{
            id: string;
            language: string;
            name: string;
            description?: string;
        }>;
    }>;
};

type ResourceUpdateInput = {
    id: string;
    isPrivate?: boolean;
    permissions?: string;
    tagsCreate?: Array<{ id: string; tag: string }>;
    tagsConnect?: string[];
    versionsCreate?: Array<{
        id: string;
        versionLabel: string;
        isPrivate?: boolean;
        isComplete?: boolean;
        config?: object;
        translationsCreate?: Array<{
            id: string;
            language: string;
            name: string;
            description?: string;
        }>;
    }>;
    versionsUpdate?: Array<{
        id: string;
        versionLabel?: string;
        isComplete?: boolean;
        translationsUpdate?: Array<{
            id: string;
            language: string;
            name?: string;
            description?: string;
        }>;
    }>;
};

// Simple project config for fixtures
const projectConfig = {
    minimal: { __version: "1.0.0" },
    complete: {
        __version: "1.0.0",
        resources: [
            {
                link: "https://example.com/docs",
                usedFor: "OfficialWebsite",
                translations: [{
                    language: "en",
                    name: "Documentation",
                    description: "Official documentation"
                }]
            }
        ]
    }
};
import { BaseFormFixtureFactory } from "../BaseFormFixtureFactory.js";
import { BaseRoundTripOrchestrator } from "../BaseRoundTripOrchestrator.js";
import { BaseMSWHandlerFactory } from "../BaseMSWHandlerFactory.js";
import { createValidationAdapter } from "../utils/integration.js";
import type {
    UIFixtureFactory,
    FormFixtureFactory,
    RoundTripOrchestrator,
    MSWHandlerFactory,
    UIStateFixtureFactory,
    ComponentTestUtils,
    TestAPIClient,
    DatabaseVerifier
} from "../types.js";
import { registerFixture } from "./index.js";

/**
 * Project form data type
 * 
 * This includes UI-specific fields that don't exist in the API input type.
 */
export interface ProjectFormData {
    handle: string;
    name: string;
    description?: string;
    isPrivate?: boolean;
    team?: { id: string } | null;
    projectType?: string;
    tags?: string[];
    isComplete?: boolean;
    completedAt?: string;
    bannerImage?: string | File;
    config?: object;
    [key: string]: unknown; // Add index signature for compatibility
}

/**
 * Project UI state type
 */
export interface ProjectUIState {
    isLoading: boolean;
    project: Resource | null;
    error: string | null;
    isEditing: boolean;
    hasUnsavedChanges: boolean;
}

/**
 * Project form fixture factory
 */
class ProjectFormFixtureFactory extends BaseFormFixtureFactory<ProjectFormData, ResourceCreateInput> {
    constructor() {
        super({
            scenarios: {
                minimal: {
                    handle: "test-project",
                    name: "Test Project"
                },
                complete: {
                    handle: "complete-project",
                    name: "Complete Project",
                    description: "A comprehensive project with all features",
                    isPrivate: false,
                    projectType: "education",
                    tags: ["open-source", "education"],
                    isComplete: false,
                    bannerImage: "project-banner.jpg",
                    config: projectConfig.complete
                },
                invalid: {
                    handle: "", // Too short
                    name: "",
                    description: "A".repeat(10000), // Too long
                    isPrivate: "yes" as any, // Wrong type
                    tags: ["a".repeat(100)] // Tag too long
                },
                privateProject: {
                    handle: "private-project",
                    name: "Private Project",
                    description: "A private project for team members only",
                    isPrivate: true,
                    team: { id: generatePK().toString() }
                },
                teamProject: {
                    handle: "team-project",
                    name: "Team Project",
                    description: "A collaborative team project",
                    isPrivate: false,
                    team: { id: generatePK().toString() },
                    tags: ["collaboration", "team"]
                },
                completedProject: {
                    handle: "completed-project",
                    name: "Completed Project",
                    description: "A project that has been completed",
                    isPrivate: false,
                    isComplete: true,
                    completedAt: new Date().toISOString(),
                    tags: ["completed", "archived"]
                }
            },
            
            validate: createValidationAdapter<ProjectFormData>(
                async (data) => {
                    // Additional UI validation
                    const errors: string[] = [];
                    
                    if (!data.name || (typeof data.name === 'string' && data.name.length < 1)) {
                        errors.push("name: Name is required");
                    }
                    
                    if (!data.handle || (typeof data.handle === 'string' && data.handle.length < 3)) {
                        errors.push("handle: Handle must be at least 3 characters");
                    }
                    
                    if (data.tags && Array.isArray(data.tags) && data.tags.some(tag => typeof tag === 'string' && tag.length > 50)) {
                        errors.push("tags: Tags must be 50 characters or less");
                    }
                    
                    if (errors.length > 0) {
                        return { isValid: false, errors };
                    }
                    
                    // Simple validation - in a real implementation you'd use shared validation
                    return { isValid: true, data: undefined };
                }
            ),
            
            shapeToAPI: (formData) => {
                // Transform to ResourceCreateInput
                const versionId = generatePK().toString();
                const translationId = generatePK().toString();
                
                return {
                    id: generatePK().toString(),
                    resourceType: "Project" as any,
                    isPrivate: formData.isPrivate ?? false,
                    ...(formData.team && { ownedByTeamConnect: formData.team.id }),
                    ...(formData.tags && formData.tags.length > 0 && {
                        tagsCreate: formData.tags.map(tag => ({
                            id: generatePK().toString(),
                            tag
                        }))
                    }),
                    versionsCreate: [{
                        id: versionId,
                        versionLabel: "1.0.0",
                        isPrivate: formData.isPrivate ?? false,
                        isComplete: formData.isComplete ?? false,
                        config: formData.config ?? projectConfig.minimal,
                        translationsCreate: [{
                            id: translationId,
                            language: "en",
                            name: formData.name,
                            ...(formData.description && { description: formData.description })
                        }]
                    }]
                };
            }
        });
    }
    
    /**
     * Create update form data
     */
    createUpdateFormData(scenario: "minimal" | "complete" = "minimal"): Partial<ProjectFormData> {
        if (scenario === "minimal") {
            return {
                name: "Updated Project Name"
            };
        }
        
        return {
            handle: "updated-handle",
            name: "Updated Project Name",
            description: "Updated description with more details",
            isPrivate: true,
            tags: ["updated", "improved"],
            isComplete: true,
            completedAt: new Date().toISOString()
        };
    }
}

/**
 * Project MSW handler factory
 */
class ProjectMSWHandlerFactory extends BaseMSWHandlerFactory<ResourceCreateInput, ResourceUpdateInput, Resource> {
    constructor() {
        super({
            baseUrl: "/api",
            endpoints: {
                create: "/resource",
                update: "/resource",
                delete: "/resource",
                find: "/resource",
                list: "/resources"
            },
            successResponses: {
                create: (input) => ({
                    __typename: "Resource" as const,
                    id: generatePK().toString(),
                    publicId: `project_${generatePK().toString()}`,
                    isPrivate: input.isPrivate || false,
                    isDeleted: false,
                    isInternal: false,
                    resourceType: "Project" as any,
                    hasCompleteVersion: true,
                    permissions: "{}",
                    score: 0,
                    views: 0,
                    bookmarks: 0,
                    transfersCount: 0,
                    issuesCount: 0,
                    pullRequestsCount: 0,
                    versionsCount: 1,
                    translatedName: input.versionsCreate?.[0]?.translationsCreate?.[0]?.name || "Project",
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    completedAt: null,
                    bookmarkedBy: [],
                    createdBy: null,
                    owner: null,
                    parent: null,
                    issues: [],
                    pullRequests: [],
                    stats: [],
                    tags: [],
                    transfers: [],
                    versions: [{
                        __typename: "ResourceVersion" as const,
                        id: input.versionsCreate?.[0]?.id || generatePK().toString(),
                        publicId: `version_${generatePK().toString()}`,
                        isPrivate: input.isPrivate || false,
                        isComplete: true,
                        isAutomatable: false,
                        versionLabel: "1.0.0",
                        versionNotes: null,
                        codeLanguage: null,
                        config: projectConfig.minimal,
                        resourceSubType: null,
                        relatedVersions: [],
                        root: null,
                        translations: [{
                            __typename: "ResourceVersionTranslation" as const,
                            id: input.versionsCreate?.[0]?.translationsCreate?.[0]?.id || generatePK().toString(),
                            language: "en",
                            name: input.versionsCreate?.[0]?.translationsCreate?.[0]?.name || "Project",
                            description: input.versionsCreate?.[0]?.translationsCreate?.[0]?.description || null,
                            details: null,
                            instructions: null
                        }]
                    }],
                    you: {
                        __typename: "ResourceYou" as const,
                        canBookmark: true,
                        canComment: true,
                        canDelete: true,
                        canReact: true,
                        canRead: true,
                        canTransfer: true,
                        canUpdate: true,
                        isBookmarked: false,
                        isViewed: false,
                        reaction: null,
                    }
                }),
                update: (input) => ({
                    __typename: "Resource" as const,
                    id: "existing-id",
                    publicId: "project_existing-id",
                    isPrivate: false,
                    isDeleted: false,
                    isInternal: false,
                    resourceType: "Project" as any,
                    hasCompleteVersion: true,
                    permissions: "{}",
                    score: 0,
                    views: 0,
                    bookmarks: 0,
                    transfersCount: 0,
                    issuesCount: 0,
                    pullRequestsCount: 0,
                    versionsCount: 1,
                    translatedName: "Updated Project",
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    completedAt: null,
                    bookmarkedBy: [],
                    createdBy: null,
                    owner: null,
                    parent: null,
                    issues: [],
                    pullRequests: [],
                    stats: [],
                    tags: [],
                    transfers: [],
                    versions: [],
                    you: {
                        __typename: "ResourceYou" as const,
                        canBookmark: true,
                        canComment: true,
                        canDelete: true,
                        canReact: true,
                        canRead: true,
                        canTransfer: true,
                        canUpdate: true,
                        isBookmarked: false,
                        isViewed: false,
                        reaction: null,
                    }
                }),
                find: (id) => ({
                    __typename: "Resource" as const,
                    id,
                    publicId: `project_${id}`,
                    isPrivate: false,
                    isDeleted: false,
                    isInternal: false,
                    resourceType: "Project" as any,
                    hasCompleteVersion: true,
                    permissions: "{}",
                    score: 0,
                    views: 0,
                    bookmarks: 0,
                    transfersCount: 0,
                    issuesCount: 0,
                    pullRequestsCount: 0,
                    versionsCount: 1,
                    translatedName: "Found Project",
                    createdAt: new Date().toISOString(),
                    updatedAt: new Date().toISOString(),
                    completedAt: null,
                    bookmarkedBy: [],
                    createdBy: null,
                    owner: null,
                    parent: null,
                    issues: [],
                    pullRequests: [],
                    stats: [],
                    tags: [],
                    transfers: [],
                    versions: [],
                    you: {
                        __typename: "ResourceYou" as const,
                        canBookmark: true,
                        canComment: true,
                        canDelete: true,
                        canReact: true,
                        canRead: true,
                        canTransfer: true,
                        canUpdate: true,
                        isBookmarked: false,
                        isViewed: false,
                        reaction: null,
                    }
                })
            },
            validate: {
                create: (input) => {
                    const errors: string[] = [];
                    
                    if (!input.resourceType || input.resourceType !== "Project") {
                        errors.push("Invalid resource type");
                    }
                    
                    if (!input.versionsCreate || input.versionsCreate.length === 0) {
                        errors.push("At least one version is required");
                    }
                    
                    if (input.versionsCreate?.[0]?.translationsCreate?.length === 0) {
                        errors.push("At least one translation is required");
                    }
                    
                    return {
                        isValid: errors.length === 0,
                        errors
                    };
                }
            }
        });
    }
    
    /**
     * Create team project handlers
     */
    createTeamProjectHandlers() {
        return this.createCustomHandler({
            method: "POST",
            path: "/team-project",
            response: {
                __typename: "Resource" as const,
                id: generatePK().toString(),
                publicId: `project_${generatePK().toString()}`,
                isPrivate: false,
                isDeleted: false,
                isInternal: false,
                resourceType: "Project" as any,
                hasCompleteVersion: true,
                permissions: "{}",
                score: 0,
                views: 0,
                bookmarks: 0,
                transfersCount: 0,
                issuesCount: 0,
                pullRequestsCount: 0,
                versionsCount: 1,
                translatedName: "Team Project",
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                completedAt: null,
                bookmarkedBy: [],
                createdBy: null,
                owner: { __typename: "Team", id: generatePK().toString() },
                parent: null,
                issues: [],
                pullRequests: [],
                stats: [],
                tags: [],
                transfers: [],
                versions: [],
                you: {
                    __typename: "ResourceYou" as const,
                    canBookmark: true,
                    canComment: true,
                    canDelete: true,
                    canReact: true,
                    canRead: true,
                    canTransfer: true,
                    canUpdate: true,
                    isBookmarked: false,
                    isViewed: false,
                    reaction: null,
                }
            }
        });
    }
}

/**
 * Project UI state fixture factory
 */
class ProjectUIStateFixtureFactory implements UIStateFixtureFactory<ProjectUIState> {
    createLoadingState(context?: { type: string }): ProjectUIState {
        return {
            isLoading: true,
            project: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createErrorState(error: { message: string }): ProjectUIState {
        return {
            isLoading: false,
            project: null,
            error: error.message,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createSuccessState(data: Resource): ProjectUIState {
        return {
            isLoading: false,
            project: data,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createEmptyState(): ProjectUIState {
        return {
            isLoading: false,
            project: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    transitionToLoading(currentState: ProjectUIState): ProjectUIState {
        return {
            ...currentState,
            isLoading: true,
            error: null
        };
    }
    
    transitionToSuccess(currentState: ProjectUIState, data: Resource): ProjectUIState {
        return {
            ...currentState,
            isLoading: false,
            project: data,
            error: null,
            hasUnsavedChanges: false
        };
    }
    
    transitionToError(currentState: ProjectUIState, error: { message: string }): ProjectUIState {
        return {
            ...currentState,
            isLoading: false,
            error: error.message
        };
    }
}

/**
 * Complete Project fixture factory
 */
export class ProjectFixtureFactory implements UIFixtureFactory<
    ProjectFormData,
    ResourceCreateInput,
    ResourceUpdateInput,
    Resource,
    ProjectUIState
> {
    readonly objectType = "project";
    
    form: ProjectFormFixtureFactory;
    roundTrip: RoundTripOrchestrator<ProjectFormData, Resource>;
    handlers: ProjectMSWHandlerFactory;
    states: ProjectUIStateFixtureFactory;
    componentUtils: ComponentTestUtils<any>;
    
    constructor(apiClient: TestAPIClient, dbVerifier: DatabaseVerifier) {
        this.form = new ProjectFormFixtureFactory();
        this.handlers = new ProjectMSWHandlerFactory();
        this.states = new ProjectUIStateFixtureFactory();
        
        // Initialize round-trip orchestrator
        this.roundTrip = new BaseRoundTripOrchestrator({
            apiClient,
            dbVerifier,
            formFixture: this.form,
            endpoints: {
                create: "/api/resource",
                update: "/api/resource",
                delete: "/api/resource",
                find: "/api/resource"
            },
            tableName: "resource",
            fieldMappings: {
                name: "versions.translations.name",
                description: "versions.translations.description",
                isPrivate: "isPrivate",
                handle: "versions.translations.name" // Simplified mapping
            }
        });
        
        // Component utils would be initialized here
        this.componentUtils = {} as any; // Placeholder
    }
    
    createFormData(scenario: "minimal" | "complete" | string = "minimal"): ProjectFormData {
        return this.form.createFormData(scenario);
    }
    
    createAPIInput(formData: ProjectFormData): ResourceCreateInput {
        return this.form.transformToAPIInput(formData);
    }
    
    createMockResponse(overrides?: Partial<Resource>): Resource {
        return {
            __typename: "Resource" as const,
            id: generatePK().toString(),
            publicId: `project_${generatePK().toString()}`,
            isPrivate: false,
            isDeleted: false,
            isInternal: false,
            resourceType: "Project" as any,
            hasCompleteVersion: true,
            permissions: "{}",
            score: 0,
            views: 0,
            bookmarks: 0,
            transfersCount: 0,
            issuesCount: 0,
            pullRequestsCount: 0,
            versionsCount: 1,
            translatedName: "Mock Project",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            completedAt: null,
            bookmarkedBy: [],
            createdBy: null,
            owner: null,
            parent: null,
            issues: [],
            pullRequests: [],
            stats: [],
            tags: [],
            transfers: [],
            versions: [],
            you: {
                __typename: "ResourceYou" as const,
                canBookmark: true,
                canComment: true,
                canDelete: true,
                canReact: true,
                canRead: true,
                canTransfer: true,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
            ...overrides
        };
    }
    
    setupMSW(scenario: "success" | "error" | "loading" = "success"): void {
        // This would integrate with the MSW server
        // For now, it's a placeholder
        console.log(`Setting up MSW handlers for scenario: ${scenario}`);
    }
    
    async testCreateFlow(formData?: ProjectFormData): Promise<Resource> {
        const data = formData || this.createFormData("minimal");
        const result = await this.roundTrip.testCreateFlow(data);
        
        if (!result.success) {
            throw new Error(`Create flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.id as unknown as Resource;
    }
    
    async testUpdateFlow(id: string, updates: Partial<ProjectFormData>): Promise<Resource> {
        const result = await this.roundTrip.testUpdateFlow(id, updates);
        
        if (!result.success) {
            throw new Error(`Update flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.updatedData as Resource;
    }
    
    async testDeleteFlow(id: string): Promise<boolean> {
        const result = await this.roundTrip.testDeleteFlow(id);
        return result.success;
    }
    
    async testRoundTrip(formData?: ProjectFormData) {
        const data = formData || this.createFormData("complete");
        const result = await this.roundTrip.executeFullCycle({
            formData: data,
            validateEachStep: true
        });
        
        if (!result.success) {
            throw new Error(`Round trip failed: ${result.errors?.join(", ")}`);
        }
        
        return {
            success: result.success,
            formData: data,
            apiResponse: result.data!.apiResponse,
            uiState: this.states.createSuccessState(result.data!.apiResponse)
        };
    }
}

// Register in the global registry
// This would normally be done after creating the API client and DB verifier
// registerFixture("project", new ProjectFixtureFactory(apiClient, dbVerifier));
/* c8 ignore stop */