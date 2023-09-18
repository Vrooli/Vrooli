import { ScheduleExceptionModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ScheduleExceptionFormat: Formatter<ScheduleExceptionModelLogic> = {
    gqlRelMap: {
        __typename: "ScheduleException",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "ScheduleException",
        schedule: "Schedule",
    },
    countFields: {},
};
