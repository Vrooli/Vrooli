/* eslint-disable no-magic-numbers */
import { type Prisma } from "@prisma/client";
import { generatePK, generatePublicId } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for Meeting model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing - cached for performance
let _meetingDbIds: Record<string, bigint> | null = null;
export function getMeetingDbIds() {
    if (!_meetingDbIds) {
        _meetingDbIds = {
            meeting1: generatePK(),
            meeting2: generatePK(),
            meeting3: generatePK(),
            invite1: generatePK(),
            invite2: generatePK(),
            invite3: generatePK(),
            translation1: generatePK(),
            translation2: generatePK(),
            translation3: generatePK(),
            team1: generatePK(),
            team2: generatePK(),
            user1: generatePK(),
            user2: generatePK(),
            user3: generatePK(),
            createdBy1: generatePK(),
            createdBy2: generatePK(),
        };
    }
    return _meetingDbIds;
}

/**
 * Enhanced test fixtures for Meeting model following standard structure
 */
export const meetingDbFixtures: DbTestFixtures<Prisma.meetingCreateInput> = {
    minimal: {
        id: getMeetingDbIds().meeting1,
        publicId: generatePublicId(),
        team: { connect: { id: getMeetingDbIds().team1 } },
        openToAnyoneWithInvite: false,
        showOnTeamProfile: true,
    },
    complete: {
        id: getMeetingDbIds().meeting2,
        publicId: generatePublicId(),
        team: { connect: { id: getMeetingDbIds().team1 } },
        openToAnyoneWithInvite: true,
        showOnTeamProfile: true,
        translations: {
            create: [
                {
                    id: getMeetingDbIds().translation1,
                    language: "en",
                    name: "Team Weekly Sync",
                    description: "Weekly team synchronization meeting to discuss progress and blockers",
                    link: "https://meet.example.com/team-sync",
                },
                {
                    id: getMeetingDbIds().translation2,
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
                    id: getMeetingDbIds().invite1,
                    user: { connect: { id: getMeetingDbIds().user1 } },
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
            id: getMeetingDbIds().meeting3,
            publicId: generatePublicId(),
            team: { connect: { id: BigInt("999999999999999") } }, // Non-existent bigint ID
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
        },
        invalidInviteUser: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: getMeetingDbIds().team1 } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
            invites: {
                create: [{
                    id: generatePK(),
                    user: { connect: { id: BigInt("999999999999998") } }, // Non-existent bigint ID
                }],
            },
        },
    },
    edgeCases: {
        pastMeeting: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: getMeetingDbIds().team1 } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
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
            team: { connect: { id: getMeetingDbIds().team1 } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
            invites: {
                create: Array.from({ length: 50 }, (_, i) => ({
                    id: generatePK(),
                    user: { connect: { id: generatePK() } }, // Generate unique user IDs
                })),
            },
        },
        multiLanguageTranslations: {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: getMeetingDbIds().team1 } },
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
            team: { connect: { id: getMeetingDbIds().team2 } }, // Use team2 for private meetings
            openToAnyoneWithInvite: false,
            showOnTeamProfile: false,
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
export class MeetingDbFactory extends EnhancedDbFactory<Prisma.meetingCreateInput> {

    /**
     * Get the test fixtures for Meeting model
     */
    protected getFixtures(): DbTestFixtures<Prisma.meetingCreateInput> {
        return meetingDbFixtures;
    }

