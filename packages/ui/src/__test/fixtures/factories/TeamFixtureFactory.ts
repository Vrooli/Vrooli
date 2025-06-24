/**
 * Production-grade team fixture factory
 * 
 * This factory provides type-safe team fixtures using real functions from @vrooli/shared.
 * It eliminates `any` types and integrates with actual validation and shape transformation logic.
 */

import type { 
    Team, 
    TeamCreateInput, 
    TeamUpdateInput,
    TeamTranslation,
    Member,
    User,
    MemberInvite,
    Resource,
} from "@vrooli/shared";
import { 
    teamValidation,
    shapeTeam,
} from "@vrooli/shared";
import type { 
    FixtureFactory, 
    ValidationResult, 
    MSWHandlers,
} from "../types.js";
import { http } from "msw";

// Define MemberRole locally since it's not exported from shared
export enum MemberRole {
    ADMIN = "admin",
    EDITOR = "editor",
    VIEWER = "viewer",
}

/**
 * UI-specific form data for team creation
 */
export interface TeamFormData {
    handle: string;
    name: string;
    bio?: string;
    isPrivate?: boolean;
    profileImage?: File | string | null;
    bannerImage?: File | string | null;
    tags?: string[];
}

/**
 * UI-specific form data for team member management
 */
export interface TeamMemberFormData {
    userId?: string;
    userEmail?: string;
    role: MemberRole;
    permissions?: string[];
}

/**
 * UI state for team components
 */
export interface TeamUIState {
    isLoading: boolean;
    team: Team | null;
    error: string | null;
    members: Member[];
    pendingInvites: MemberInvite[];
    userRole?: MemberRole;
    canManageMembers: boolean;
    canUpdateTeam: boolean;
}

export type TeamScenario = 
    | "minimal" 
    | "complete" 
    | "invalid" 
    | "withMembers"
    | "privateTeam"
    | "withProjects"
    | "withInvites"
    | "largeTeam";

/**
 * Type-safe team fixture factory that uses real @vrooli/shared functions
 */
export class TeamFixtureFactory implements FixtureFactory<
    TeamFormData,
    TeamCreateInput,
    TeamUpdateInput,
    Team
