import { ScheduleRecurrenceModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ScheduleRecurrenceFormat: Formatter<ScheduleRecurrenceModelLogic> = {
    gqlRelMap: {
        __typename: "ScheduleRecurrence",
        schedule: "Schedule",
    },
    prismaRelMap: {
        __typename: "ScheduleRecurrence",
        schedule: "Schedule",
    },
    countFields: {},
};
