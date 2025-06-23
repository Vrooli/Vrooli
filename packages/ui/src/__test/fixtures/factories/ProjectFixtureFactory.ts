/**
 * Production-grade project fixture factory
 * 
 * This factory provides type-safe project fixtures using real functions from @vrooli/shared.
 * Projects in Vrooli are Resources with resourceType="Project" and ResourceVersions.
 */

import type { 
    Resource, 
    ResourceCreateInput, 
    ResourceUpdateInput,
    ResourceVersion,
    ResourceVersionCreateInput,
    ResourceVersionUpdateInput,
    ResourceVersionTranslation,
    Tag,
    Team,
    User,
} from "@vrooli/shared";
import { 
    resourceValidation,
    shapeResource,
    ResourceType,
    MemberRole,
} from "@vrooli/shared";
import type { 
    FixtureFactory, 
    ValidationResult, 
    MSWHandlers,
} from "../types.js";
import { rest } from "msw";

/**
 * UI-specific form data for project creation
 */
export interface ProjectFormData {
    handle: string;
    name: string;
    description?: string;
    details?: string;
    instructions?: string;
    versionLabel?: string;
    isPrivate?: boolean;
    tags?: string[];
    teamId?: string;
    completedAt?: Date | null;
    hasSupporting?: boolean;
    supportingUrls?: string[];
}

/**
 * UI state for project components
 */
export interface ProjectUIState {
    isLoading: boolean;
    project: Resource | null;
    currentVersion: ResourceVersion | null;
    error: string | null;
    isOwner: boolean;
    canEdit: boolean;
    canDelete: boolean;
    completionPercentage: number;
}

export type ProjectScenario = 
    | "minimal" 
    | "complete" 
    | "invalid" 
    | "privateProject"
    | "teamProject"
    | "completedProject"
    | "withTags"
    | "multiVersion";

/**
 * Type-safe project fixture factory that uses real @vrooli/shared functions
 */
export class ProjectFixtureFactory implements FixtureFactory<
    ProjectFormData,
    ResourceCreateInput,
    ResourceUpdateInput,
    Resource
