import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { ScheduleExceptionDbFactory as EnhancedScheduleExceptionDbFactory } from "./ScheduleExceptionDbFactory.js";

/**
 * Database fixtures for ScheduleException model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const scheduleExceptionDbIds = {
    exception1: generatePK(),
    exception2: generatePK(),
    exception3: generatePK(),
    exception4: generatePK(),
    schedule1: generatePK(),
    schedule2: generatePK(),
    schedule3: generatePK(),
};

/**
 * Minimal schedule exception data for database creation
 */
export const minimalScheduleExceptionDb: Prisma.schedule_exceptionCreateInput = {
    id: scheduleExceptionDbIds.exception1,
    originalStartTime: new Date("2025-07-04T09:00:00Z"),
    newStartTime: new Date("2025-07-05T10:00:00Z"),
    newEndTime: new Date("2025-07-05T17:00:00Z"),
    schedule: {
        connect: { id: scheduleExceptionDbIds.schedule1 },
    },
};

/**
 * Complete schedule exception with all optional fields
 */
export const completeScheduleExceptionDb: Prisma.schedule_exceptionCreateInput = {
    id: scheduleExceptionDbIds.exception2,
    originalStartTime: new Date("2025-12-25T09:00:00Z"),
    newStartTime: new Date("2025-12-26T11:00:00Z"),
    newEndTime: new Date("2025-12-26T15:00:00Z"),
    schedule: {
        connect: { id: scheduleExceptionDbIds.schedule2 },
    },
};

/**
 * Schedule exception for cancellation (no new times)
 */
export const cancelledScheduleExceptionDb: Prisma.schedule_exceptionCreateInput = {
    id: scheduleExceptionDbIds.exception3,
    originalStartTime: new Date("2025-08-15T14:00:00Z"),
    newStartTime: null,
    newEndTime: null,
    schedule: {
        connect: { id: scheduleExceptionDbIds.schedule3 },
    },
};

/**
 * Legacy factory for creating schedule exception database fixtures with overrides
 * @deprecated Use EnhancedScheduleExceptionDbFactory instead
 */
export class ScheduleExceptionDbFactory extends EnhancedScheduleExceptionDbFactory {
    static createMinimal(
        scheduleId: string,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>
    ): Prisma.schedule_exceptionCreateInput {
        return {
            id: generatePK(),
            originalStartTime: new Date("2025-07-04T09:00:00Z"),
            newStartTime: new Date("2025-07-05T10:00:00Z"),
            newEndTime: new Date("2025-07-05T17:00:00Z"),
            schedule: {
                connect: { id: scheduleId },
            },
            ...overrides,
        };
    }

    static createComplete(
        scheduleId: string,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>
    ): Prisma.schedule_exceptionCreateInput {
        return {
            id: generatePK(),
            originalStartTime: new Date("2025-12-25T09:00:00Z"),
            newStartTime: new Date("2025-12-26T11:00:00Z"),
            newEndTime: new Date("2025-12-26T15:00:00Z"),
            schedule: {
                connect: { id: scheduleId },
            },
            ...overrides,
        };
    }

    static createCancellation(
        scheduleId: string,
        originalTime: Date,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>
    ): Prisma.schedule_exceptionCreateInput {
        return {
            id: generatePK(),
            originalStartTime: originalTime,
            newStartTime: null,
            newEndTime: null,
            schedule: {
                connect: { id: scheduleId },
            },
            ...overrides,
        };
    }

    static createRescheduled(
        scheduleId: string,
        originalTime: Date,
        newStartTime: Date,
        newEndTime: Date,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>
    ): Prisma.schedule_exceptionCreateInput {
        return {
            id: generatePK(),
            originalStartTime: originalTime,
            newStartTime: newStartTime,
            newEndTime: newEndTime,
            schedule: {
                connect: { id: scheduleId },
            },
            ...overrides,
        };
    }

    /**
     * Create schedule exception with time extension
     */
    static createExtended(
        scheduleId: string,
        originalTime: Date,
        extensionHours: number = 2,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>
    ): Prisma.schedule_exceptionCreateInput {
        const newEndTime = new Date(originalTime.getTime() + (extensionHours * 60 * 60 * 1000));
        return {
            id: generatePK(),
            originalStartTime: originalTime,
            newStartTime: originalTime, // Same start time
            newEndTime: newEndTime,
            schedule: {
                connect: { id: scheduleId },
            },
            ...overrides,
        };
    }

    /**
     * Create schedule exception moved to different day
     */
    static createMovedToNextDay(
        scheduleId: string,
        originalTime: Date,
        durationHours: number = 8,
        overrides?: Partial<Prisma.schedule_exceptionCreateInput>
    ): Prisma.schedule_exceptionCreateInput {
        const nextDay = new Date(originalTime);
        nextDay.setDate(nextDay.getDate() + 1);
        const newEndTime = new Date(nextDay.getTime() + (durationHours * 60 * 60 * 1000));
        
        return {
            id: generatePK(),
            originalStartTime: originalTime,
            newStartTime: nextDay,
            newEndTime: newEndTime,
            schedule: {
                connect: { id: scheduleId },
            },
            ...overrides,
        };
    }
}

