import { ScheduleModel } from "./schedule";
const __typename = "ScheduleException";
const suppFields = [];
export const ScheduleExceptionModel = ({
    __typename,
    delegate: (prisma) => prisma.schedule_exception,
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
//# sourceMappingURL=scheduleException.js.map