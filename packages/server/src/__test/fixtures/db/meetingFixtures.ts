import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Meeting model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const meetingDbIds = {
    meeting1: generatePK(),
    meeting2: generatePK(),
    meeting3: generatePK(),
    invite1: generatePK(),
    invite2: generatePK(),
    invite3: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
    translation3: generatePK(),
};

/**
 * Enhanced test fixtures for Meeting model following standard structure
 */
export const meetingDbFixtures: DbTestFixtures<Prisma.MeetingCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        team: { connect: { id: "team_placeholder_id" } },
        openToAnyoneWithInvite: false,
        showOnTeamProfile: true,
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        team: { connect: { id: "team_placeholder_id" } },
        openToAnyoneWithInvite: true,
        showOnTeamProfile: true,
        scheduledFor: new Date(Date.now() + 24 * 60 * 60 * 1000),
        translations: {
            create: [
                {
                    id: generatePK(),
                    language: "en",
                    name: "Team Weekly Sync",
                    description: "Weekly team synchronization meeting to discuss progress and blockers",
                    link: "https://meet.example.com/team-sync",
                },
                {
                    id: generatePK(),
                    language: "es",
                    name: "Sincronización Semanal del Equipo",
                    description: "Reunión semanal de sincronización del equipo para discutir el progreso y los bloqueos",
                    link: "https://meet.example.com/team-sync",
                },
            ],
        },
        invites: {
            create: [
                {
                    id: generatePK(),
                    user: { connect: { id: "user_placeholder_id_1" } },
                    createdBy: { connect: { id: "user_placeholder_id_2" } },
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required team connection
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            openToAnyoneWithInvite: "yes", // Should be boolean
            showOnTeamProfile: "no", // Should be boolean
            scheduledFor: "not-a-date", // Should be Date
        },
        invalidTeamConnection: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: "non-existent-team-id" } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
        },
        invalidInviteUser: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: "team_placeholder_id" } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
            invites: {
                create: [{
                    id: generatePK(),
                    user: { connect: { id: "non-existent-user-id" } },
                    createdBy: { connect: { id: "user_placeholder_id" } },
                }],
            },
        },
    },
    edgeCases: {
        pastMeeting: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: "team_placeholder_id" } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
            scheduledFor: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // 1 week ago
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Past Meeting",
                    description: "A meeting that was scheduled in the past",
                }],
            },
        },
        manyInvites: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: "team_placeholder_id" } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
            invites: {
                create: Array.from({ length: 50 }, (_, i) => ({
                    id: generatePK(),
                    user: { connect: { id: `user_${i}_id` } },
                    createdBy: { connect: { id: "creator_user_id" } },
                })),
            },
        },
        multiLanguageTranslations: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: "team_placeholder_id" } },
            openToAnyoneWithInvite: true,
            showOnTeamProfile: true,
            translations: {
                create: [
                    { id: generatePK(), language: "en", name: "Global Team Meeting", description: "International team sync" },
                    { id: generatePK(), language: "es", name: "Reunión Global del Equipo", description: "Sincronización del equipo internacional" },
                    { id: generatePK(), language: "fr", name: "Réunion d'Équipe Globale", description: "Synchronisation de l'équipe internationale" },
                    { id: generatePK(), language: "de", name: "Globales Team-Meeting", description: "Internationale Team-Synchronisation" },
                    { id: generatePK(), language: "ja", name: "グローバルチームミーティング", description: "国際チーム同期" },
                ],
            },
        },
        privateTeamMeeting: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: "private_team_id" } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: false,
            scheduledFor: new Date(Date.now() + 2 * 60 * 60 * 1000), // 2 hours from now
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Private Strategy Meeting",
                    description: "Confidential discussion about upcoming initiatives",
                }],
            },
        },
    },
};

/**
 * Enhanced factory for creating meeting database fixtures
 */
export class MeetingDbFactory extends EnhancedDbFactory<Prisma.MeetingCreateInput> {
    
    /**
     * Get the test fixtures for Meeting model
     */
    protected getFixtures(): DbTestFixtures<Prisma.MeetingCreateInput> {
        return meetingDbFixtures;
    }

    /**
     * Get Meeting-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: meetingDbIds.meeting1, // Duplicate ID
                    publicId: generatePublicId(),
                    team: { connect: { id: "team_placeholder_id" } },
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    team: { connect: { id: "non-existent-team-id" } },
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId violates constraint
                    team: { connect: { id: "team_placeholder_id" } },
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                },
            },
            validation: {
                requiredFieldMissing: meetingDbFixtures.invalid.missingRequired,
                invalidDataType: meetingDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    team: { connect: { id: "team_placeholder_id" } },
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                    scheduledFor: new Date("invalid-date"),
                },
            },
            businessLogic: {
                pastMeetingWithInvites: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    team: { connect: { id: "team_placeholder_id" } },
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                    scheduledFor: new Date(Date.now() - 24 * 60 * 60 * 1000), // Past date
                    invites: {
                        create: [{
                            id: generatePK(),
                            user: { connect: { id: "user_placeholder_id" } },
                            createdBy: { connect: { id: "creator_user_id" } },
                        }],
                    },
                },
                hiddenMeetingWithPublicInvites: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    team: { connect: { id: "team_placeholder_id" } },
                    openToAnyoneWithInvite: true, // Public invites
                    showOnTeamProfile: false, // But hidden from team profile
                },
            },
        };
    }

    /**
     * Add team association to a meeting fixture
     */
    protected addTeamAssociation(data: Prisma.MeetingCreateInput, teamId: string): Prisma.MeetingCreateInput {
        return {
            ...data,
            team: { connect: { id: teamId } },
        };
    }