/**
 * Helper to seed schedule exceptions for testing
 */
export async function seedScheduleExceptions(
    prisma: any,
    options: {
        scheduleId: string;
        count?: number;
        type?: "minimal" | "complete" | "mixed" | "cancellation" | "rescheduled";
        baseDate?: Date;
    }
) {
    const { scheduleId, count = 3, type = "mixed", baseDate = new Date("2025-07-04T09:00:00Z") } = options;
    const exceptions = [];

    for (let i = 0; i < count; i++) {
        let exceptionData: Prisma.schedule_exceptionCreateInput;
        const currentDate = new Date(baseDate);
        currentDate.setDate(currentDate.getDate() + i);

        switch (type) {
            case "minimal":
                exceptionData = ScheduleExceptionDbFactory.createMinimal(scheduleId, {
                    originalStartTime: currentDate,
                });
                break;
            case "complete":
                exceptionData = ScheduleExceptionDbFactory.createComplete(scheduleId, {
                    originalStartTime: currentDate,
                });
                break;
            case "cancellation":
                exceptionData = ScheduleExceptionDbFactory.createCancellation(scheduleId, currentDate);
                break;
            case "rescheduled":
                const newStartTime = new Date(currentDate);
                newStartTime.setHours(newStartTime.getHours() + 2);
                const newEndTime = new Date(newStartTime);
                newEndTime.setHours(newEndTime.getHours() + 6);
                exceptionData = ScheduleExceptionDbFactory.createRescheduled(
                    scheduleId,
                    currentDate,
                    newStartTime,
                    newEndTime
                );
                break;
            case "mixed":
            default:
                // Rotate between different types
                const types = ["minimal", "complete", "cancellation"];
                const selectedType = types[i % types.length];
                if (selectedType === "minimal") {
                    exceptionData = ScheduleExceptionDbFactory.createMinimal(scheduleId, {
                        originalStartTime: currentDate,
                    });
                } else if (selectedType === "complete") {
                    exceptionData = ScheduleExceptionDbFactory.createComplete(scheduleId, {
                        originalStartTime: currentDate,
                    });
                } else {
                    exceptionData = ScheduleExceptionDbFactory.createCancellation(scheduleId, currentDate);
                }
                break;
        }

        const exception = await prisma.schedule_exception.create({
            data: exceptionData,
            include: {
                schedule: true,
            },
        });
        exceptions.push(exception);
    }

    return exceptions;
}

/**
 * Helper to clean up schedule exceptions for testing
 */
export async function cleanupScheduleExceptions(prisma: any, exceptionIds: string[]) {
    return prisma.schedule_exception.deleteMany({
        where: {
            id: {
                in: exceptionIds,
            },
        },
    });
}

/**
 * Helper to verify schedule exception state
 */
export async function verifyScheduleExceptionState(
    prisma: any,
    exceptionId: string,
    expected: Partial<{
        originalStartTime: Date;
        newStartTime: Date | null;
        newEndTime: Date | null;
        scheduleId: string;
    }>
) {
    const actual = await prisma.schedule_exception.findUnique({
        where: { id: exceptionId },
        include: { schedule: true },
    });

    if (!actual) {
        throw new Error(`Schedule exception with ID ${exceptionId} not found`);
    }

    // Compare dates by converting to ISO strings for reliable comparison
    if (expected.originalStartTime) {
        const actualOriginal = new Date(actual.originalStartTime).toISOString();
        const expectedOriginal = expected.originalStartTime.toISOString();
        if (actualOriginal !== expectedOriginal) {
            throw new Error(`originalStartTime mismatch: expected ${expectedOriginal}, got ${actualOriginal}`);
        }
    }

    if (expected.newStartTime !== undefined) {
        const actualNew = actual.newStartTime ? new Date(actual.newStartTime).toISOString() : null;
        const expectedNew = expected.newStartTime ? expected.newStartTime.toISOString() : null;
        if (actualNew !== expectedNew) {
            throw new Error(`newStartTime mismatch: expected ${expectedNew}, got ${actualNew}`);
        }
    }

    if (expected.newEndTime !== undefined) {
        const actualEnd = actual.newEndTime ? new Date(actual.newEndTime).toISOString() : null;
        const expectedEnd = expected.newEndTime ? expected.newEndTime.toISOString() : null;
        if (actualEnd !== expectedEnd) {
            throw new Error(`newEndTime mismatch: expected ${expectedEnd}, got ${actualEnd}`);
        }
    }

    if (expected.scheduleId) {
        if (actual.scheduleId.toString() !== expected.scheduleId) {
            throw new Error(`scheduleId mismatch: expected ${expected.scheduleId}, got ${actual.scheduleId}`);
        }
    }

    return actual;
}