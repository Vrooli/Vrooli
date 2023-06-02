import { ScheduleModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Schedule" as const;
export const ScheduleFormat: Formatter<ScheduleModelLogic> = {
    gqlRelMap: {
        __typename,
        exceptions: "ScheduleException",
        labels: "Label",
        recurrences: "ScheduleRecurrence",
        runProjects: "RunProject",
        runRoutines: "RunRoutine",
        focusModes: "FocusMode",
        meetings: "Meeting",
    },
    prismaRelMap: {
        __typename,
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
