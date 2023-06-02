import { ScheduleRecurrenceModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ScheduleRecurrence" as const;
export const ScheduleRecurrenceFormat: Formatter<ScheduleRecurrenceModelLogic> = {
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
