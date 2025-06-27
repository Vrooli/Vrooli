/* c8 ignore start */
/**
 * Team API Response Fixtures
 * 
 * Comprehensive fixtures for team management endpoints including
 * team creation, member management, and collaboration features.
 */

import type {
    Team,
    TeamCreateInput,
    TeamUpdateInput,
    Member,
    User,
    Project,
    Meeting,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";
import { userResponseFactory } from "./userResponses.js";

/**
 * Team API response factory
 * 
 * Handles team creation, updates, member management, and team resources.
 */
export class TeamResponseFactory extends BaseAPIResponseFactory<
    Team,
    TeamCreateInput,
    TeamUpdateInput
> {
    protected readonly entityName = "team";

    /**
     * Create mock team data
     */
    createMockData(options?: MockDataOptions): Team {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const teamId = options?.overrides?.id || generatePK().toString();

        const baseTeam: Team = {
            __typename: "Team",
            id: teamId,
            created_at: now,
            updated_at: now,
            bannerImage: null,
            handle: options?.overrides?.handle || `team_${teamId.slice(0, 8)}`,
            isOpenToNewMembers: true,
            isPrivate: false,
            name: options?.overrides?.name || "Test Team",
            profileImage: null,
            bookmarks: 0,
            reportsReceivedCount: 0,
            you: {
                canAddMembers: false,
                canDelete: false,
                canBookmark: true,
                canReport: true,
                canUpdate: false,
                canRead: true,
                isBookmarked: false,
                isViewed: false,
                role: null,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            // Create team with members and resources
            const owner = userResponseFactory.createMockData();
            const members = this.createMembers(teamId, owner, 3);

            return {
                ...baseTeam,
                bannerImage: "https://example.com/team-banner.jpg",
                profileImage: "https://example.com/team-logo.jpg",
                bookmarks: 25,
                resourceLists: [{
                    __typename: "ResourceList",
                    id: generatePK().toString(),
                    created_at: now,
                    updated_at: now,
                    index: 0,
                    isUsedFor: "TeamResource",
                    translations: [{
                        __typename: "ResourceListTranslation",
                        id: generatePK().toString(),
                        language: "en",
                        name: "Team Resources",
                        description: "Shared resources for the team",
                    }],
                    resources: [],
                }],
                members,
                membersCount: members.length,
                translations: [{
                    __typename: "TeamTranslation",
                    id: generatePK().toString(),
                    language: "en",
                    bio: "We're a collaborative team focused on innovation and quality.",
                }],
                you: {
                    canAddMembers: true,
                    canDelete: true,
                    canBookmark: true,
                    canReport: false,
                    canUpdate: true,
                    canRead: true,
                    isBookmarked: false,
                    isViewed: true,
                    role: "Owner",
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseTeam,
            ...options?.overrides,
        };
    }

    /**
     * Create team from input
     */
    createFromInput(input: TeamCreateInput): Team {
        const now = new Date().toISOString();
        const teamId = generatePK().toString();

        return {
            __typename: "Team",
            id: teamId,
            created_at: now,
            updated_at: now,
            bannerImage: input.bannerImage || null,
            handle: input.handle || `team_${teamId.slice(0, 8)}`,
            isOpenToNewMembers: input.isOpenToNewMembers ?? true,
            isPrivate: input.isPrivate || false,
            name: input.name,
            profileImage: input.profileImage || null,
            bookmarks: 0,
            reportsReceivedCount: 0,
            resourceLists: input.resourceListsCreate?.map((rl, index) => ({
                __typename: "ResourceList" as const,
                id: generatePK().toString(),
                created_at: now,
                updated_at: now,
                index,
                isUsedFor: "TeamResource",
                translations: rl.translationsCreate?.map(t => ({
                    __typename: "ResourceListTranslation" as const,
                    id: generatePK().toString(),
                    language: t.language,
                    name: t.name || "",
                    description: t.description || null,
                })) || [],
                resources: [],
            })) || [],
            translations: input.translationsCreate?.map(t => ({
                __typename: "TeamTranslation" as const,
                id: generatePK().toString(),
                language: t.language,
                bio: t.bio || null,
            })) || [],
            members: [],
            membersCount: 1, // Creator is automatically a member
            you: {
                canAddMembers: true,
                canDelete: true,
                canBookmark: true,
                canReport: false,
                canUpdate: true,
                canRead: true,
                isBookmarked: false,
                isViewed: true,
                role: "Owner",
            },
        };
    }

    /**
     * Update team from input
     */
    updateFromInput(existing: Team, input: TeamUpdateInput): Team {
        const updates: Partial<Team> = {
            updated_at: new Date().toISOString(),
        };

        if (input.bannerImage !== undefined) updates.bannerImage = input.bannerImage;
        if (input.handle !== undefined) updates.handle = input.handle;
        if (input.isOpenToNewMembers !== undefined) updates.isOpenToNewMembers = input.isOpenToNewMembers;
        if (input.isPrivate !== undefined) updates.isPrivate = input.isPrivate;
        if (input.name !== undefined) updates.name = input.name;
        if (input.profileImage !== undefined) updates.profileImage = input.profileImage;

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
    async validateCreateInput(input: TeamCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.name) {
            errors.name = "Team name is required";
        } else if (input.name.length < 2) {
            errors.name = "Team name must be at least 2 characters";
        }

        if (input.handle && !/^[a-zA-Z0-9_-]+$/.test(input.handle)) {
            errors.handle = "Handle can only contain letters, numbers, underscores, and hyphens";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: TeamUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.name !== undefined && input.name.length < 2) {
            errors.name = "Team name must be at least 2 characters";
        }

        if (input.handle !== undefined && !/^[a-zA-Z0-9_-]+$/.test(input.handle)) {
            errors.handle = "Handle can only contain letters, numbers, underscores, and hyphens";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create team members
     */
    private createMembers(teamId: string, owner: User, additionalCount = 2): Member[] {
        const now = new Date().toISOString();
        const members: Member[] = [];

        // Add owner
        members.push({
            __typename: "Member",
            id: generatePK().toString(),
            created_at: now,
            updated_at: now,
            isAdmin: true,
            role: "Owner",
            user: owner,
            team: { id: teamId } as Team,
        });

        // Add additional members
        for (let i = 0; i < additionalCount; i++) {
            const user = userResponseFactory.createMockData();
            members.push({
                __typename: "Member",
                id: generatePK().toString(),
                created_at: now,
                updated_at: now,
                isAdmin: i === 0, // First additional member is admin
                role: i === 0 ? "Admin" : "Member",
                user,
                team: { id: teamId } as Team,
            });
        }

        return members;
    }

    /**
     * Create team with projects
     */
    createTeamWithProjects(projectCount = 3): Team {
        const team = this.createMockData({ scenario: "complete" });
        const now = new Date().toISOString();
        const projects: Partial<Project>[] = [];

        for (let i = 0; i < projectCount; i++) {
            projects.push({
                __typename: "Project",
                id: generatePK().toString(),
                created_at: now,
                updated_at: now,
                handle: `project${i + 1}`,
                name: `Project ${i + 1}`,
                isPrivate: false,
                permissions: JSON.stringify({ canRead: true }),
                owner: { __typename: "Team", id: team.id } as Team,
            });
        }

        return {
            ...team,
            projects: projects as Project[],
            projectsCount: projectCount,
        };
    }

    /**
     * Create team with meetings
     */
    createTeamWithMeetings(meetingCount = 2): Team {
        const team = this.createMockData({ scenario: "complete" });
        const now = new Date().toISOString();
        const meetings: Partial<Meeting>[] = [];

        for (let i = 0; i < meetingCount; i++) {
            meetings.push({
                __typename: "Meeting",
                id: generatePK().toString(),
                created_at: now,
                updated_at: now,
                openToAnyoneWithInvite: false,
                showOnTeamProfile: true,
                team: { __typename: "Team", id: team.id } as Team,
                translations: [{
                    __typename: "MeetingTranslation",
                    id: generatePK().toString(),
                    language: "en",
                    name: `Team Meeting ${i + 1}`,
                    description: `Regular team sync meeting ${i + 1}`,
                }],
            });
        }

        return {
            ...team,
            meetings: meetings as Meeting[],
            meetingsCount: meetingCount,
        };
    }
}

/**
 * Pre-configured team response scenarios
 */
export const teamResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<TeamCreateInput>) => {
        const factory = new TeamResponseFactory();
        const defaultInput: TeamCreateInput = {
            name: "New Team",
            handle: "new-team",
            isOpenToNewMembers: true,
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (team?: Team) => {
        const factory = new TeamResponseFactory();
        return factory.createSuccessResponse(
            team || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new TeamResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Team, updates?: Partial<TeamUpdateInput>) => {
        const factory = new TeamResponseFactory();
        const team = existing || factory.createMockData({ scenario: "complete" });
        const input: TeamUpdateInput = {
            id: team.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(team, input),
        );
    },

    withProjectsSuccess: (projectCount?: number) => {
        const factory = new TeamResponseFactory();
        return factory.createSuccessResponse(
            factory.createTeamWithProjects(projectCount),
        );
    },

    withMeetingsSuccess: (meetingCount?: number) => {
        const factory = new TeamResponseFactory();
        return factory.createSuccessResponse(
            factory.createTeamWithMeetings(meetingCount),
        );
    },

    listSuccess: (teams?: Team[]) => {
        const factory = new TeamResponseFactory();
        const DEFAULT_COUNT = 10;
        return factory.createPaginatedResponse(
            teams || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: teams?.length || DEFAULT_COUNT },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new TeamResponseFactory();
        return factory.createValidationErrorResponse({
            name: "Team name is required",
            handle: "Handle is already taken",
        });
    },

    notFoundError: (teamId?: string) => {
        const factory = new TeamResponseFactory();
        return factory.createNotFoundErrorResponse(
            teamId || "non-existent-team",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new TeamResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["team:update"],
        );
    },

    memberLimitError: () => {
        const factory = new TeamResponseFactory();
        const MEMBER_LIMIT = 100;
        return factory.createBusinessErrorResponse("limit", {
            resource: "members",
            limit: MEMBER_LIMIT,
            current: MEMBER_LIMIT,
            message: "Team has reached the maximum number of members",
        });
    },

    privateTeamError: () => {
        const factory = new TeamResponseFactory();
        return factory.createBusinessErrorResponse("state", {
            reason: "Team is private",
            message: "You must be invited to join this private team",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new TeamResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate = 0.1) {
            return new TeamResponseFactory().createMSWHandlers({ errorRate });
        },
        withDelay: function createWithDelay(delay?: number) {
            const DEFAULT_DELAY_MS = 500;
            return new TeamResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const teamResponseFactory = new TeamResponseFactory();
