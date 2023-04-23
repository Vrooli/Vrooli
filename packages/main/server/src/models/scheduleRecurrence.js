import { ScheduleModel } from "./schedule";
const __typename = "ScheduleRecurrence";
const suppFields = [];
export const ScheduleRecurrenceModel = ({
    __typename,
    delegate: (prisma) => prisma.schedule_recurrence,
    display: {
        select: () => ({ id: true, schedule: { select: ScheduleModel.display.select() } }),
        label: (select, languages) => ScheduleModel.display.label(select.schedule, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            schedule: "Schedule",
        },
        prismaRelMap: {
            __typename,
            schedule: "Schedule",
        },
        countFields: {},
    },
    mutate: {},
    validate: {},
});
//# sourceMappingURL=scheduleRecurrence.js.map