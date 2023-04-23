
/**
 * Creates a "where" object to find schedules that occur within a given time frame.
 * @param startDate The start of the time frame as a Date, in UTC.
 * @param endDate The end of the time frame as a Date, in UTC.
 */
export const schedulesWhereInTimeframe = (
    startDate: Date,
    endDate: Date,
) => ({
    endTime: { gte: new Date(startDate).toISOString() },
    startTime: { lt: new Date(endDate).toISOString() },
});

/**
 * Creates a "where" object to find schedule exceptions that occur within a given time frame.
 * @param startDate The start of the time frame as a Date, in UTC.
 * @param endDate The end of the time frame as a Date, in UTC.
 */
export const scheduleExceptionsWhereInTimeframe = (
    startDate: Date,
    endDate: Date,
) => ({
    originalStartTime: { gte: new Date(startDate).toISOString() },
    OR: [
        { newStartTime: { lte: new Date(endDate).toISOString() } },
        { newStartTime: null },
    ],
});

/**
 * Creates a "where" object to find schedule recurrences that occur within a given time frame.
 * @param startDate The start of the time frame as a Date, in UTC.
 * @param endDate The end of the time frame as a Date, in UTC.
 */
export const scheduleRecurrencesWhereInTimeframe = (
    startDate: Date,
    endDate: Date,
) => ({
    endDate: { gte: new Date(startDate).toISOString() },
});