    /**
     * Add invites to a meeting fixture
     */
    protected addInvites(data: Prisma.MeetingCreateInput, invites: Array<{ userId: string; createdById: string }>): Prisma.MeetingCreateInput {
        return {
            ...data,
            invites: {
                create: invites.map(invite => ({
                    id: generatePK(),
                    user: { connect: { id: invite.userId } },
                    createdBy: { connect: { id: invite.createdById } },
                })),
            },
        };
    }

    /**
     * Meeting-specific validation
     */
    protected validateSpecific(data: Prisma.MeetingCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Meeting
        if (!data.team) errors.push("Meeting must be associated with a team");
        if (data.openToAnyoneWithInvite === undefined) errors.push("openToAnyoneWithInvite is required");
        if (data.showOnTeamProfile === undefined) errors.push("showOnTeamProfile is required");

        // Check business logic
        if (data.scheduledFor && data.scheduledFor < new Date()) {
            warnings.push("Meeting is scheduled in the past");
        }

        if (data.openToAnyoneWithInvite && !data.showOnTeamProfile) {
            warnings.push("Meeting allows public invites but is hidden from team profile");
        }

        // Check translation requirements
        if (data.invites && (!data.translations || !data.translations.create)) {
            warnings.push("Meeting with invites should have translations for better UX");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        teamId: string,
        overrides?: Partial<Prisma.MeetingCreateInput>,
    ): Prisma.MeetingCreateInput {
        const factory = new MeetingDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addTeamAssociation(data, teamId);
    }

    static createWithTranslations(
        teamId: string,
        translations: Array<{ language: string; name: string; description?: string; link?: string }>,
        overrides?: Partial<Prisma.MeetingCreateInput>,
    ): Prisma.MeetingCreateInput {
        const factory = new MeetingDbFactory();
        const data = factory.createMinimal({
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    name: t.name,
                    description: t.description,
                    link: t.link,
                })),
            },
            ...overrides,
        });
        return factory.addTeamAssociation(data, teamId);
    }

    static createScheduled(
        teamId: string,
        scheduledFor: Date,
        overrides?: Partial<Prisma.MeetingCreateInput>,
    ): Prisma.MeetingCreateInput {
        return this.createWithTranslations(
            teamId,
            [{ 
                language: "en", 
                name: "Scheduled Meeting",
                description: "Team sync meeting",
            }],
            {
                scheduledFor,
                ...overrides,
            },
        );
    }

    static createWithInvites(
        teamId: string,
        invitedByIds: Array<{ userId: string; createdById: string }>,
        overrides?: Partial<Prisma.MeetingCreateInput>,
    ): Prisma.MeetingCreateInput {
        const factory = new MeetingDbFactory();
        const data = this.createWithTranslations(
            teamId,
            [{ language: "en", name: "Meeting with Invites" }],
            overrides,
        );
        return factory.addInvites(data, invitedByIds);
    }
}

/**
 * Database fixtures for MeetingInvite model
 */
export class MeetingInviteDbFactory {
    static createMinimal(
        userId: string,
        meetingId: string,
        createdById: string,
        overrides?: Partial<Prisma.MeetingInviteCreateInput>,
    ): Prisma.MeetingInviteCreateInput {
        return {
            id: generatePK(),
            user: { connect: { id: userId } },
            meeting: { connect: { id: meetingId } },
            createdBy: { connect: { id: createdById } },
            ...overrides,
        };
    }
}

/**
 * Enhanced helper to seed multiple test meetings with comprehensive options
 */
export async function seedMeetings(
    prisma: any,
    options: {
        teamId: string;
        createdById: string;
        count?: number;
        withInvites?: Array<{ userId: string }>;
        scheduleDates?: Date[];
    },
): Promise<BulkSeedResult<any>> {
    const factory = new MeetingDbFactory();
    const meetings = [];
    const count = options.count || 1;
    let inviteCount = 0;
    let scheduledCount = 0;

    for (let i = 0; i < count; i++) {
        const scheduledFor = options.scheduleDates?.[i] || new Date(Date.now() + (i + 1) * 24 * 60 * 60 * 1000);
        
        let meetingData: Prisma.MeetingCreateInput;
        
        if (options.withInvites) {
            meetingData = MeetingDbFactory.createWithInvites(
                options.teamId,
                options.withInvites.map(inv => ({ 
                    userId: inv.userId, 
                    createdById: options.createdById, 
                })),
                { scheduledFor },
            );
            inviteCount += options.withInvites.length;
        } else {
            meetingData = MeetingDbFactory.createScheduled(options.teamId, scheduledFor, {
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: `Meeting ${i + 1}`,
                        description: `Description for meeting ${i + 1}`,
                    }],
                },
            });
        }
        
        if (scheduledFor > new Date()) {
            scheduledCount++;
        }

        const meeting = await prisma.meeting.create({
            data: meetingData,
            include: { 
                invites: true,
                translations: true,
                team: true,
            },
        });
        meetings.push(meeting);
    }

    return {
        records: meetings,
        summary: {
            total: meetings.length,
            withInvites: inviteCount,
            scheduled: scheduledCount,
            teams: 1,
        },
    };
}
