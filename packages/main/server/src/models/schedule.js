import { ScheduleSortBy } from "@local/consts";
import { selPad } from "../builders";
import { FocusModeModel } from "./focusMode";
import { MeetingModel } from "./meeting";
import { RunProjectModel } from "./runProject";
import { RunRoutineModel } from "./runRoutine";
const __typename = "Schedule";
const suppFields = [];
export const ScheduleModel = ({
    __typename,
    delegate: (prisma) => prisma.schedule,
    display: {
        select: () => ({
            id: true,
            focusModes: selPad(FocusModeModel.display.select),
            meetings: selPad(MeetingModel.display.select),
            runProjects: selPad(RunProjectModel.display.select),
            runRoutines: selPad(RunRoutineModel.display.select),
        }),
        label: (select, languages) => {
            if (select.focusModes)
                return FocusModeModel.display.label(select.focusModes, languages);
            if (select.meetings)
                return MeetingModel.display.label(select.meetings, languages);
            if (select.runProjects)
                return RunProjectModel.display.label(select.runProjects, languages);
            if (select.runRoutines)
                return RunRoutineModel.display.label(select.runRoutines, languages);
            return "";
        },
    },
    format: {
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
    },
    mutate: {},
    search: {
        defaultSort: ScheduleSortBy.StartTimeAsc,
        sortBy: ScheduleSortBy,
        searchFields: {
            createdTimeFrame: true,
            endTimeFrame: true,
            scheduleForUserId: true,
            startTimeFrame: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                { focusModes: { some: FocusModeModel.search.searchStringQuery() } },
                { meetings: { some: MeetingModel.search.searchStringQuery() } },
                { runProjects: { some: RunProjectModel.search.searchStringQuery() } },
                { runRoutines: { some: RunRoutineModel.search.searchStringQuery() } },
            ],
        }),
    },
    validate: {},
});
//# sourceMappingURL=schedule.js.map