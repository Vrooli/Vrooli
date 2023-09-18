import { ScheduleModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ScheduleFormat: Formatter<ScheduleModelLogic> = {
    gqlRelMap: {
        __typename: "Schedule",
        exceptions: "ScheduleException",
        labels: "Label",
        recurrences: "ScheduleRecurrence",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
        focusModes: "FocusMode",
        meetings: "Meeting",
    },
    prismaRelMap: {
        __typename: "Schedule",
        exceptions: "ScheduleException",
        labels: "Label",
        recurrences: "ScheduleRecurrence",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
        focusModes: "FocusMode",
        meetings: "Meeting",
    },
    joinMap: { labels: "label" },
    countFields: {},
};