    /**
     * Get Meeting-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getMeetingDbIds().meeting1, // Duplicate ID
                    publicId: generatePublicId(),
                    team: { connect: { id: getMeetingDbIds().team1 } },
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    team: { connect: { id: BigInt("999999999999997") } }, // Non-existent bigint ID
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId violates constraint
                    team: { connect: { id: getMeetingDbIds().team1 } },
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
                    team: { connect: { id: getMeetingDbIds().team1 } },
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                },
            },
            businessLogic: {
                pastMeetingWithInvites: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    team: { connect: { id: getMeetingDbIds().team1 } },
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: true,
                    invites: {
                        create: [{
                            id: generatePK(),
                            user: { connect: { id: getMeetingDbIds().user1 } },
                        }],
                    },
                },
                hiddenMeetingWithPublicInvites: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    team: { connect: { id: getMeetingDbIds().team1 } },
                    openToAnyoneWithInvite: true, // Public invites
                    showOnTeamProfile: false, // But hidden from team profile
                },
            },
        };
    }

    /**
     * Add team association to a meeting fixture
     */
    protected addTeamAssociation(data: Prisma.meetingCreateInput, teamId: bigint): Prisma.meetingCreateInput {
        return {
            ...data,
            team: { connect: { id: teamId } },
        };
    }

    /**
     * Add invites to a meeting fixture
     */
    protected addInvites(data: Prisma.meetingCreateInput, invites: Array<{ userId: bigint }>): Prisma.meetingCreateInput {
        return {
            ...data,
            invites: {
                create: invites.map(invite => ({
                    id: generatePK(),
                    user: { connect: { id: invite.userId } },
                })),
            },
        };
    }

    /**
     * Meeting-specific validation
     */
    protected validateSpecific(data: Prisma.meetingCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Meeting
        if (!data.team) errors.push("Meeting must be associated with a team");
        if (data.openToAnyoneWithInvite === undefined) errors.push("openToAnyoneWithInvite is required");
        if (data.showOnTeamProfile === undefined) errors.push("showOnTeamProfile is required");

        // Check business logic
        // Note: Schedule validation would be handled at the schedule level, not here

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
        teamId: bigint,
        overrides?: Partial<Prisma.meetingCreateInput>,
    ): Prisma.meetingCreateInput {
        const factory = new MeetingDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addTeamAssociation(data, teamId);
    }

    static createWithTranslations(
        teamId: bigint,
        translations: Array<{ language: string; name: string; description?: string; link?: string }>,
        overrides?: Partial<Prisma.meetingCreateInput>,
    ): Prisma.meetingCreateInput {
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
        teamId: bigint,
        overrides?: Partial<Prisma.meetingCreateInput>,
    ): Prisma.meetingCreateInput {
        return this.createWithTranslations(
            teamId,
            [{
                language: "en",
                name: "Scheduled Meeting",
                description: "Team sync meeting",
            }],
            {
                ...overrides,
            },
        );
    }

    static createWithInvites(
        teamId: bigint,
        invitedByIds: Array<{ userId: bigint }>,
        overrides?: Partial<Prisma.meetingCreateInput>,
    ): Prisma.meetingCreateInput {
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
        userId: bigint,
        meetingId: bigint,
        overrides?: Partial<Prisma.meeting_inviteCreateInput>,
    ): Prisma.meeting_inviteCreateInput {
        return {
            id: generatePK(),
            user: { connect: { id: userId } },
            meeting: { connect: { id: meetingId } },
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
        teamId: bigint;
        count?: number;
        withInvites?: Array<{ userId: bigint }>;
        scheduleDates?: Date[];
    },
): Promise<BulkSeedResult<any>> {
    const factory = new MeetingDbFactory();
    const meetings = [];
    const count = options.count || 1;
    let inviteCount = 0;

    for (let i = 0; i < count; i++) {
        let meetingData: Prisma.meetingCreateInput;

        if (options.withInvites) {
            meetingData = MeetingDbFactory.createWithInvites(
                options.teamId,
                options.withInvites.map(inv => ({
                    userId: inv.userId,
                })),
            );
            inviteCount += options.withInvites.length;
        } else {
            meetingData = MeetingDbFactory.createScheduled(options.teamId, {
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
            withAuth: 0,
            bots: 0,
            teams: 1,
            withInvites: inviteCount,
        },
    };
}