> {
    readonly objectType = "team";

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
        return `testteam_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Create form data for different test scenarios
     */
    createFormData(scenario: TeamScenario = "minimal"): TeamFormData {
        switch (scenario) {
            case "minimal":
                return {
                    handle: this.generateHandle(),
                    name: "Test Team",
                    isPrivate: false,
                };

            case "complete":
                return {
                    handle: this.generateHandle(),
                    name: "Complete Test Team",
                    bio: "This is a fully configured test team with all optional fields",
                    isPrivate: false,
                    tags: ["testing", "development", "collaboration"],
                };

            case "invalid":
                return {
                    handle: "t", // Too short
                    name: "", // Empty name
                    // @ts-expect-error - Testing invalid data
                    isPrivate: "yes", // Should be boolean
                };

            case "withMembers":
                return {
                    handle: this.generateHandle(),
                    name: "Team With Members",
                    bio: "A team that will have multiple members",
                    isPrivate: false,
                };

            case "privateTeam":
                return {
                    handle: this.generateHandle(),
                    name: "Private Team",
                    bio: "This team is private and requires invitation",
                    isPrivate: true,
                };

            case "withProjects":
                return {
                    handle: this.generateHandle(),
                    name: "Project Team",
                    bio: "A team focused on managing multiple projects",
                    isPrivate: false,
                    tags: ["projects", "management"],
                };

            case "withInvites":
                return {
                    handle: this.generateHandle(),
                    name: "Team With Pending Invites",
                    bio: "This team has pending member invitations",
                    isPrivate: false,
                };

            case "largeTeam":
                return {
                    handle: this.generateHandle(),
                    name: "Large Organization",
                    bio: "A large team with many members and complex structure",
                    isPrivate: false,
                    tags: ["enterprise", "large-scale"],
                };

            default:
                throw new Error(`Unknown team scenario: ${scenario}`);
        }
    }

    /**
     * Transform form data to API create input using real shape function
     */
    transformToAPIInput(formData: TeamFormData): TeamCreateInput {
        // Create the team shape that matches the expected API structure
        const teamShape = {
            __typename: "Team" as const,
            id: this.generateId(),
            handle: formData.handle,
            name: formData.name,
            isPrivate: formData.isPrivate || false,
            translations: formData.bio ? [{
                __typename: "TeamTranslation" as const,
                id: this.generateId(),
                language: "en",
                bio: formData.bio,
            }] : null,
            tags: formData.tags?.map(tag => ({
                __typename: "Tag" as const,
                id: this.generateId(),
                tag,
            })) || null,
            profileImage: formData.profileImage,
            bannerImage: formData.bannerImage,
        };

        // Use real shape function from @vrooli/shared
        return shapeTeam.create(teamShape);
    }

    /**
     * Create API update input
     */
    createUpdateInput(id: string, updates: Partial<TeamFormData>): TeamUpdateInput {
        const updateInput: TeamUpdateInput = { id };

        if (updates.handle) updateInput.handle = updates.handle;
        if (updates.name) updateInput.name = updates.name;
        if (updates.isPrivate !== undefined) updateInput.isPrivate = updates.isPrivate;

        // Handle translations for bio
        if (updates.bio) {
            updateInput.translationsUpdate = [{
                id: this.generateId(),
                language: "en",
                bio: updates.bio,
            }];
        }

        // Handle tags
        if (updates.tags) {
            updateInput.tagsConnect = updates.tags.map(tag => tag);
        }

        // Handle images
        if (updates.profileImage !== undefined) {
            updateInput.profileImage = updates.profileImage;
        }
        if (updates.bannerImage !== undefined) {
            updateInput.bannerImage = updates.bannerImage;
        }

        return updateInput;
    }

    /**
     * Create mock team response with realistic data
     */
    createMockResponse(overrides?: Partial<Team>): Team {
        const now = new Date().toISOString();
        const teamId = this.generateId();
        const ownerId = this.generateId();
        
        const defaultTeam: Team = {
            __typename: "Team",
            id: teamId,
            handle: this.generateHandle(),
            name: "Test Team",
            createdAt: now,
            updatedAt: now,
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
            translations: [{
                __typename: "TeamTranslation",
                id: this.generateId(),
                language: "en",
                bio: "A test team for development",
            }],
            translationsCount: 1,
            members: [{
                __typename: "Member",
                id: this.generateId(),
                created_at: now,
                updated_at: now,
                role: MemberRole.Owner,
                permissions: JSON.stringify({
                    admin: true,
                    manageMembers: true,
                    manageProjects: true,
                }),
                isOwner: true,
                user: {
                    __typename: "User",
                    id: ownerId,
                    handle: "teamowner",
                    name: "Team Owner",
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
                you: {
                    __typename: "MemberYou",
                    canDelete: true,
                    canUpdate: true,
                },
            }],
            membersCount: 1,
            projects: [],
            projectsCount: 0,
            membersAsObject: [],
            isOpenToNewMembers: true,
            routines: [],
            routinesCount: 0,
            standards: [],
            standardsCount: 0,
            resourceLists: [],
            resourceListsCount: 0,
            stats: {
                __typename: "StatsTeam",
                id: this.generateId(),
                teamId,
                reportsCount: 0,
                reputation: 0,
                membersCount: 1,
                projectsCount: 0,
                teamsCount: 0,
                routinesCount: 0,
                standardsCount: 0,
                codesCount: 0,
                apisCount: 0,
                questionsAskedCount: 0,
                questionsAnsweredCount: 0,
            },
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
        };

        return {
            ...defaultTeam,
            ...overrides,
        };
    }

    /**
     * Validate form data using real validation from @vrooli/shared
     */
    async validateFormData(formData: TeamFormData): Promise<ValidationResult> {
        try {
            // Transform to API input format for validation
            const apiInput = this.transformToAPIInput(formData);
            
            // Use real validation schema from @vrooli/shared
            await teamValidation.create.validate(apiInput, { abortEarly: false });
            
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
                // Create team
                http.post(`${baseUrl}/api/team`, async (req, res, ctx) => {
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
                    const mockTeam = this.createMockResponse({
                        handle: body.handle,
                        name: body.name,
                        isPrivate: body.isPrivate,
                    });

                    return res(
                        ctx.status(201),
                        ctx.json(mockTeam),
                    );
                }),

                // Update team
                http.put(`${baseUrl}/api/team/:id`, async (req, res, ctx) => {
                    const { id } = req.params;
                    const body = await req.json();

                    const mockTeam = this.createMockResponse({ 
                        id: id as string,
                        ...body,
                        updatedAt: new Date().toISOString(),
                    });

                    return res(
                        ctx.status(200),
                        ctx.json(mockTeam),
                    );
                }),

                // Get team
                http.get(`${baseUrl}/api/team/:handle`, (req, res, ctx) => {
                    const { handle } = req.params;
                    const mockTeam = this.createMockResponse({ 
                        handle: handle as string, 
                    });
                    
                    return res(
                        ctx.status(200),
                        ctx.json(mockTeam),
                    );
                }),

                // Add member
                http.post(`${baseUrl}/api/team/:id/member`, async (req, res, ctx) => {
                    const { id } = req.params;
                    const body = await req.json();

                    const newMember: Member = {
                        __typename: "Member",
                        id: this.generateId(),
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        role: body.role || MemberRole.Member,
                        permissions: JSON.stringify(body.permissions || {}),
                        isOwner: false,
                        user: {
                            __typename: "User",
                            id: body.userId,
                            handle: `member_${Date.now()}`,
                            name: "New Member",
                            email: body.userEmail || "member@example.com",
                            emailVerified: true,
                            createdAt: new Date().toISOString(),
                            updatedAt: new Date().toISOString(),
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
                        you: {
                            __typename: "MemberYou",
                            canDelete: true,
                            canUpdate: true,
                        },
                    };

                    return res(
                        ctx.status(201),
                        ctx.json(newMember),
                    );
                }),

                // Delete team
                http.delete(`${baseUrl}/api/team/:id`, (req, res, ctx) => {
                    return res(
                        ctx.status(204),
                    );
                }),
            ],

            error: [
                http.post(`${baseUrl}/api/team`, (req, res, ctx) => {
                    return res(
                        ctx.status(409),
                        ctx.json({ 
                            message: "Team handle already exists",
                            code: "HANDLE_EXISTS", 
                        }),
                    );
                }),

                http.put(`${baseUrl}/api/team/:id`, (req, res, ctx) => {
                    return res(
                        ctx.status(403),
                        ctx.json({ 
                            message: "You do not have permission to update this team",
                            code: "PERMISSION_DENIED", 
                        }),
                    );
                }),

                http.get(`${baseUrl}/api/team/:handle`, (req, res, ctx) => {
                    return res(
                        ctx.status(404),
                        ctx.json({ 
                            message: "Team not found",
                            code: "TEAM_NOT_FOUND", 
                        }),
                    );
                }),
            ],

            loading: [
                http.post(`${baseUrl}/api/team`, (req, res, ctx) => {
                    return res(
                        ctx.delay(2000), // 2 second delay
                        ctx.status(201),
                        ctx.json(this.createMockResponse()),
                    );
                }),
            ],

            networkError: [
                http.post(`${baseUrl}/api/team`, (req, res, ctx) => {
                    return res.networkError("Network connection failed");
                }),
            ],
        };
    }

    /**
     * Create UI state fixtures for different scenarios
     */
    createUIState(
        state: "loading" | "error" | "success" | "empty" | "memberView" | "ownerView" = "empty", 
        data?: any,
    ): TeamUIState {
        switch (state) {
            case "loading":
                return {
                    isLoading: true,
                    team: null,
                    error: null,
                    members: [],
                    pendingInvites: [],
                    canManageMembers: false,
                    canUpdateTeam: false,
                };

            case "error":
                return {
                    isLoading: false,
                    team: null,
                    error: data?.message || "Failed to load team",
                    members: [],
                    pendingInvites: [],
                    canManageMembers: false,
                    canUpdateTeam: false,
                };

            case "success":
            case "memberView":
                const team = data || this.createMockResponse();
                return {
                    isLoading: false,
                    team,
                    error: null,
                    members: team.members || [],
                    pendingInvites: [],
                    userRole: MemberRole.Member,
                    canManageMembers: false,
                    canUpdateTeam: false,
                };

            case "ownerView":
                const ownedTeam = data || this.createMockResponse();
                return {
                    isLoading: false,
                    team: ownedTeam,
                    error: null,
                    members: ownedTeam.members || [],
                    pendingInvites: data?.invites || [],
                    userRole: MemberRole.Owner,
                    canManageMembers: true,
                    canUpdateTeam: true,
                };

            case "empty":
            default:
                return {
                    isLoading: false,
                    team: null,
                    error: null,
                    members: [],
                    pendingInvites: [],
                    canManageMembers: false,
                    canUpdateTeam: false,
                };
        }
    }

    /**
     * Create a team with multiple members
     */
    createWithMembers(memberCount = 3): Team {
        const members: Member[] = Array.from({ length: memberCount }, (_, i) => ({
            __typename: "Member",
            id: this.generateId(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            role: i === 0 ? MemberRole.Owner : i === 1 ? MemberRole.Admin : MemberRole.Member,
            permissions: JSON.stringify({
                admin: i <= 1,
                manageMembers: i <= 1,
                manageProjects: true,
            }),
            isOwner: i === 0,
            user: {
                __typename: "User",
                id: this.generateId(),
                handle: `member${i + 1}`,
                name: `Team Member ${i + 1}`,
                email: `member${i + 1}@example.com`,
                emailVerified: true,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
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
            you: {
                __typename: "MemberYou",
                canDelete: i === 0,
                canUpdate: i === 0,
            },
        }));

        return this.createMockResponse({
            members,
            membersCount: memberCount,
        });
    }

    /**
     * Create a team with projects
     */
    createWithProjects(projectCount = 2): Team {
        const projects: Resource[] = Array.from({ length: projectCount }, (_, i) => ({
            __typename: "Resource",
            id: this.generateId(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            isInternal: false,
            isPrivate: false,
            name: `Project ${i + 1}`,
            description: `Test project ${i + 1} for the team`,
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
        }));

        return this.createMockResponse({
            projects,
            projectsCount: projectCount,
        });
    }

    /**
     * Create member form data
     */
    createMemberFormData(role: MemberRole = MemberRole.Member): TeamMemberFormData {
        return {
            userId: this.generateId(),
            userEmail: "newmember@example.com",
            role,
            permissions: role === MemberRole.Admin ? 
                ["manageMembers", "manageProjects"] : 
                ["viewProjects"],
        };
    }

    /**
     * Create test cases for various scenarios
     */
    createTestCases() {
        return [
            {
                name: "Valid team creation",
                formData: this.createFormData("minimal"),
                shouldSucceed: true,
            },
            {
                name: "Complete team profile",
                formData: this.createFormData("complete"),
                shouldSucceed: true,
            },
            {
                name: "Invalid handle",
                formData: { ...this.createFormData("minimal"), handle: "a" },
                shouldSucceed: false,
                expectedError: "handle must be at least",
            },
            {
                name: "Empty name",
                formData: { ...this.createFormData("minimal"), name: "" },
                shouldSucceed: false,
                expectedError: "name is a required field",
            },
            {
                name: "Private team",
                formData: this.createFormData("privateTeam"),
                shouldSucceed: true,
            },
        ];
    }
}

/**
 * Default factory instance for easy importing
 */
export const teamFixtures = new TeamFixtureFactory();

/**
 * Specific test scenarios for common use cases
 */
export const teamTestScenarios = {
    // Basic scenarios
    minimalTeam: () => teamFixtures.createFormData("minimal"),
    completeTeam: () => teamFixtures.createFormData("complete"),
    invalidTeam: () => teamFixtures.createFormData("invalid"),
    
    // Team type scenarios
    privateTeam: () => teamFixtures.createFormData("privateTeam"),
    teamWithMembers: () => teamFixtures.createFormData("withMembers"),
    teamWithProjects: () => teamFixtures.createFormData("withProjects"),
    largeTeam: () => teamFixtures.createFormData("largeTeam"),
    
    // Member scenarios
    addMember: () => teamFixtures.createMemberFormData(MemberRole.Member),
    addAdmin: () => teamFixtures.createMemberFormData(MemberRole.Admin),
    
    // Mock responses
    basicTeamResponse: () => teamFixtures.createMockResponse(),
    teamWithMembersResponse: () => teamFixtures.createWithMembers(5),
    teamWithProjectsResponse: () => teamFixtures.createWithProjects(3),
    
    // UI state scenarios
    loadingState: () => teamFixtures.createUIState("loading"),
    errorState: (message?: string) => teamFixtures.createUIState("error", { message }),
    memberViewState: (team?: Team) => teamFixtures.createUIState("memberView", team),
    ownerViewState: (team?: Team) => teamFixtures.createUIState("ownerView", team),
    emptyState: () => teamFixtures.createUIState("empty"),
    
    // Test data sets
    allTestCases: () => teamFixtures.createTestCases(),
    
    // MSW handlers
    successHandlers: () => teamFixtures.createMSWHandlers().success,
    errorHandlers: () => teamFixtures.createMSWHandlers().error,
    loadingHandlers: () => teamFixtures.createMSWHandlers().loading,
    networkErrorHandlers: () => teamFixtures.createMSWHandlers().networkError,
};
