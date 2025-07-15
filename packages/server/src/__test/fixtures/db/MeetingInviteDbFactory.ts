import { ChatInviteStatus as InviteStatus } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface MeetingInviteRelationConfig extends RelationConfig {
    meeting: { meetingId: string };
    user: { userId: string };
}

/**
 * Enhanced database fixture factory for MeetingInvite model
 * Provides comprehensive testing capabilities for meeting invitation systems
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for different invite statuses
 * - Message customization
 * - Bulk invite operations
 * - Predefined test scenarios
 * - Constraint validation
 */
export class MeetingInviteDbFactory extends EnhancedDatabaseFactory<
    meeting_invite,
    Prisma.meeting_inviteCreateInput,
    Prisma.meeting_inviteInclude,
    Prisma.meeting_inviteUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("MeetingInvite", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.meeting_invite;
    }

    /**
     * Get complete test fixtures for MeetingInvite model
     */
    protected getFixtures(): DbTestFixtures<Prisma.meeting_inviteCreateInput, Prisma.meeting_inviteUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                status: InviteStatus.Pending,
                meeting: { connect: { id: this.generateId() } },
                user: { connect: { id: this.generateId() } },
            },
            complete: {
                id: this.generateId(),
                status: InviteStatus.Pending,
                message: "You're invited to our quarterly planning meeting. Your input would be valuable!",
                meeting: { connect: { id: this.generateId() } },
                user: { connect: { id: this.generateId() } },
            },
            invalid: {
                missingRequired: {
                    // Missing id, meeting, user
                    status: InviteStatus.Pending,
                    message: "Invalid invite",
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    status: "pending", // Should be InviteStatus enum
                    message: 123, // Should be string
                    meeting: "not-an-object", // Should be object
                    user: null, // Should be object
                },
                duplicateInvite: {
                    id: this.generateId(),
                    status: InviteStatus.Pending,
                    meeting: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } }, // Assumes this combo exists
                },
                invalidStatus: {
                    id: this.generateId(),
                    // @ts-expect-error - Intentionally invalid status
                    status: "InvalidStatus",
                    meeting: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
            },
            edgeCases: {
                maxLengthMessage: {
                    id: this.generateId(),
                    status: InviteStatus.Pending,
                    message: "a".repeat(4096), // Max length message
                    meeting: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                unicodeMessage: {
                    id: this.generateId(),
                    status: InviteStatus.Pending,
                    message: "You're invited! 🎉 Join us for an important meeting 📅",
                    meeting: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                acceptedInvite: {
                    id: this.generateId(),
                    status: InviteStatus.Accepted,
                    message: "Looking forward to your participation",
                    meeting: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                declinedInvite: {
                    id: this.generateId(),
                    status: InviteStatus.Declined,
                    message: "Unfortunately, I have a conflict at that time",
                    meeting: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                noMessageInvite: {
                    id: this.generateId(),
                    status: InviteStatus.Pending,
                    message: null,
                    meeting: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                lastMinuteInvite: {
                    id: this.generateId(),
                    status: InviteStatus.Pending,
                    message: "Urgent: Meeting starting in 30 minutes. Can you join?",
                    meeting: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
            },
            updates: {
                minimal: {
                    status: InviteStatus.Accepted,
                },
                complete: {
                    status: InviteStatus.Declined,
                    message: "Updated: I won't be able to attend due to a schedule conflict",
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.meeting_inviteCreateInput>): Prisma.meeting_inviteCreateInput {
        // Note: meeting and user connections must be provided via overrides or config
        return {
            id: this.generateId(),
            status: InviteStatus.Pending,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.meeting_inviteCreateInput>): Prisma.meeting_inviteCreateInput {
        return {
            id: this.generateId(),
            status: InviteStatus.Pending,
            message: "You're invited to join our team meeting. Your expertise would be valuable for this discussion.",
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            pendingInvite: {
                name: "pendingInvite",
                description: "Standard pending meeting invitation",
                config: {
                    overrides: {
                        status: InviteStatus.Pending,
                        message: "Please join us for our weekly team meeting",
                    },
                    meeting: { meetingId: this.bigIntToString(this.generateId()) },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            acceptedInvite: {
                name: "acceptedInvite",
                description: "Accepted meeting invitation",
                config: {
                    overrides: {
                        status: InviteStatus.Accepted,
                        message: "I'll be there! Looking forward to it",
                    },
                    meeting: { meetingId: this.bigIntToString(this.generateId()) },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            declinedInvite: {
                name: "declinedInvite",
                description: "Declined meeting invitation",
                config: {
                    overrides: {
                        status: InviteStatus.Declined,
                        message: "Sorry, I have another commitment at that time",
                    },
                    meeting: { meetingId: this.bigIntToString(this.generateId()) },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            executiveInvite: {
                name: "executiveInvite",
                description: "Executive meeting invitation",
                config: {
                    overrides: {
                        status: InviteStatus.Pending,
                        message: "Your presence is requested at the quarterly board meeting",
                    },
                    meeting: { meetingId: this.bigIntToString(this.generateId()) },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            clientInvite: {
                name: "clientInvite",
                description: "Client meeting invitation",
                config: {
                    overrides: {
                        status: InviteStatus.Pending,
                        message: "We'd love to show you our new product features. Are you available for a demo?",
                    },
                    meeting: { meetingId: this.bigIntToString(this.generateId()) },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            teamRetroInvite: {
                name: "teamRetroInvite",
                description: "Team retrospective invitation",
                config: {
                    overrides: {
                        status: InviteStatus.Pending,
                        message: "Time for our sprint retrospective. Please prepare your thoughts on what went well and what could improve.",
                    },
                    meeting: { meetingId: this.bigIntToString(this.generateId()) },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.meeting_inviteInclude {
        return {
            meeting: {
                select: {
                    id: true,
                    publicId: true,
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: true,
                    translations: {
                        select: {
                            language: true,
                            name: true,
                            description: true,
                            link: true,
                        },
                    },
                    team: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                        },
                    },
                },
            },
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.meeting_inviteCreateInput,
        config: MeetingInviteRelationConfig,
        tx: any,
    ): Promise<Prisma.meeting_inviteCreateInput> {
        const data = { ...baseData };

        // Handle meeting connection (required)
        if (config.meeting) {
            data.meeting = {
                connect: { id: config.meeting.meetingId },
            };
        } else {
            throw new Error("MeetingInvite requires a meeting connection");
        }

        // Handle user connection (required)
        if (config.user) {
            data.user = {
                connect: { id: config.user.userId },
            };
        } else {
            throw new Error("MeetingInvite requires a user connection");
        }

        return data;
    }

    /**
     * Create a pending invite
     */
    async createPendingInvite(
        meetingId: string,
        userId: string,
        message?: string,
    ): Promise<MeetingInvite> {
        return await this.createWithRelations({
            overrides: {
                status: InviteStatus.Pending,
                message: message || "You're invited to join our meeting",
            },
            meeting: { meetingId },
            user: { userId },
        });
    }

    /**
     * Create bulk invites for multiple users
     */
    async createBulkInvites(
        meetingId: string,
        userIds: string[],
        message?: string,
    ): Promise<meeting_invite[]> {
        const invites = await Promise.all(
            userIds.map(userId => 
                this.createPendingInvite(meetingId, userId, message),
            ),
        );
        return invites;
    }

    /**
     * Accept an invite
     */
    async acceptInvite(inviteId: string): Promise<MeetingInvite> {
        const updated = await this.prisma.meeting_invite.update({
            where: { id: inviteId },
            data: { status: InviteStatus.Accepted },
            include: this.getDefaultInclude(),
        });

        // Also add as attendee
        const invite = await this.prisma.meeting_invite.findUnique({
            where: { id: inviteId },
        });
        
        if (invite) {
            await this.prisma.meeting_attendee.create({
                data: {
                    id: this.generateId(),
                    meetingId: invite.meetingId,
                    userId: invite.userId,
                },
            });
        }

        return updated;
    }

    /**
     * Decline an invite
     */
    async declineInvite(inviteId: string, reason?: string): Promise<MeetingInvite> {
        const updated = await this.prisma.meeting_invite.update({
            where: { id: inviteId },
            data: { 
                status: InviteStatus.Declined,
                message: reason || "Unable to attend",
            },
            include: this.getDefaultInclude(),
        });
        return updated;
    }

    protected async checkModelConstraints(record: MeetingInvite): Promise<string[]> {
        const violations: string[] = [];
        
        // Check for duplicate invite
        const duplicate = await this.prisma.meeting_invite.findFirst({
            where: {
                meetingId: record.meetingId,
                userId: record.userId,
                id: { not: record.id },
            },
        });
        
        if (duplicate) {
            violations.push("User already has an invite to this meeting");
        }

        // Check if user is already an attendee
        const existingAttendee = await this.prisma.meeting_attendee.findFirst({
            where: {
                meetingId: record.meetingId,
                userId: record.userId,
            },
        });
        
        if (existingAttendee && record.status === InviteStatus.Pending) {
            violations.push("User is already an attendee of this meeting");
        }

        // Check message length
        if (record.message && record.message.length > 4096) {
            violations.push("Message exceeds maximum length of 4096 characters");
        }

        // Check valid status
        if (!Object.values(InviteStatus).includes(record.status as InviteStatus)) {
            violations.push("Invalid invite status");
        }

        // Check meeting exists
        const meeting = await this.prisma.meeting.findUnique({
            where: { id: record.meetingId },
        });
        
        if (!meeting) {
            violations.push("Meeting does not exist");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // MeetingInvite has no dependent records
        };
    }

    protected async deleteRelatedRecords(
        record: meeting_invite,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // MeetingInvite has no dependent records to delete
    }

    /**
     * Get invites by status
     */
    async getInvitesByStatus(
        status: InviteStatus,
        meetingId?: string,
    ): Promise<meeting_invite[]> {
        const where: Prisma.meeting_inviteWhereInput = { status };
        if (meetingId) {
            where.meetingId = meetingId;
        }

        return await this.prisma.meeting_invite.findMany({
            where,
            include: this.getDefaultInclude(),
            orderBy: { createdAt: "desc" },
        });
    }

    /**
     * Get user's pending meeting invites
     */
    async getUserPendingInvites(userId: string): Promise<meeting_invite[]> {
        return await this.prisma.meeting_invite.findMany({
            where: {
                userId,
                status: InviteStatus.Pending,
            },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: "desc" },
        });
    }

    /**
     * Send reminder for pending invites
     */
    async getReminderCandidates(
        hoursBeforeMeeting = 24,
    ): Promise<meeting_invite[]> {
        const cutoffTime = new Date(Date.now() + hoursBeforeMeeting * 60 * 60 * 1000);
        
        return await this.prisma.meeting_invite.findMany({
            where: {
                status: InviteStatus.Pending,
                meeting: {
                    schedule: {
                        recurrences: {
                            some: {
                                nextRunAt: {
                                    lte: cutoffTime,
                                },
                            },
                        },
                    },
                },
            },
            include: this.getDefaultInclude(),
        });
    }

    /**
     * Create test invites with different statuses
     */
    async createTestInviteSet(
        meetingId: string,
        userIds: string[],
    ): Promise<{
        pending: meeting_invite[];
        accepted: meeting_invite[];
        declined: meeting_invite[];
    }> {
        if (userIds.length < 3) {
            throw new Error("Need at least 3 user IDs for test set");
        }

        const pending = await this.createPendingInvite(
            meetingId,
            userIds[0],
            "Pending invitation - awaiting response",
        );

        const acceptedInvite = await this.createPendingInvite(
            meetingId,
            userIds[1],
            "This will be accepted",
        );
        const accepted = await this.acceptInvite(acceptedInvite.id);

        const declinedInvite = await this.createPendingInvite(
            meetingId,
            userIds[2],
            "This will be declined",
        );
        const declined = await this.declineInvite(
            declinedInvite.id,
            "Schedule conflict",
        );

        return {
            pending: [pending],
            accepted: [accepted],
            declined: [declined],
        };
    }

    /**
     * Create invites for different meeting types
     */
    async createMeetingTypeInvites(userId: string): Promise<{
        teamMeeting: meeting_invite;
        clientMeeting: meeting_invite;
        webinar: meeting_invite;
        oneOnOne: meeting_invite;
    }> {
        const teamMeeting = await this.createPendingInvite(
            "meeting-team",
            userId,
            "Weekly team sync - please review the agenda beforehand",
        );

        const clientMeeting = await this.createPendingInvite(
            "meeting-client",
            userId,
            "Client presentation - your technical expertise would be valuable",
        );

        const webinar = await this.createPendingInvite(
            "meeting-webinar",
            userId,
            "Join our product launch webinar! Feel free to invite colleagues",
        );

        const oneOnOne = await this.createPendingInvite(
            "meeting-1on1",
            userId,
            "Monthly 1:1 check-in - let's discuss your goals and progress",
        );

        return {
            teamMeeting,
            clientMeeting,
            webinar,
            oneOnOne,
        };
    }
}

// Export factory creator function
export const createMeetingInviteDbFactory = (prisma: PrismaClient) => 
    new MeetingInviteDbFactory(prisma);

// Export the class for type usage
export { MeetingInviteDbFactory as MeetingInviteDbFactoryClass };
