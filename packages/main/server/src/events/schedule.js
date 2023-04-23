export const schedulesWhereInTimeframe = (startDate, endDate) => ({
    endTime: { gte: new Date(startDate).toISOString() },
    startTime: { lt: new Date(endDate).toISOString() },
});
export const scheduleExceptionsWhereInTimeframe = (startDate, endDate) => ({
    originalStartTime: { gte: new Date(startDate).toISOString() },
    OR: [
        { newStartTime: { lte: new Date(endDate).toISOString() } },
        { newStartTime: null },
    ],
});
export const scheduleRecurrencesWhereInTimeframe = (startDate, endDate) => ({
    endDate: { gte: new Date(startDate).toISOString() },
});
//# sourceMappingURL=schedule.js.map