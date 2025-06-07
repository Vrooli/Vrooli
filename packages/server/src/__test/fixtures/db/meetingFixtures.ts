import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Meeting model - used for seeding test data
 */

export class MeetingDbFactory {
    static createMinimal(
        teamId: string,
        overrides?: Partial<Prisma.MeetingCreateInput>
    ): Prisma.MeetingCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            team: { connect: { id: teamId } },
            openToAnyoneWithInvite: false,
            showOnTeamProfile: true,
            ...overrides,
        };
    }

    static createWithTranslations(
        teamId: string,
        translations: Array<{ language: string; name: string; description?: string; link?: string }>,
        overrides?: Partial<Prisma.MeetingCreateInput>
    ): Prisma.MeetingCreateInput {
        return {
            ...this.createMinimal(teamId, overrides),
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    name: t.name,
                    description: t.description,
                    link: t.link,
                })),
            },
        };
    }

    static createScheduled(
        teamId: string,
        scheduledFor: Date,
        overrides?: Partial<Prisma.MeetingCreateInput>
    ): Prisma.MeetingCreateInput {
        return this.createWithTranslations(
            teamId,
            [{ 
                language: "en", 
                name: "Scheduled Meeting",
                description: "Team sync meeting"
            }],
            {
                scheduledFor,
                ...overrides,
            }
        );
    }

    static createWithInvites(
        teamId: string,
        invitedByIds: Array<{ userId: string; createdById: string }>,
        overrides?: Partial<Prisma.MeetingCreateInput>
    ): Prisma.MeetingCreateInput {
        return {
            ...this.createWithTranslations(
                teamId,
                [{ language: "en", name: "Meeting with Invites" }],
                overrides
            ),
            invites: {
                create: invitedByIds.map(({ userId, createdById }) => ({
                    id: generatePK(),
                    user: { connect: { id: userId } },
                    createdBy: { connect: { id: createdById } },
                })),
            },
        };
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
        overrides?: Partial<Prisma.MeetingInviteCreateInput>
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
 * Helper to seed meetings for testing
 */
export async function seedMeetings(
    prisma: any,
    options: {
        teamId: string;
        createdById: string;
        count?: number;
        withInvites?: Array<{ userId: string }>;
        scheduleDates?: Date[];
    }
) {
    const meetings = [];
    const count = options.count || 1;

    for (let i = 0; i < count; i++) {
        const scheduledFor = options.scheduleDates?.[i] || new Date(Date.now() + (i + 1) * 24 * 60 * 60 * 1000);
        
        const meetingData = options.withInvites
            ? MeetingDbFactory.createWithInvites(
                options.teamId,
                options.withInvites.map(inv => ({ 
                    userId: inv.userId, 
                    createdById: options.createdById 
                })),
                { scheduledFor }
            )
            : MeetingDbFactory.createScheduled(options.teamId, scheduledFor, {
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: `Meeting ${i + 1}`,
                        description: `Description for meeting ${i + 1}`,
                    }],
                },
            });

        const meeting = await prisma.meeting.create({
            data: meetingData,
            include: { invites: true },
        });
        meetings.push(meeting);
    }

    return meetings;
}