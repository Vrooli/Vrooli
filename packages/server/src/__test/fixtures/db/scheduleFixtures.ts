import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Schedule model - used for seeding test data
 */

export class ScheduleDbFactory {
    static createMinimal(
        forId: string,
        forType: string,
        overrides?: Partial<Prisma.ScheduleCreateInput>
    ): Prisma.ScheduleCreateInput {
        const base: Prisma.ScheduleCreateInput = {
            id: generatePK(),
            publicId: generatePublicId(),
            startTime: new Date(),
            endTime: new Date(Date.now() + 24 * 60 * 60 * 1000), // 1 day later
            timezone: "UTC",
            ...overrides,
        };

        // Add the appropriate connection based on object type
        const connections: Record<string, any> = {
            FocusMode: { focusMode: { connect: { id: forId } } },
            Meeting: { meeting: { connect: { id: forId } } },
            RunProject: { runProject: { connect: { id: forId } } },
            RunRoutine: { runRoutine: { connect: { id: forId } } },
        };

        return {
            ...base,
            ...(connections[forType] || {}),
        };
    }

    static createRecurring(
        forId: string,
        forType: string,
        recurrence: Partial<Prisma.ScheduleRecurrenceCreateWithoutScheduleInput>,
        overrides?: Partial<Prisma.ScheduleCreateInput>
    ): Prisma.ScheduleCreateInput {
        return {
            ...this.createMinimal(forId, forType, overrides),
            recurrences: {
                create: {
                    id: generatePK(),
                    recurrenceType: "Daily",
                    interval: 1,
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: null,
                    ...recurrence,
                },
            },
        };
    }

    static createWithExceptions(
        forId: string,
        forType: string,
        exceptions: Array<{ date: Date; newStartTime?: Date; newEndTime?: Date }>,
        overrides?: Partial<Prisma.ScheduleCreateInput>
    ): Prisma.ScheduleCreateInput {
        return {
            ...this.createMinimal(forId, forType, overrides),
            exceptions: {
                create: exceptions.map(ex => ({
                    id: generatePK(),
                    originalStartTime: ex.date,
                    newStartTime: ex.newStartTime,
                    newEndTime: ex.newEndTime,
                })),
            },
        };
    }

    static createWithLabels(
        forId: string,
        forType: string,
        labelIds: string[],
        overrides?: Partial<Prisma.ScheduleCreateInput>
    ): Prisma.ScheduleCreateInput {
        return {
            ...this.createMinimal(forId, forType, overrides),
            labels: {
                connect: labelIds.map(id => ({ id })),
            },
        };
    }
}

/**
 * Helper to seed schedules for testing
 */
export async function seedSchedules(
    prisma: any,
    options: {
        forObjects: Array<{ id: string; type: string }>;
        withRecurrence?: boolean;
        withExceptions?: boolean;
        withLabels?: string[];
    }
) {
    const schedules = [];

    for (const obj of options.forObjects) {
        let scheduleData: Prisma.ScheduleCreateInput;

        if (options.withRecurrence) {
            scheduleData = ScheduleDbFactory.createRecurring(
                obj.id,
                obj.type,
                {
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: 1, // Monday
                },
                {
                    ...(options.withLabels && {
                        labels: {
                            connect: options.withLabels.map(id => ({ id })),
                        },
                    }),
                }
            );
        } else if (options.withExceptions) {
            const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
            const nextWeek = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000);
            
            scheduleData = ScheduleDbFactory.createWithExceptions(
                obj.id,
                obj.type,
                [
                    { date: tomorrow, newStartTime: new Date(tomorrow.getTime() + 2 * 60 * 60 * 1000) },
                    { date: nextWeek }, // Cancelled
                ],
                {
                    ...(options.withLabels && {
                        labels: {
                            connect: options.withLabels.map(id => ({ id })),
                        },
                    }),
                }
            );
        } else {
            scheduleData = options.withLabels
                ? ScheduleDbFactory.createWithLabels(obj.id, obj.type, options.withLabels)
                : ScheduleDbFactory.createMinimal(obj.id, obj.type);
        }

        const schedule = await prisma.schedule.create({
            data: scheduleData,
            include: { 
                recurrences: true, 
                exceptions: true,
                labels: true,
            },
        });
        schedules.push(schedule);
    }

    return schedules;
}