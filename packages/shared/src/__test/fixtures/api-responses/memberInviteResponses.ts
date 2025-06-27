/* c8 ignore start */
/**
 * Member Invite API Response Fixtures
 * 
 * Comprehensive fixtures for team member invitation management including
 * invite creation, acceptance/decline, role assignment, and invitation tracking.
 */

import type {
    MemberInvite,
    MemberInviteCreateInput,
    MemberInviteUpdateInput,
    MemberInviteStatus,
    Team,
    User,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";
import { teamResponseFactory } from "./teamResponses.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_MESSAGE_LENGTH = 500;
const INVITE_EXPIRY_DAYS = 7;
const MILLISECONDS_PER_HOUR = 60 * 60 * 1000;
const MILLISECONDS_PER_DAY = 24 * MILLISECONDS_PER_HOUR;
const HOURS_IN_2 = 2;
const HOURS_IN_6 = 6;
const DAYS_IN_1 = 1;
const DAYS_IN_3 = 3;
const DAYS_IN_7 = 7;

// Member invite statuses
const MEMBER_INVITE_STATUSES = ["Pending", "Accepted", "Declined"] as const;

// Team roles (typical team management roles)
const TEAM_ROLES = ["member", "admin", "owner"] as const;

/**
 * Member Invite API response factory
 */
export class MemberInviteResponseFactory extends BaseAPIResponseFactory<
    MemberInvite,
    MemberInviteCreateInput,
    MemberInviteUpdateInput
> {
    protected readonly entityName = "member_invite";

    /**
     * Create mock member invite data
     */
    createMockData(options?: MockDataOptions): MemberInvite {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const inviteId = options?.overrides?.id || generatePK().toString();

        const baseMemberInvite: MemberInvite = {
            __typename: "MemberInvite",
            id: inviteId,
            created_at: now,
            updated_at: now,
            message: "You've been invited to join our team!",
            status: "Pending",
            team: teamResponseFactory.createMockData(),
            user: userResponseFactory.createMockData(),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseMemberInvite,
                message: scenario === "edge-case" 
                    ? null 
                    : "We'd love to have you join our development team! You'll be working on exciting projects with a great group of people.",
                status: scenario === "edge-case" ? "Declined" : "Accepted",
                team: teamResponseFactory.createMockData({ scenario: "complete" }),
                user: userResponseFactory.createMockData({ scenario: "complete" }),
                updated_at: scenario === "edge-case" 
                    ? new Date(Date.now() - (DAYS_IN_3 * MILLISECONDS_PER_DAY)).toISOString() // Updated 3 days ago
                    : new Date(Date.now() - (HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(), // Updated 2 hours ago
                you: {
                    canDelete: scenario !== "edge-case",
                    canUpdate: scenario !== "edge-case",
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseMemberInvite,
            ...options?.overrides,
        };
    }

    /**
     * Create member invite from input
     */
    createFromInput(input: MemberInviteCreateInput): MemberInvite {
        const now = new Date().toISOString();
        const inviteId = generatePK().toString();

        return {
            __typename: "MemberInvite",
            id: inviteId,
            created_at: now,
            updated_at: now,
            message: input.message || null,
            status: "Pending",
            team: teamResponseFactory.createMockData({ overrides: { id: input.teamConnect } }),
            user: userResponseFactory.createMockData({ overrides: { id: input.userConnect } }),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };
    }

    /**
     * Update member invite from input
     */
    updateFromInput(existing: MemberInvite, input: MemberInviteUpdateInput): MemberInvite {
        const updates: Partial<MemberInvite> = {
            updated_at: new Date().toISOString(),
        };

        if (input.message !== undefined) updates.message = input.message;
        if (input.status !== undefined) updates.status = input.status;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: MemberInviteCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.teamConnect) {
            errors.teamConnect = "Team ID is required";
        }

        if (!input.userConnect) {
            errors.userConnect = "User ID is required";
        }

        if (input.message && input.message.length > MAX_MESSAGE_LENGTH) {
            errors.message = `Message must be ${MAX_MESSAGE_LENGTH} characters or less`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: MemberInviteUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.message !== undefined && input.message && input.message.length > MAX_MESSAGE_LENGTH) {
            errors.message = `Message must be ${MAX_MESSAGE_LENGTH} characters or less`;
        }

        if (input.status !== undefined && !MEMBER_INVITE_STATUSES.includes(input.status as any)) {
            errors.status = `Status must be one of: ${MEMBER_INVITE_STATUSES.join(", ")}`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create member invites for different statuses
     */
    createMemberInvitesForAllStatuses(): MemberInvite[] {
        return MEMBER_INVITE_STATUSES.map((status, index) =>
            this.createMockData({
                overrides: {
                    id: `invite_${status.toLowerCase()}_${index}`,
                    status: status as MemberInviteStatus,
                    created_at: new Date(Date.now() - (index * DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(),
                    updated_at: status !== "Pending" 
                        ? new Date(Date.now() - (index * HOURS_IN_6 * MILLISECONDS_PER_HOUR)).toISOString()
                        : new Date(Date.now() - (index * DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create member invites for a specific team
     */
    createMemberInvitesForTeam(teamId: string, count = 5): MemberInvite[] {
        const team = teamResponseFactory.createMockData({ overrides: { id: teamId } });
        
        return Array.from({ length: count }, (_, index) =>
            this.createMockData({
                overrides: {
                    id: `invite_${teamId}_${index}`,
                    team,
                    user: userResponseFactory.createMockData({ 
                        overrides: { 
                            id: `user_invite_${index}`,
                            name: `Invited User ${index + 1}`,
                            handle: `invited_user_${index + 1}`,
                        },
                    }),
                    status: index === 0 ? "Pending" : (index % 2 === 0 ? "Accepted" : "Declined"),
                    message: index === 0 ? "Join our amazing team!" : null,
                    created_at: new Date(Date.now() - (index * HOURS_IN_6 * MILLISECONDS_PER_HOUR)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create member invites for a specific user
     */
    createMemberInvitesForUser(userId: string, count = 3): MemberInvite[] {
        const user = userResponseFactory.createMockData({ overrides: { id: userId } });
        
        return Array.from({ length: count }, (_, index) =>
            this.createMockData({
                overrides: {
                    id: `invite_user_${userId}_${index}`,
                    user,
                    team: teamResponseFactory.createMockData({ 
                        overrides: { 
                            id: `team_for_user_${index}`,
                            name: `Team ${index + 1}`,
                            handle: `team_${index + 1}`,
                        },
                    }),
                    status: index === 0 ? "Pending" : (index === 1 ? "Accepted" : "Declined"),
                    message: index === 0 ? "We'd love to have you on our team!" : null,
                    created_at: new Date(Date.now() - (index * DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create different member invite scenarios
     */
    createMemberInviteScenarios(): MemberInvite[] {
        const baseTime = Date.now();
        
        return [
            // Fresh pending invite
            this.createMockData({
                overrides: {
                    id: "fresh_pending_invite",
                    status: "Pending",
                    message: "Join our development team and help build amazing products!",
                    created_at: new Date(baseTime - (HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(), // 2 hours ago
                    team: teamResponseFactory.createMockData({
                        overrides: {
                            name: "Development Team",
                            handle: "dev_team",
                        },
                    }),
                },
            }),

            // Recently accepted invite
            this.createMockData({
                overrides: {
                    id: "recent_accepted_invite",
                    status: "Accepted",
                    message: "Welcome to our marketing team!",
                    created_at: new Date(baseTime - (DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(), // 1 day ago
                    updated_at: new Date(baseTime - (HOURS_IN_6 * MILLISECONDS_PER_HOUR)).toISOString(), // 6 hours ago
                    team: teamResponseFactory.createMockData({
                        overrides: {
                            name: "Marketing Team",
                            handle: "marketing_team",
                        },
                    }),
                },
            }),

            // Declined invite
            this.createMockData({
                overrides: {
                    id: "declined_invite",
                    status: "Declined",
                    message: "Join our research team for cutting-edge projects",
                    created_at: new Date(baseTime - (DAYS_IN_3 * MILLISECONDS_PER_DAY)).toISOString(), // 3 days ago
                    updated_at: new Date(baseTime - (DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(), // 1 day ago
                    team: teamResponseFactory.createMockData({
                        overrides: {
                            name: "Research Team",
                            handle: "research_team",
                        },
                    }),
                },
            }),

            // Invite without custom message
            this.createMockData({
                overrides: {
                    id: "no_message_invite",
                    status: "Pending",
                    message: null,
                    created_at: new Date(baseTime - (HOURS_IN_6 * MILLISECONDS_PER_HOUR)).toISOString(), // 6 hours ago
                    team: teamResponseFactory.createMockData({
                        overrides: {
                            name: "Design Team",
                            handle: "design_team",
                        },
                    }),
                },
            }),

            // Old pending invite (potentially expired)
            this.createMockData({
                overrides: {
                    id: "old_pending_invite",
                    status: "Pending",
                    message: "Join our operations team",
                    created_at: new Date(baseTime - (DAYS_IN_7 * MILLISECONDS_PER_DAY)).toISOString(), // 7 days ago (at expiry)
                    team: teamResponseFactory.createMockData({
                        overrides: {
                            name: "Operations Team",
                            handle: "ops_team",
                        },
                    }),
                },
            }),

            // Admin role invite
            this.createMockData({
                overrides: {
                    id: "admin_role_invite",
                    status: "Pending",
                    message: "We'd like you to join as an admin to help manage the team",
                    created_at: new Date(baseTime - (HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(),
                    team: teamResponseFactory.createMockData({
                        overrides: {
                            name: "Leadership Team",
                            handle: "leadership_team",
                        },
                    }),
                },
            }),
        ];
    }

    /**
     * Create invite already processed error response
     */
    createInviteAlreadyProcessedErrorResponse(status: MemberInviteStatus) {
        return this.createBusinessErrorResponse("already_processed", {
            resource: "member_invite",
            currentStatus: status,
            allowedActions: status === "Pending" ? ["accept", "decline"] : [],
            message: `This member invite has already been ${status.toLowerCase()}`,
        });
    }

    /**
     * Create invite expired error response
     */
    createInviteExpiredErrorResponse(inviteId: string, expiredAt: string) {
        return this.createBusinessErrorResponse("expired", {
            resource: "member_invite",
            inviteId,
            expiredAt,
            expiryDays: INVITE_EXPIRY_DAYS,
            message: `Member invite expired after ${INVITE_EXPIRY_DAYS} days`,
        });
    }

    /**
     * Create user already in team error response
     */
    createUserAlreadyInTeamErrorResponse(userId: string, teamId: string) {
        return this.createBusinessErrorResponse("already_member", {
            resource: "member_invite",
            userId,
            teamId,
            message: "User is already a member of this team",
        });
    }

    /**
     * Create self-invite error response
     */
    createSelfInviteErrorResponse() {
        return this.createValidationErrorResponse({
            userConnect: "You cannot invite yourself to a team",
        });
    }

    /**
     * Create team full error response
     */
    createTeamFullErrorResponse(teamId: string, maxMembers = 50) {
        return this.createBusinessErrorResponse("team_full", {
            resource: "member_invite",
            teamId,
            maxMembers,
            message: `Team has reached the maximum number of members (${maxMembers})`,
        });
    }

    /**
     * Create insufficient permissions error response
     */
    createInsufficientPermissionsErrorResponse(requiredRole: string) {
        return this.createPermissionErrorResponse(
            "invite members",
            [`team:${requiredRole}`],
            `You need ${requiredRole} permissions to invite members to this team`,
        );
    }
}

/**
 * Pre-configured member invite response scenarios
 */
export const memberInviteResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<MemberInviteCreateInput>) => {
        const factory = new MemberInviteResponseFactory();
        const defaultInput: MemberInviteCreateInput = {
            teamConnect: generatePK().toString(),
            userConnect: generatePK().toString(),
            message: "You've been invited to join our team!",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (memberInvite?: MemberInvite) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSuccessResponse(
            memberInvite || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: MemberInvite, updates?: Partial<MemberInviteUpdateInput>) => {
        const factory = new MemberInviteResponseFactory();
        const memberInvite = existing || factory.createMockData({ scenario: "complete" });
        const input: MemberInviteUpdateInput = {
            id: memberInvite.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(memberInvite, input),
        );
    },

    acceptSuccess: (inviteId?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: inviteId,
                    status: "Accepted",
                    updated_at: new Date().toISOString(),
                },
            }),
        );
    },

    declineSuccess: (inviteId?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: inviteId,
                    status: "Declined",
                    updated_at: new Date().toISOString(),
                },
            }),
        );
    },

    listSuccess: (memberInvites?: MemberInvite[]) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createPaginatedResponse(
            memberInvites || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: memberInvites?.length || DEFAULT_COUNT },
        );
    },

    allStatusesSuccess: () => {
        const factory = new MemberInviteResponseFactory();
        const invites = factory.createMemberInvitesForAllStatuses();
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    teamInvitesSuccess: (teamId?: string) => {
        const factory = new MemberInviteResponseFactory();
        const invites = factory.createMemberInvitesForTeam(teamId || generatePK().toString());
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    userInvitesSuccess: (userId?: string) => {
        const factory = new MemberInviteResponseFactory();
        const invites = factory.createMemberInvitesForUser(userId || generatePK().toString());
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    pendingInvitesSuccess: () => {
        const factory = new MemberInviteResponseFactory();
        const invites = factory.createMemberInviteScenarios().filter(invite => invite.status === "Pending");
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    scenariosSuccess: () => {
        const factory = new MemberInviteResponseFactory();
        const scenarios = factory.createMemberInviteScenarios();
        return factory.createPaginatedResponse(
            scenarios,
            { page: 1, totalCount: scenarios.length },
        );
    },

    adminInviteSuccess: () => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    message: "We'd like you to join as an admin to help manage the team",
                    team: teamResponseFactory.createMockData({
                        overrides: {
                            name: "Leadership Team",
                            handle: "leadership_team",
                        },
                    }),
                },
            }),
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new MemberInviteResponseFactory();
        return factory.createValidationErrorResponse({
            teamConnect: "Team ID is required",
            userConnect: "User ID is required",
        });
    },

    notFoundError: (inviteId?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createNotFoundErrorResponse(
            inviteId || "non-existent-invite",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["team:admin"],
        );
    },

    alreadyProcessedError: (status: MemberInviteStatus = "Accepted") => {
        const factory = new MemberInviteResponseFactory();
        return factory.createInviteAlreadyProcessedErrorResponse(status);
    },

    expiredInviteError: (inviteId?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createInviteExpiredErrorResponse(
            inviteId || generatePK().toString(),
            new Date(Date.now() - (INVITE_EXPIRY_DAYS * MILLISECONDS_PER_DAY)).toISOString(),
        );
    },

    userAlreadyInTeamError: (userId?: string, teamId?: string) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createUserAlreadyInTeamErrorResponse(
            userId || generatePK().toString(),
            teamId || generatePK().toString(),
        );
    },

    selfInviteError: () => {
        const factory = new MemberInviteResponseFactory();
        return factory.createSelfInviteErrorResponse();
    },

    teamFullError: (teamId?: string, maxMembers?: number) => {
        const factory = new MemberInviteResponseFactory();
        return factory.createTeamFullErrorResponse(teamId || generatePK().toString(), maxMembers);
    },

    insufficientPermissionsError: (requiredRole = "admin") => {
        const factory = new MemberInviteResponseFactory();
        return factory.createInsufficientPermissionsErrorResponse(requiredRole);
    },

    messageTooLongError: () => {
        const factory = new MemberInviteResponseFactory();
        return factory.createValidationErrorResponse({
            message: `Message must be ${MAX_MESSAGE_LENGTH} characters or less`,
        });
    },

    // MSW handlers
    handlers: {
        success: () => new MemberInviteResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new MemberInviteResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new MemberInviteResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const memberInviteResponseFactory = new MemberInviteResponseFactory();