> {
    readonly objectType = "project";

    /**
     * Generate a unique ID for testing
     */
    private generateId(): string {
        return `test_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Generate a unique handle
     */
    private generateHandle(): string {
        return `testproject_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Generate default project configuration
     */
    private generateProjectConfig() {
        return {
            version: 1,
            formConfig: {
                version: 1,
                sections: [],
            },
            resourceConfig: {
                version: 1,
                listOrTree: "list" as const,
                resources: [],
            },
        };
    }

    /**
     * Create form data for different test scenarios
     */
    createFormData(scenario: ProjectScenario = "minimal"): ProjectFormData {
        switch (scenario) {
            case "minimal":
                return {
                    handle: this.generateHandle(),
                    name: "Test Project",
                    versionLabel: "1.0.0",
                    isPrivate: false,
                };

            case "complete":
                return {
                    handle: this.generateHandle(),
                    name: "Complete Test Project",
                    description: "A comprehensive test project with all fields",
                    details: "This project includes detailed documentation and instructions for testing purposes.",
                    instructions: "1. Start here\n2. Follow the steps\n3. Complete the project",
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    tags: ["testing", "development", "documentation"],
                    hasSupporting: true,
                    supportingUrls: ["https://example.com/docs", "https://example.com/tutorial"],
                };

            case "invalid":
                return {
                    handle: "", // Empty handle
                    name: "", // Empty name
                    // @ts-expect-error - Testing invalid version label
                    versionLabel: null,
                    isPrivate: false,
                };

            case "privateProject":
                return {
                    handle: this.generateHandle(),
                    name: "Private Project",
                    description: "This project is private and not publicly accessible",
                    versionLabel: "0.1.0",
                    isPrivate: true,
                };

            case "teamProject":
                return {
                    handle: this.generateHandle(),
                    name: "Team Project",
                    description: "A project owned by a team",
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    teamId: this.generateId(),
                    tags: ["team", "collaboration"],
                };

            case "completedProject":
                return {
                    handle: this.generateHandle(),
                    name: "Completed Project",
                    description: "This project has been completed",
                    versionLabel: "2.0.0",
                    isPrivate: false,
                    completedAt: new Date(),
                    tags: ["completed", "archived"],
                };

            case "withTags":
                return {
                    handle: this.generateHandle(),
                    name: "Tagged Project",
                    description: "A project with multiple tags",
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    tags: ["ai", "machine-learning", "nlp", "research", "open-source"],
                };

            case "multiVersion":
                return {
                    handle: this.generateHandle(),
                    name: "Multi-Version Project",
                    description: "A project with multiple versions",
                    versionLabel: "3.2.1",
                    isPrivate: false,
                    details: "This version includes breaking changes from v2",
                };

            default:
                throw new Error(`Unknown project scenario: ${scenario}`);
        }
    }

    /**
     * Transform form data to API create input using real shape function
     */
    transformToAPIInput(formData: ProjectFormData): ResourceCreateInput {
        // Create translation for the version
        const translation: ResourceVersionTranslation = {
            __typename: "ResourceVersionTranslation",
            id: this.generateId(),
            language: "en",
            name: formData.name,
            description: formData.description || null,
            details: formData.details || null,
            instructions: formData.instructions || null,
        };

        // Create resource version
        const version: ResourceVersion = {
            __typename: "ResourceVersion",
            id: this.generateId(),
            versionLabel: formData.versionLabel || "1.0.0",
            isPrivate: formData.isPrivate || false,
            config: this.generateProjectConfig(),
            translations: [translation],
            translationsCount: 1,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            completedAt: formData.completedAt?.toISOString() || null,
            isComplete: !!formData.completedAt,
            resources: formData.supportingUrls?.map(url => ({
                __typename: "Resource" as const,
                id: this.generateId(),
                link: url,
                isInternal: false,
                isPrivate: false,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                usedBy: [],
                usedByCount: 0,
                versions: [],
                versionsCount: 0,
                you: {
                    __typename: "ResourceYou" as const,
                    canDelete: false,
                    canUpdate: false,
                    canReport: false,
                    isBookmarked: false,
                    isReacted: false,
                    reaction: null,
                },
            })) || [],
            resourcesCount: formData.supportingUrls?.length || 0,
            you: {
                __typename: "ResourceVersionYou",
                canComment: true,
                canCopy: true,
                canDelete: true,
                canReport: false,
                canUpdate: true,
                canUse: true,
                canBookmark: true,
                canReact: true,
                isBookmarked: false,
                isReacted: false,
                reaction: null,
            },
        };

        // Create the resource shape that matches the expected API structure
        const resourceShape = {
            __typename: "Resource" as const,
            id: this.generateId(),
            handle: formData.handle,
            isPrivate: formData.isPrivate || false,
            resourceType: ResourceType.Project,
            ownedByTeam: formData.teamId ? { id: formData.teamId } : null,
            ownedByUser: formData.teamId ? null : { id: this.generateId() },
            versions: [version],
            tags: formData.tags?.map(tag => ({
                __typename: "Tag" as const,
                id: this.generateId(),
                tag,
            })) || null,
        };

        // Use real shape function from @vrooli/shared
        return shapeResource.create(resourceShape);
    }

    /**
     * Create API update input
     */
    createUpdateInput(id: string, updates: Partial<ProjectFormData>): ResourceUpdateInput {
        const updateInput: ResourceUpdateInput = { id };

        if (updates.handle) updateInput.handle = updates.handle;
        if (updates.isPrivate !== undefined) updateInput.isPrivate = updates.isPrivate;

        // Handle version updates
        if (updates.name || updates.description || updates.versionLabel) {
            updateInput.versionsUpdate = [{
                id: this.generateId(),
                versionLabel: updates.versionLabel,
                translationsUpdate: [{
                    id: this.generateId(),
                    language: "en",
                    name: updates.name,
                    description: updates.description,
                    details: updates.details,
                    instructions: updates.instructions,
                }],
            }];
        }

        // Handle tags
        if (updates.tags) {
            updateInput.tagsConnect = updates.tags.map(tag => tag);
        }

        return updateInput;
    }

    /**
     * Create mock project response with realistic data
     */
    createMockResponse(overrides?: Partial<Resource>): Resource {
        const now = new Date().toISOString();
        const projectId = this.generateId();
        const versionId = this.generateId();
        const userId = this.generateId();
        
        const defaultProject: Resource = {
            __typename: "Resource",
            id: projectId,
            handle: this.generateHandle(),
            publicId: `project_${projectId}`,
            createdAt: now,
            updatedAt: now,
            isPrivate: false,
            isInternal: false,
            isDeleted: false,
            owner: {
                __typename: "User",
                id: userId,
                handle: "projectowner",
                name: "Project Owner",
                email: "owner@example.com",
                emailVerified: true,
                createdAt: now,
                updatedAt: now,
                isBot: false,
                isPrivate: false,
                profileImage: null,
                bannerImage: null,
                premium: false,
                premiumExpiration: null,
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "UserYou",
                    isBlocked: false,
                    isBlockedByYou: false,
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isReacted: false,
                    reactionSummary: {
                        __typename: "ReactionSummary",
                        emotion: null,
                        count: 0,
                    },
                },
            },
            permissions: JSON.stringify({
                canUpdate: true,
                canDelete: true,
                canTransfer: true,
            }),
            resourceType: ResourceType.Project,
            tags: [],
            tagsCount: 0,
            usedBy: [],
            usedByCount: 0,
            versions: [{
                __typename: "ResourceVersion",
                id: versionId,
                versionLabel: "1.0.0",
                isPrivate: false,
                isComplete: false,
                completedAt: null,
                createdAt: now,
                updatedAt: now,
                config: this.generateProjectConfig(),
                translations: [{
                    __typename: "ResourceVersionTranslation",
                    id: this.generateId(),
                    language: "en",
                    name: "Test Project",
                    description: "A test project for development",
                    details: null,
                    instructions: null,
                }],
                translationsCount: 1,
                resources: [],
                resourcesCount: 0,
                you: {
                    __typename: "ResourceVersionYou",
                    canComment: true,
                    canCopy: true,
                    canDelete: true,
                    canReport: false,
                    canUpdate: true,
                    canUse: true,
                    canBookmark: true,
                    canReact: true,
                    isBookmarked: false,
                    isReacted: false,
                    reaction: null,
                },
            }],
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

        return {
            ...defaultProject,
            ...overrides,
        };
    }

    /**
     * Validate form data using real validation from @vrooli/shared
     */
    async validateFormData(formData: ProjectFormData): Promise<ValidationResult> {
        try {
            // Transform to API input format for validation
            const apiInput = this.transformToAPIInput(formData);
            
            // Use real validation schema from @vrooli/shared
            await resourceValidation.create(apiInput).validate(apiInput, { abortEarly: false });
            
            return { isValid: true };
        } catch (error: any) {
            return {
                isValid: false,
                errors: error.errors || [error.message],
                fieldErrors: error.inner?.reduce((acc: any, err: any) => {
                    if (err.path) {
                        acc[err.path] = acc[err.path] || [];
                        acc[err.path].push(err.message);
                    }
                    return acc;
                }, {}),
            };
        }
    }

    /**
     * Create MSW handlers for different scenarios
     */
    createMSWHandlers(): MSWHandlers {
        const baseUrl = process.env.VITE_SERVER_URL || "http://localhost:3000";

        return {
            success: [
                // Create project
                rest.post(`${baseUrl}/api/project`, async (req, res, ctx) => {
                    const body = await req.json();
                    
                    // Validate the request body
                    const validation = await this.validateFormData(body);
                    if (!validation.isValid) {
                        return res(
                            ctx.status(400),
                            ctx.json({ 
                                errors: validation.errors,
                                fieldErrors: validation.fieldErrors, 
                            }),
                        );
                    }

                    // Return successful response
                    const mockProject = this.createMockResponse({
                        handle: body.handle,
                        versions: [{
                            ...this.createMockResponse().versions[0],
                            translations: [{
                                ...this.createMockResponse().versions[0].translations[0],
                                name: body.name,
                                description: body.description,
                            }],
                        }],
                    });

                    return res(
                        ctx.status(201),
                        ctx.json(mockProject),
                    );
                }),

                // Update project
                rest.put(`${baseUrl}/api/project/:id`, async (req, res, ctx) => {
                    const { id } = req.params;
                    const body = await req.json();

                    const mockProject = this.createMockResponse({ 
                        id: id as string,
                        updatedAt: new Date().toISOString(),
                    });

                    return res(
                        ctx.status(200),
                        ctx.json(mockProject),
                    );
                }),

                // Get project
                rest.get(`${baseUrl}/api/project/:handle`, (req, res, ctx) => {
                    const { handle } = req.params;
                    const mockProject = this.createMockResponse({ 
                        handle: handle as string, 
                    });
                    
                    return res(
                        ctx.status(200),
                        ctx.json(mockProject),
                    );
                }),

                // Complete project
                rest.post(`${baseUrl}/api/project/:id/complete`, (req, res, ctx) => {
                    const { id } = req.params;
                    
                    const mockProject = this.createMockResponse({ 
                        id: id as string,
                        versions: [{
                            ...this.createMockResponse().versions[0],
                            isComplete: true,
                            completedAt: new Date().toISOString(),
                        }],
                    });

                    return res(
                        ctx.status(200),
                        ctx.json(mockProject),
                    );
                }),

                // Delete project
                rest.delete(`${baseUrl}/api/project/:id`, (req, res, ctx) => {
                    return res(
                        ctx.status(204),
                    );
                }),
            ],

            error: [
                rest.post(`${baseUrl}/api/project`, (req, res, ctx) => {
                    return res(
                        ctx.status(409),
                        ctx.json({ 
                            message: "Project handle already exists",
                            code: "HANDLE_EXISTS", 
                        }),
                    );
                }),

                rest.put(`${baseUrl}/api/project/:id`, (req, res, ctx) => {
                    return res(
                        ctx.status(403),
                        ctx.json({ 
                            message: "You do not have permission to update this project",
                            code: "PERMISSION_DENIED", 
                        }),
                    );
                }),

                rest.get(`${baseUrl}/api/project/:handle`, (req, res, ctx) => {
                    return res(
                        ctx.status(404),
                        ctx.json({ 
                            message: "Project not found",
                            code: "PROJECT_NOT_FOUND", 
                        }),
                    );
                }),
            ],

            loading: [
                rest.post(`${baseUrl}/api/project`, (req, res, ctx) => {
                    return res(
                        ctx.delay(2000), // 2 second delay
                        ctx.status(201),
                        ctx.json(this.createMockResponse()),
                    );
                }),
            ],

            networkError: [
                rest.post(`${baseUrl}/api/project`, (req, res, ctx) => {
                    return res.networkError("Network connection failed");
                }),
            ],
        };
    }

    /**
     * Create UI state fixtures for different scenarios
     */
    createUIState(
        state: "loading" | "error" | "success" | "empty" | "ownerView" | "viewerView" = "empty", 
        data?: any,
    ): ProjectUIState {
        switch (state) {
            case "loading":
                return {
                    isLoading: true,
                    project: null,
                    currentVersion: null,
                    error: null,
                    isOwner: false,
                    canEdit: false,
                    canDelete: false,
                    completionPercentage: 0,
                };

            case "error":
                return {
                    isLoading: false,
                    project: null,
                    currentVersion: null,
                    error: data?.message || "Failed to load project",
                    isOwner: false,
                    canEdit: false,
                    canDelete: false,
                    completionPercentage: 0,
                };

            case "success":
            case "viewerView":
                const project = data || this.createMockResponse();
                return {
                    isLoading: false,
                    project,
                    currentVersion: project.versions[0],
                    error: null,
                    isOwner: false,
                    canEdit: false,
                    canDelete: false,
                    completionPercentage: project.versions[0].isComplete ? 100 : 0,
                };

            case "ownerView":
                const ownedProject = data || this.createMockResponse();
                return {
                    isLoading: false,
                    project: ownedProject,
                    currentVersion: ownedProject.versions[0],
                    error: null,
                    isOwner: true,
                    canEdit: true,
                    canDelete: true,
                    completionPercentage: ownedProject.versions[0].isComplete ? 100 : 50,
                };

            case "empty":
            default:
                return {
                    isLoading: false,
                    project: null,
                    currentVersion: null,
                    error: null,
                    isOwner: false,
                    canEdit: false,
                    canDelete: false,
                    completionPercentage: 0,
                };
        }
    }

    /**
     * Create a team-owned project
     */
    createTeamProject(teamId: string, teamName = "Test Team"): Resource {
        return this.createMockResponse({
            owner: {
                __typename: "Team",
                id: teamId,
                handle: `team_${teamId}`,
                name: teamName,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                isPrivate: false,
                profileImage: null,
                bannerImage: null,
                permissions: JSON.stringify({
                    canUpdate: true,
                    canDelete: true,
                    canInviteMembers: true,
                }),
                tags: [],
                tagsCount: 0,
                translations: [],
                translationsCount: 0,
                members: [],
                membersCount: 1,
                projects: [],
                projectsCount: 1,
                isOpenToNewMembers: true,
                routines: [],
                routinesCount: 0,
                standards: [],
                standardsCount: 0,
                resourceLists: [],
                resourceListsCount: 0,
                you: {
                    __typename: "TeamYou",
                    canAddMembers: true,
                    canDelete: true,
                    canRemoveMembers: true,
                    canUpdate: true,
                    canReport: false,
                    isBookmarked: false,
                    isReacted: false,
                    reaction: null,
                    yourMemberRole: MemberRole.Owner,
                },
            },
        });
    }

    /**
     * Create a completed project
     */
    createCompletedProject(completionDate: Date = new Date()): Resource {
        const project = this.createMockResponse();
        project.versions[0].isComplete = true;
        project.versions[0].completedAt = completionDate.toISOString();
        return project;
    }

    /**
     * Create a project with multiple versions
     */
    createWithVersions(versionCount = 3): Resource {
        const project = this.createMockResponse();
        const additionalVersions = Array.from({ length: versionCount - 1 }, (_, i) => ({
            ...project.versions[0],
            id: this.generateId(),
            versionLabel: `${i + 2}.0.0`,
            createdAt: new Date(Date.now() - (i + 1) * 86400000).toISOString(), // Each version 1 day older
        }));
        
        project.versions = [project.versions[0], ...additionalVersions];
        project.versionsCount = versionCount;
        return project;
    }

    /**
     * Create test cases for various scenarios
     */
    createTestCases() {
        return [
            {
                name: "Valid project creation",
                formData: this.createFormData("minimal"),
                shouldSucceed: true,
            },
            {
                name: "Complete project profile",
                formData: this.createFormData("complete"),
                shouldSucceed: true,
            },
            {
                name: "Empty handle",
                formData: { ...this.createFormData("minimal"), handle: "" },
                shouldSucceed: false,
                expectedError: "handle is a required field",
            },
            {
                name: "Empty name",
                formData: { ...this.createFormData("minimal"), name: "" },
                shouldSucceed: false,
                expectedError: "name is a required field",
            },
            {
                name: "Private project",
                formData: this.createFormData("privateProject"),
                shouldSucceed: true,
            },
            {
                name: "Team project",
                formData: this.createFormData("teamProject"),
                shouldSucceed: true,
            },
        ];
    }
}

/**
 * Default factory instance for easy importing
 */
export const projectFixtures = new ProjectFixtureFactory();

/**
 * Specific test scenarios for common use cases
 */
export const projectTestScenarios = {
    // Basic scenarios
    minimalProject: () => projectFixtures.createFormData("minimal"),
    completeProject: () => projectFixtures.createFormData("complete"),
    invalidProject: () => projectFixtures.createFormData("invalid"),
    
    // Project type scenarios
    privateProject: () => projectFixtures.createFormData("privateProject"),
    teamProject: () => projectFixtures.createFormData("teamProject"),
    completedProject: () => projectFixtures.createFormData("completedProject"),
    taggedProject: () => projectFixtures.createFormData("withTags"),
    multiVersionProject: () => projectFixtures.createFormData("multiVersion"),
    
    // Mock responses
    basicProjectResponse: () => projectFixtures.createMockResponse(),
    teamProjectResponse: () => projectFixtures.createTeamProject("team123"),
    completedProjectResponse: () => projectFixtures.createCompletedProject(),
    multiVersionResponse: () => projectFixtures.createWithVersions(5),
    
    // UI state scenarios
    loadingState: () => projectFixtures.createUIState("loading"),
    errorState: (message?: string) => projectFixtures.createUIState("error", { message }),
    viewerState: (project?: Resource) => projectFixtures.createUIState("viewerView", project),
    ownerState: (project?: Resource) => projectFixtures.createUIState("ownerView", project),
    emptyState: () => projectFixtures.createUIState("empty"),
    
    // Test data sets
    allTestCases: () => projectFixtures.createTestCases(),
    
    // MSW handlers
    successHandlers: () => projectFixtures.createMSWHandlers().success,
    errorHandlers: () => projectFixtures.createMSWHandlers().error,
    loadingHandlers: () => projectFixtures.createMSWHandlers().loading,
    networkErrorHandlers: () => projectFixtures.createMSWHandlers().networkError,
};
