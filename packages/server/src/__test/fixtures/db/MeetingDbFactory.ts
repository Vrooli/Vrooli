// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type meeting, type Prisma, type PrismaClient } from "@prisma/client";
import { generatePublicId } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

type MeetingWithRelations = meeting & {
    invites?: Array<{ id: bigint }>;
    attendees?: Array<{ id: bigint }>;
    subscriptions?: Array<{ id: bigint }>;
    translations?: Array<{ id: bigint }>;
};

interface MeetingRelationConfig extends RelationConfig {
    team: { teamId: string | bigint };
    schedule?: { scheduleId: string | bigint };
    attendees?: Array<{ userId: string | bigint }>;
    invites?: Array<{ userId: string | bigint; message?: string }>;
    translations?: Array<{
        language: string;
        name?: string;
        description?: string;
        link?: string;
    }>;
}

/**
 * Enhanced database fixture factory for Meeting model
 * Provides comprehensive testing capabilities for meeting systems
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for team meetings
 * - Schedule integration
 * - Attendee and invite management
 * - Translation support
 * - Public/private meeting options
 * - Predefined test scenarios
 */
export class MeetingDbFactory extends EnhancedDatabaseFactory<
    meeting,
    Prisma.meetingCreateInput,
    Prisma.meetingInclude,
    Prisma.meetingUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("Meeting", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.meeting;
    }

    /**
     * Get complete test fixtures for Meeting model
     */
    protected getFixtures(): DbTestFixtures<Prisma.meetingCreateInput, Prisma.meetingUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                publicId: generatePublicId(),
                openToAnyoneWithInvite: false,
                showOnTeamProfile: false,
                team: { connect: { id: this.generateId() } },
            },
            complete: {
                id: this.generateId(),
                publicId: generatePublicId(),
                openToAnyoneWithInvite: true,
                showOnTeamProfile: true,
                team: { connect: { id: this.generateId() } },
                schedule: { connect: { id: this.generateId() } },
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            name: "Weekly Team Standup",
                            description: "Weekly synchronization meeting for the entire team",
                            link: "https://meet.example.com/team-standup",
                        },
                        {
                            id: this.generateId(),
                            language: "es",
                            name: "Reunión Semanal del Equipo",
                            description: "Reunión de sincronización semanal para todo el equipo",
                            link: "https://meet.example.com/team-standup",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, team
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: false,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    publicId: 123, // Should be string
                    openToAnyoneWithInvite: "yes", // Should be boolean
                    showOnTeamProfile: 1, // Should be boolean
                    team: null, // Should be object
                },
                conflictingSettings: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: false, // Odd to be open but not shown
                    team: { connect: { id: this.generateId() } },
                },
                missingTeam: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: false,
                    // Missing required team connection
                },
            },
            edgeCases: {
                publicMeeting: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: true,
                    team: { connect: { id: this.generateId() } },
                },
                privateMeeting: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: false,
                    team: { connect: { id: this.generateId() } },
                },
                recurringMeeting: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                    team: { connect: { id: this.generateId() } },
                    schedule: { connect: { id: this.generateId() } },
                },
                unicodeMeeting: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: true,
                    team: { connect: { id: this.generateId() } },
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            name: "International Meeting 🌍 会议",
                            description: "Global team sync with participants from around the world",
                            link: "https://meet.example.com/global",
                        }],
                    },
                },
                maxAttendeesMeeting: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: true,
                    team: { connect: { id: this.generateId() } },
                    // Would add many attendees after creation
                },
            },
            updates: {
                minimal: {
                    openToAnyoneWithInvite: true,
                },
                complete: {
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                    translations: {
                        update: [{
                            where: { id: this.generateId() },
                            data: {
                                name: "Updated Meeting Name",
                                description: "Updated meeting description",
                                link: "https://meet.example.com/updated",
                            },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.meetingCreateInput>): Prisma.meetingCreateInput {
        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            openToAnyoneWithInvite: false,
            showOnTeamProfile: false,
            team: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.meetingCreateInput>): Prisma.meetingCreateInput {
        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            openToAnyoneWithInvite: true,
            showOnTeamProfile: true,
            team: { connect: { id: this.generateId() } },
            translations: {
                create: [
                    {
                        id: this.generateId(),
                        language: "en",
                        name: "Team Meeting",
                        description: "Regular team synchronization meeting",
                        link: "https://meet.example.com/team",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            teamStandup: {
                name: "teamStandup",
                description: "Daily team standup meeting",
                config: {
                    overrides: {
                        openToAnyoneWithInvite: false,
                        showOnTeamProfile: true,
                    },
                    team: { teamId: this.generateId() },
                    translations: [{
                        language: "en",
                        name: "Daily Standup",
                        description: "15-minute daily synchronization",
                        link: "https://meet.example.com/standup",
                    }],
                    attendees: [
                        { userId: this.generateId() },
                        { userId: this.generateId() },
                        { userId: this.generateId() },
                    ],
                },
            },
            publicWebinar: {
                name: "publicWebinar",
                description: "Public webinar open to all",
                config: {
                    overrides: {
                        openToAnyoneWithInvite: true,
                        showOnTeamProfile: true,
                    },
                    team: { teamId: this.generateId() },
                    translations: [{
                        language: "en",
                        name: "Product Launch Webinar",
                        description: "Join us for the launch of our new product",
                        link: "https://meet.example.com/webinar",
                    }],
                },
            },
            privateStrategy: {
                name: "privateStrategy",
                description: "Private strategy meeting",
                config: {
                    overrides: {
                        openToAnyoneWithInvite: false,
                        showOnTeamProfile: false,
                    },
                    team: { teamId: this.generateId() },
                    translations: [{
                        language: "en",
                        name: "Q4 Strategy Session",
                        description: "Executive team strategy planning",
                    }],
                    attendees: [
                        { userId: this.generateId() },
                        { userId: this.generateId() },
                        { userId: this.generateId() },
                    ],
                },
            },
            recurringTeamMeeting: {
                name: "recurringTeamMeeting",
                description: "Weekly recurring team meeting",
                config: {
                    overrides: {
                        openToAnyoneWithInvite: false,
                        showOnTeamProfile: true,
                    },
                    team: { teamId: this.generateId() },
                    schedule: { scheduleId: "schedule-weekly" },
                    translations: [{
                        language: "en",
                        name: "Weekly Team Sync",
                        description: "Weekly team progress and planning meeting",
                        link: "https://meet.example.com/weekly",
                    }],
                },
            },
            clientPresentation: {
                name: "clientPresentation",
                description: "Client presentation meeting",
                config: {
                    overrides: {
                        openToAnyoneWithInvite: true,
                        showOnTeamProfile: false,
                    },
                    team: { teamId: this.generateId() },
                    translations: [{
                        language: "en",
                        name: "Product Demo",
                        description: "Product demonstration for prospective client",
                        link: "https://meet.example.com/demo",
                    }],
                    invites: [
                        { userId: this.generateId(), message: "Looking forward to showing you our product!" },
                        { userId: this.generateId(), message: "Please join us for the demo" },
                    ],
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.meetingInclude {
        return {
            team: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                },
            },
            schedule: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            attendees: {
                select: {
                    id: true,
                    user: {
                        select: {
                            id: true,
                            publicId: true,
                            handle: true,
                        },
                    },
                },
            },
            invites: {
                select: {
                    id: true,
                    status: true,
                    message: true,
                    user: {
                        select: {
                            id: true,
                            publicId: true,
                            handle: true,
                        },
                    },
                },
            },
            translations: true,
            _count: {
                select: {
                    attendees: true,
                    invites: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.meetingCreateInput,
        config: MeetingRelationConfig,
        _tx: PrismaClient,
    ): Promise<Prisma.meetingCreateInput> {
        const data = { ...baseData };

        // Handle team connection (required)
        if (config.team) {
            data.team = {
                connect: { id: BigInt(config.team.teamId) },
            };
        } else {
            throw new Error("Meeting requires a team connection");
        }

        // Handle schedule connection
        if (config.schedule) {
            data.schedule = {
                connect: { id: BigInt(config.schedule.scheduleId) },
            };
        }

        // Handle attendees
        if (config.attendees && Array.isArray(config.attendees)) {
            data.attendees = {
                create: config.attendees.map(attendee => ({
                    id: this.generateId(),
                    userId: BigInt(attendee.userId),
                })),
            };
        }

        // Handle invites
        if (config.invites && Array.isArray(config.invites)) {
            data.invites = {
                create: config.invites.map(invite => ({
                    id: this.generateId(),
                    userId: BigInt(invite.userId),
                    status: "Pending",
                    message: invite.message || "You're invited to join our meeting",
                })),
            };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: this.generateId(),
                    ...trans,
                })),
            };
        }

        return data;
    }

    /**
     * Create a team meeting
     */
    async createTeamMeeting(
        teamId: string | bigint,
        name: string,
        options?: {
            description?: string;
            link?: string;
            isPublic?: boolean;
            showOnProfile?: boolean;
        },
    ): Promise<meeting> {
        return await this.createWithRelations({
            overrides: {
                openToAnyoneWithInvite: options?.isPublic ?? false,
                showOnTeamProfile: options?.showOnProfile ?? true,
            },
            team: { teamId },
            translations: [{
                language: "en",
                name,
                description: options?.description,
                link: options?.link,
            }],
        });
    }

    /**
     * Create a recurring meeting
     */
    async createRecurringMeeting(
        teamId: string | bigint,
        scheduleId: string | bigint,
        name: string,
    ): Promise<meeting> {
        return await this.createWithRelations({
            overrides: {
                showOnTeamProfile: true,
            },
            team: { teamId },
            schedule: { scheduleId },
            translations: [{
                language: "en",
                name,
                description: "Recurring meeting",
            }],
        });
    }

    /**
     * Add attendees to a meeting
     */
    async addAttendees(meetingId: string | bigint, userIds: (string | bigint)[]): Promise<void> {
        await this.prisma.meeting_attendees.createMany({
            data: userIds.map(userId => ({
                id: this.generateId(),
                meetingId: BigInt(meetingId),
                userId: BigInt(userId),
            })),
            skipDuplicates: true,
        });
    }

    /**
     * Remove attendee from meeting
     */
    async removeAttendee(meetingId: string | bigint, userId: string | bigint): Promise<void> {
        await this.prisma.meeting_attendees.deleteMany({
            where: {
                meetingId: BigInt(meetingId),
                userId: BigInt(userId),
            },
        });
    }

    /**
     * Create meeting invites
     */
    async createInvites(
        meetingId: string | bigint,
        invites: Array<{ userId: string | bigint; message?: string }>,
    ): Promise<void> {
        await this.prisma.meeting_invite.createMany({
            data: invites.map(invite => ({
                id: this.generateId(),
                meetingId: BigInt(meetingId),
                userId: BigInt(invite.userId),
                status: "Pending",
                message: invite.message || "You're invited to join our meeting",
            })),
            skipDuplicates: true,
        });
    }

    protected async checkModelConstraints(record: meeting): Promise<string[]> {
        const violations: string[] = [];

        // Check team exists
        const team = await this.prisma.team.findUnique({
            where: { id: record.teamId },
        });

        if (!team) {
            violations.push("Team does not exist");
        }

        // Check schedule exists if specified
        if (record.scheduleId) {
            const schedule = await this.prisma.schedule.findUnique({
                where: { id: record.scheduleId },
            });

            if (!schedule) {
                violations.push("Schedule does not exist");
            }
        }

        // Check logical constraints
        if (!record.showOnTeamProfile && record.openToAnyoneWithInvite) {
            violations.push("Meeting open to anyone should be shown on team profile");
        }

        // Check attendee limit (hypothetical)
        const attendeeCount = await this.prisma.meeting_attendees.count({
            where: { meetingId: record.id },
        });

        const MAX_ATTENDEES = 1000;
        if (attendeeCount > MAX_ATTENDEES) {
            violations.push(`Meeting exceeds maximum attendee limit of ${MAX_ATTENDEES}`);
        }

        return violations;
    }

    protected getCascadeInclude(): object {
        return {
            attendees: true,
            invites: true,
            translations: true,
            subscriptions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: MeetingWithRelations,
        remainingDepth: number,
        tx: PrismaClient,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        function shouldDelete(relation: string): boolean {
            return !includeOnly || includeOnly.includes(relation);
        }

        // Delete in order of dependencies

        // Delete invites
        if (shouldDelete("invites") && record.invites?.length) {
            await tx.meeting_invite.deleteMany({
                where: { meetingId: record.id },
            });
        }

        // Delete attendees
        if (shouldDelete("attendees") && record.attendees?.length) {
            await tx.meeting_attendees.deleteMany({
                where: { meetingId: record.id },
            });
        }

        // Delete subscriptions
        if (shouldDelete("subscriptions") && record.subscriptions?.length) {
            await tx.notification_subscription.deleteMany({
                where: { meetingId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete("translations") && record.translations?.length) {
            await tx.meeting_translation.deleteMany({
                where: { meetingId: record.id },
            });
        }
    }

    /**
     * Get public meetings for a team
     */
    async getPublicTeamMeetings(teamId: string | bigint): Promise<meeting[]> {
        return await this.prisma.meeting.findMany({
            where: {
                teamId: BigInt(teamId),
                showOnTeamProfile: true,
            },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: "desc" },
        });
    }

    /**
     * Get meetings with open invites
     */
    async getOpenMeetings(): Promise<meeting[]> {
        return await this.prisma.meeting.findMany({
            where: {
                openToAnyoneWithInvite: true,
            },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: "desc" },
        });
    }

    /**
     * Create different meeting types for testing
     */
    async createTestingScenarios(teamId: string | bigint): Promise<{
        publicMeeting: meeting;
        privateMeeting: meeting;
        recurringMeeting: meeting;
        webinar: meeting;
    }> {
        const publicMeeting = await this.createTeamMeeting(
            teamId,
            "Public Team Meeting",
            {
                description: "Open to all team members",
                isPublic: true,
                showOnProfile: true,
            },
        );

        const privateMeeting = await this.createTeamMeeting(
            teamId,
            "Private Strategy Session",
            {
                description: "Executive team only",
                isPublic: false,
                showOnProfile: false,
            },
        );

        const recurringMeeting = await this.createRecurringMeeting(
            teamId,
            this.generateId(),
            "Weekly Team Sync",
        );

        const webinar = await this.createTeamMeeting(
            teamId,
            "Product Launch Webinar",
            {
                description: "Join us for our exciting product launch!",
                link: "https://webinar.example.com/launch",
                isPublic: true,
                showOnProfile: true,
            },
        );

        return {
            publicMeeting,
            privateMeeting,
            recurringMeeting,
            webinar,
        };
    }
}

// Export factory creator function
export function createMeetingDbFactory(prisma: PrismaClient): MeetingDbFactory {
    return new MeetingDbFactory(prisma);
}

// Export the class for type usage
export { MeetingDbFactory as MeetingDbFactoryClass };
