/* c8 ignore start */
/**
 * MeetingInvite API Response Fixtures
 * 
 * Comprehensive fixtures for meeting invitation endpoints including
 * invite creation, RSVP management, and meeting coordination.
 */

import type {
    Meeting,
    MeetingInvite,
    MeetingInviteCreateInput,
    MeetingInviteStatus,
    MeetingInviteUpdateInput,
    User,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const INVITE_MESSAGE_MAX_LENGTH = 1000;

/**
 * MeetingInvite API response factory
 */
export class MeetingInviteResponseFactory extends BaseAPIResponseFactory<
    MeetingInvite,
    MeetingInviteCreateInput,
    MeetingInviteUpdateInput
> {
    protected readonly entityName = "meeting invite";

    /**
     * Create mock meeting invite data
     */
    createMockData(options?: MockDataOptions): MeetingInvite {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const inviteId = options?.overrides?.id || generatePK().toString();

        const baseMeetingInvite: MeetingInvite = {
            __typename: "MeetingInvite",
            id: inviteId,
            createdAt: now,
            updatedAt: now,
            meeting: this.createMockMeeting(),
            message: "You're invited to join our meeting. Please RSVP at your earliest convenience.",
            status: "Pending",
            user: userResponseFactory.createMockData(),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseMeetingInvite,
                message: "Important quarterly planning session - your input is valuable and attendance is highly encouraged. We'll be discussing upcoming projects, resource allocation, and strategic initiatives for the next quarter.",
                status: "Accepted",
                meeting: this.createMockMeeting({
                    translations: [{
                        __typename: "MeetingTranslation",
                        id: generatePK().toString(),
                        language: "en",
                        name: "Quarterly Planning Session",
                        description: "Strategic planning meeting for Q4 objectives and initiatives",
                    }],
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                }),
                user: userResponseFactory.createMockData({ scenario: "complete" }),
                you: {
                    canDelete: true,
                    canUpdate: true,
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseMeetingInvite,
            ...options?.overrides,
        };
    }

    /**
     * Create meeting invite from input
     */
    createFromInput(input: MeetingInviteCreateInput): MeetingInvite {
        const now = new Date().toISOString();
        const inviteId = generatePK().toString();

        return {
            __typename: "MeetingInvite",
            id: inviteId,
            createdAt: now,
            updatedAt: now,
            meeting: {
                __typename: "Meeting",
                id: input.meetingConnect,
            } as Meeting,
            message: input.message || "You're invited to join this meeting.",
            status: "Pending",
            user: {
                __typename: "User",
                id: input.userConnect,
            } as User,
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };
    }

    /**
     * Update meeting invite from input
     */
    updateFromInput(existing: MeetingInvite, input: MeetingInviteUpdateInput): MeetingInvite {
        const updates: Partial<MeetingInvite> = {
            updatedAt: new Date().toISOString(),
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
    async validateCreateInput(input: MeetingInviteCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.meetingConnect) {
            errors.meetingConnect = "Meeting ID is required";
        }

        if (!input.userConnect) {
            errors.userConnect = "User ID is required";
        }

        if (input.message && input.message.length > INVITE_MESSAGE_MAX_LENGTH) {
            errors.message = "Message must be 1000 characters or less";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: MeetingInviteUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.message !== undefined && input.message.length > INVITE_MESSAGE_MAX_LENGTH) {
            errors.message = "Message must be 1000 characters or less";
        }

        if (input.status !== undefined) {
            const validStatuses: MeetingInviteStatus[] = ["Pending", "Accepted", "Declined"];
            if (!validStatuses.includes(input.status)) {
                errors.status = "Invalid status. Must be Pending, Accepted, or Declined";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create mock meeting data (simplified for invite context)
     */
    private createMockMeeting(overrides?: Partial<Meeting>): Meeting {
        const now = new Date().toISOString();
        const meetingId = generatePK().toString();

        return {
            __typename: "Meeting",
            id: meetingId,
            createdAt: now,
            updatedAt: now,
            attendees: [],
            attendeesCount: 0,
            invites: [],
            invitesCount: 1,
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
            schedule: {
                __typename: "Schedule",
                id: generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                startTime: new Date(Date.now().toISOString() + 86400000).toISOString(), // Tomorrow
                endTime: new Date(Date.now().toISOString() + 90000000).toISOString(), // Tomorrow + 1 hour
                timezone: "UTC",
                you: {
                    canDelete: false,
                    canUpdate: false,
                },
            },
            team: {
                __typename: "Team",
                id: generatePK().toString(),
                createdAt: now,
                updatedAt: now,
                handle: "example-team",
                name: "Example Team",
                isOpenToNewMembers: true,
                isPrivate: false,
                you: {
                    canAddMembers: false,
                    canDelete: false,
                    canBookmark: true,
                    canReport: false,
                    canUpdate: false,
                    canRead: true,
                    isBookmarked: false,
                    isViewed: false,
                    role: "Member",
                },
            },
            translations: [{
                __typename: "MeetingTranslation",
                id: generatePK().toString(),
                language: "en",
                name: "Team Meeting",
                description: "Regular team sync meeting",
            }],
            translationsCount: 1,
            you: {
                canDelete: false,
                canInvite: true,
                canUpdate: false,
            },
            ...overrides,
        } as Meeting;
    }

    /**
     * Create meeting invites for different statuses
     */
    createInvitesForAllStatuses(): MeetingInvite[] {
        const statuses: MeetingInviteStatus[] = ["Pending", "Accepted", "Declined"];
        return statuses.map((status, index) =>
            this.createMockData({
                overrides: {
                    id: `invite_${status.toLowerCase()}_${index}`,
                    status,
                    user: userResponseFactory.createMockData({
                        overrides: {
                            handle: `user_${status.toLowerCase()}`,
                            name: `${status} User`,
                        },
                    }),
                },
            }),
        );
    }

    /**
     * Create urgent meeting invite
     */
    createUrgentInvite(): MeetingInvite {
        return this.createMockData({
            scenario: "complete",
            overrides: {
                message: "URGENT: Emergency meeting required - immediate attention needed for critical issue resolution.",
                meeting: this.createMockMeeting({
                    translations: [{
                        __typename: "MeetingTranslation",
                        id: generatePK().toString(),
                        language: "en",
                        name: "Emergency Response Meeting",
                        description: "Urgent meeting to address critical system issues",
                    }],
                    schedule: {
                        __typename: "Schedule",
                        id: generatePK().toString(),
                        createdAt: new Date().toISOString(),
                        updatedAt: new Date().toISOString(),
                        startTime: new Date(Date.now().toISOString() + 3600000).toISOString(), // In 1 hour
                        endTime: new Date(Date.now().toISOString() + 5400000).toISOString(), // In 1.5 hours
                        timezone: "UTC",
                        you: {
                            canDelete: false,
                            canUpdate: false,
                        },
                    },
                }),
                status: "Pending",
            },
        });
    }

    /**
     * Create RSVP conflict error response
     */
    createRSVPConflictErrorResponse(currentStatus: MeetingInviteStatus, requestedStatus: MeetingInviteStatus) {
        return this.createBusinessErrorResponse("conflict", {
            field: "status",
            currentValue: currentStatus,
            requestedValue: requestedStatus,
            message: `Cannot change RSVP status from ${currentStatus} to ${requestedStatus}`,
            validTransitions: ["Pending -> Accepted", "Pending -> Declined", "Accepted -> Declined", "Declined -> Accepted"],
        });
    }

    /**
     * Create meeting capacity error response
     */
    createMeetingCapacityErrorResponse(currentCount: number, maxCapacity: number) {
        return this.createBusinessErrorResponse("limit", {
            resource: "meeting attendees",
            current: currentCount,
            limit: maxCapacity,
            message: "Meeting has reached maximum capacity",
            waitingListAvailable: true,
        });
    }

    /**
     * Create duplicate invite error response
     */
    createDuplicateInviteErrorResponse(userId: string, meetingId: string) {
        return this.createBusinessErrorResponse("duplicate", {
            resource: "meeting invite",
            userId,
            meetingId,
            message: "User has already been invited to this meeting",
        });
    }

    /**
     * Create past meeting error response
     */
    createPastMeetingErrorResponse(meetingId: string) {
        return this.createBusinessErrorResponse("state", {
            resource: "meeting",
            meetingId,
            reason: "Meeting has already occurred",
            message: "Cannot send invites for past meetings",
        });
    }
}

/**
 * Pre-configured meeting invite response scenarios
 */
export const meetingInviteResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<MeetingInviteCreateInput>) => {
        const factory = new MeetingInviteResponseFactory();
        const defaultInput: MeetingInviteCreateInput = {
            meetingConnect: generatePK().toString(),
            userConnect: generatePK().toString(),
            message: "You're invited to join our meeting.",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (meetingInvite?: MeetingInvite) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createSuccessResponse(
            meetingInvite || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: MeetingInvite, updates?: Partial<MeetingInviteUpdateInput>) => {
        const factory = new MeetingInviteResponseFactory();
        const invite = existing || factory.createMockData({ scenario: "complete" });
        const input: MeetingInviteUpdateInput = {
            id: invite.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(invite, input),
        );
    },

    rsvpSuccess: (status: MeetingInviteStatus) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: { status },
            }),
        );
    },

    urgentInviteSuccess: () => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createSuccessResponse(
            factory.createUrgentInvite(),
        );
    },

    listSuccess: (meetingInvites?: MeetingInvite[]) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createPaginatedResponse(
            meetingInvites || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: meetingInvites?.length || DEFAULT_COUNT },
        );
    },

    statusFilteredSuccess: (status: MeetingInviteStatus) => {
        const factory = new MeetingInviteResponseFactory();
        const invites = factory.createInvitesForAllStatuses()
            .filter(invite => invite.status === status);
        return factory.createPaginatedResponse(
            invites,
            { page: 1, totalCount: invites.length },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createValidationErrorResponse({
            meetingConnect: "Meeting ID is required",
            userConnect: "User ID is required",
            message: "Message must be 1000 characters or less",
        });
    },

    notFoundError: (meetingInviteId?: string) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createNotFoundErrorResponse(
            meetingInviteId || "non-existent-meeting-invite",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["meeting:invite"],
        );
    },

    rsvpConflictError: (currentStatus?: MeetingInviteStatus, requestedStatus?: MeetingInviteStatus) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createRSVPConflictErrorResponse(
            currentStatus || "Accepted",
            requestedStatus || "Pending",
        );
    },

    meetingCapacityError: (currentCount = 50, maxCapacity = 50) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createMeetingCapacityErrorResponse(currentCount, maxCapacity);
    },

    duplicateInviteError: (userId?: string, meetingId?: string) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createDuplicateInviteErrorResponse(
            userId || generatePK().toString(),
            meetingId || generatePK().toString(),
        );
    },

    pastMeetingError: (meetingId?: string) => {
        const factory = new MeetingInviteResponseFactory();
        return factory.createPastMeetingErrorResponse(meetingId || generatePK().toString());
    },

    // MSW handlers
    handlers: {
        success: () => new MeetingInviteResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new MeetingInviteResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new MeetingInviteResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const meetingInviteResponseFactory = new MeetingInviteResponseFactory();
