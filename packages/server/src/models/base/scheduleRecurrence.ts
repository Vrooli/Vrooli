import { PrismaType } from "../types";
import { ScheduleModel } from "./schedule";
import { ModelLogic, ScheduleRecurrenceModelLogic } from "./types";

const __typename = "ScheduleRecurrence" as const;
const suppFields = [] as const;
export const ScheduleRecurrenceModel: ModelLogic<ScheduleRecurrenceModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.schedule_recurrence,
    display: {
        label: {
            select: () => ({ id: true, schedule: { select: ScheduleModel.display.label.select() } }),
            get: (select, languages) => ScheduleModel.display.label.get(select.schedule as any, languages),
        },
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
    mutate: {} as any,
    validate: {} as any,
});
