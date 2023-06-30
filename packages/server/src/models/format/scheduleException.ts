import { ScheduleExceptionModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ScheduleException" as const;
export const ScheduleExceptionFormat: Formatter<ScheduleExceptionModelLogic> = {
    gqlRelMap: {
        __typename,
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename,
        schedule: "Schedule",
    },
    countFields: {},
};
