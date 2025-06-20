import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface MeetingRelationConfig extends RelationConfig {
    team: { teamId: string };
    schedule?: { scheduleId: string };
    attendees?: Array<{ userId: string }>;
    invites?: Array<{ userId: string; message?: string }>;
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
    Prisma.Meeting,
    Prisma.MeetingCreateInput,
    Prisma.MeetingInclude,
    Prisma.MeetingUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Meeting', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.meeting;
    }

    /**
     * Get complete test fixtures for Meeting model
     */
    protected getFixtures(): DbTestFixtures<Prisma.MeetingCreateInput, Prisma.MeetingUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                openToAnyoneWithInvite: false,
                showOnTeamProfile: false,
                team: { connect: { id: "team-123" } },
            },
            complete: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                openToAnyoneWithInvite: true,
                showOnTeamProfile: true,
                team: { connect: { id: "team-456" } },
                schedule: { connect: { id: "schedule-456" } },
                translations: {
                    create: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Weekly Team Standup",
                            description: "Weekly synchronization meeting for the entire team",
                            link: "https://meet.example.com/team-standup",
                        },
                        {
                            id: generatePK().toString(),
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
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: false, // Odd to be open but not shown
                    team: { connect: { id: "team-conflict" } },
                },
                missingTeam: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: false,
                    // Missing required team connection
                },
            },
            edgeCases: {
                publicMeeting: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: true,
                    team: { connect: { id: "team-public" } },
                },
                privateMeeting: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: false,
                    team: { connect: { id: "team-private" } },
                },
                recurringMeeting: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                    team: { connect: { id: "team-recurring" } },
                    schedule: { connect: { id: "schedule-recurring" } },
                },
                unicodeMeeting: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: true,
                    team: { connect: { id: "team-unicode" } },
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            name: "International Meeting 🌍 会议",
                            description: "Global team sync with participants from around the world",
                            link: "https://meet.example.com/global",
                        }],
                    },
                },
                maxAttendeesMeeting: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: true,
                    showOnTeamProfile: true,
                    team: { connect: { id: "team-large" } },
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
                            where: { id: "translation_id" },
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

    protected generateMinimalData(overrides?: Partial<Prisma.MeetingCreateInput>): Prisma.MeetingCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            openToAnyoneWithInvite: false,
            showOnTeamProfile: false,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.MeetingCreateInput>): Prisma.MeetingCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            openToAnyoneWithInvite: true,
            showOnTeamProfile: true,
            translations: {
                create: [
                    {
                        id: generatePK().toString(),
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
                    team: { teamId: "team-standup" },
                    translations: [{
                        language: "en",
                        name: "Daily Standup",
                        description: "15-minute daily synchronization",
                        link: "https://meet.example.com/standup",
                    }],
                    attendees: [
                        { userId: "team-lead" },
                        { userId: "developer-1" },
                        { userId: "developer-2" },
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
                    team: { teamId: "team-webinar" },
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
                    team: { teamId: "team-executive" },
                    translations: [{
                        language: "en",
                        name: "Q4 Strategy Session",
                        description: "Executive team strategy planning",
                    }],
                    attendees: [
                        { userId: "ceo" },
                        { userId: "cto" },
                        { userId: "cfo" },
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
                    team: { teamId: "team-weekly" },
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
                    team: { teamId: "team-sales" },
                    translations: [{
                        language: "en",
                        name: "Product Demo",
                        description: "Product demonstration for prospective client",
                        link: "https://meet.example.com/demo",
                    }],
                    invites: [
                        { userId: "client-1", message: "Looking forward to showing you our product!" },
                        { userId: "client-2", message: "Please join us for the demo" },
                    ],
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.MeetingInclude {
        return {
            team: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
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
                            name: true,
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
                            name: true,
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
        baseData: Prisma.MeetingCreateInput,
        config: MeetingRelationConfig,
        tx: any
    ): Promise<Prisma.MeetingCreateInput> {
        let data = { ...baseData };

        // Handle team connection (required)
        if (config.team) {
            data.team = {
                connect: { id: config.team.teamId },
            };
        } else {
            throw new Error('Meeting requires a team connection');
        }

        // Handle schedule connection
        if (config.schedule) {
            data.schedule = {
                connect: { id: config.schedule.scheduleId },
            };
        }

        // Handle attendees
        if (config.attendees && Array.isArray(config.attendees)) {
            data.attendees = {
                create: config.attendees.map(attendee => ({
                    id: generatePK().toString(),
                    userId: attendee.userId,
                })),
            };
        }

        // Handle invites
        if (config.invites && Array.isArray(config.invites)) {
            data.invites = {
                create: config.invites.map(invite => ({
                    id: generatePK().toString(),
                    userId: invite.userId,
                    status: "Pending",
                    message: invite.message || "You're invited to join our meeting",
                })),
            };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK().toString(),
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
        teamId: string,
        name: string,
        options?: {
            description?: string;
            link?: string;
            isPublic?: boolean;
            showOnProfile?: boolean;
        }
    ): Promise<Prisma.Meeting> {
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
        teamId: string,
        scheduleId: string,
        name: string
    ): Promise<Prisma.Meeting> {
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
    async addAttendees(meetingId: string, userIds: string[]): Promise<void> {
        await this.prisma.meetingAttendee.createMany({
            data: userIds.map(userId => ({
                id: generatePK().toString(),
                meetingId,
                userId,
            })),
            skipDuplicates: true,
        });
    }

    /**
     * Remove attendee from meeting
     */
    async removeAttendee(meetingId: string, userId: string): Promise<void> {
        await this.prisma.meetingAttendee.deleteMany({
            where: {
                meetingId,
                userId,
            },
        });
    }

    /**
     * Create meeting invites
     */
    async createInvites(
        meetingId: string,
        invites: Array<{ userId: string; message?: string }>
    ): Promise<void> {
        await this.prisma.meetingInvite.createMany({
            data: invites.map(invite => ({
                id: generatePK().toString(),
                meetingId,
                userId: invite.userId,
                status: "Pending",
                message: invite.message || "You're invited to join our meeting",
            })),
            skipDuplicates: true,
        });
    }

    protected async checkModelConstraints(record: Prisma.Meeting): Promise<string[]> {
        const violations: string[] = [];
        
        // Check team exists
        const team = await this.prisma.team.findUnique({
            where: { id: record.teamId },
        });
        
        if (!team) {
            violations.push('Team does not exist');
        }

        // Check schedule exists if specified
        if (record.scheduleId) {
            const schedule = await this.prisma.schedule.findUnique({
                where: { id: record.scheduleId },
            });
            
            if (!schedule) {
                violations.push('Schedule does not exist');
            }
        }

        // Check logical constraints
        if (!record.showOnTeamProfile && record.openToAnyoneWithInvite) {
            violations.push('Meeting open to anyone should be shown on team profile');
        }

        // Check attendee limit (hypothetical)
        const attendeeCount = await this.prisma.meetingAttendee.count({
            where: { meetingId: record.id },
        });
        
        const MAX_ATTENDEES = 1000;
        if (attendeeCount > MAX_ATTENDEES) {
            violations.push(`Meeting exceeds maximum attendee limit of ${MAX_ATTENDEES}`);
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            attendees: true,
            invites: true,
            translations: true,
            subscriptions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Meeting,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete in order of dependencies
        
        // Delete invites
        if (shouldDelete('invites') && record.invites?.length) {
            await tx.meetingInvite.deleteMany({
                where: { meetingId: record.id },
            });
        }

        // Delete attendees
        if (shouldDelete('attendees') && record.attendees?.length) {
            await tx.meetingAttendee.deleteMany({
                where: { meetingId: record.id },
            });
        }

        // Delete subscriptions
        if (shouldDelete('subscriptions') && record.subscriptions?.length) {
            await tx.notificationSubscription.deleteMany({
                where: { meetingId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete('translations') && record.translations?.length) {
            await tx.meetingTranslation.deleteMany({
                where: { meetingId: record.id },
            });
        }
    }

    /**
     * Get public meetings for a team
     */
    async getPublicTeamMeetings(teamId: string): Promise<Prisma.Meeting[]> {
        return await this.prisma.meeting.findMany({
            where: {
                teamId,
                showOnTeamProfile: true,
            },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: 'desc' },
        });
    }

    /**
     * Get meetings with open invites
     */
    async getOpenMeetings(): Promise<Prisma.Meeting[]> {
        return await this.prisma.meeting.findMany({
            where: {
                openToAnyoneWithInvite: true,
            },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: 'desc' },
        });
    }

    /**
     * Create different meeting types for testing
     */
    async createTestingScenarios(teamId: string): Promise<{
        publicMeeting: Prisma.Meeting;
        privateMeeting: Prisma.Meeting;
        recurringMeeting: Prisma.Meeting;
        webinar: Prisma.Meeting;
    }> {
        const publicMeeting = await this.createTeamMeeting(
            teamId,
            "Public Team Meeting",
            {
                description: "Open to all team members",
                isPublic: true,
                showOnProfile: true,
            }
        );

        const privateMeeting = await this.createTeamMeeting(
            teamId,
            "Private Strategy Session",
            {
                description: "Executive team only",
                isPublic: false,
                showOnProfile: false,
            }
        );

        const recurringMeeting = await this.createRecurringMeeting(
            teamId,
            "schedule-123",
            "Weekly Team Sync"
        );

        const webinar = await this.createTeamMeeting(
            teamId,
            "Product Launch Webinar",
            {
                description: "Join us for our exciting product launch!",
                link: "https://webinar.example.com/launch",
                isPublic: true,
                showOnProfile: true,
            }
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
export const createMeetingDbFactory = (prisma: PrismaClient) => 
    MeetingDbFactory.getInstance('Meeting', prisma);

// Export the class for type usage
export { MeetingDbFactory as MeetingDbFactoryClass };