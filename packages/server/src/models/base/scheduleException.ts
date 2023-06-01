import { PrismaType } from "../types";
import { ScheduleModel } from "./schedule";
import { ModelLogic, ScheduleExceptionModelLogic } from "./types";

const __typename = "ScheduleException" as const;
const suppFields = [] as const;
export const ScheduleExceptionModel: ModelLogic<ScheduleExceptionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.schedule_exception,
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
