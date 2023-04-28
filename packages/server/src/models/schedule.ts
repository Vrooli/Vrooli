import { MaxObjects, Schedule, ScheduleCreateInput, ScheduleSearchInput, ScheduleSortBy, ScheduleUpdateInput } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel, selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { getLogic } from "../getters";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { FocusModeModel } from "./focusMode";
import { MeetingModel } from "./meeting";
import { RunProjectModel } from "./runProject";
import { RunRoutineModel } from "./runRoutine";
import { ModelLogic } from "./types";

const __typename = "Schedule" as const;
const suppFields = [] as const;
export const ScheduleModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ScheduleCreateInput,
    GqlUpdate: ScheduleUpdateInput,
    GqlModel: Schedule,
    GqlPermission: {},
    GqlSearch: ScheduleSearchInput,
    GqlSort: ScheduleSortBy,
    PrismaCreate: Prisma.scheduleUpsertArgs["create"],
    PrismaUpdate: Prisma.scheduleUpsertArgs["update"],
    PrismaModel: Prisma.scheduleGetPayload<SelectWrap<Prisma.scheduleSelect>>,
    PrismaSelect: Prisma.scheduleSelect,
    PrismaWhere: Prisma.scheduleWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.schedule,
    display: {
        select: () => ({
            id: true,
            focusModes: selPad(FocusModeModel.display.select),
            meetings: selPad(MeetingModel.display.select),
            runProjects: selPad(RunProjectModel.display.select),
            runRoutines: selPad(RunRoutineModel.display.select),
        }),
        label: (select, languages) => {
            if (select.focusModes) return FocusModeModel.display.label(select.focusModes as any, languages);
            if (select.meetings) return MeetingModel.display.label(select.meetings as any, languages);
            if (select.runProjects) return RunProjectModel.display.label(select.runProjects as any, languages);
            if (select.runRoutines) return RunRoutineModel.display.label(select.runRoutines as any, languages);
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
    mutate: {} as any,
    search: {
        defaultSort: ScheduleSortBy.StartTimeAsc,
        sortBy: ScheduleSortBy,
        searchFields: {
            createdTimeFrame: true,
            endTimeFrame: true,
            scheduleForUserId: true,
            startTimeFrame: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                { focusModes: { some: FocusModeModel.search!.searchStringQuery() } },
                { meetings: { some: MeetingModel.search!.searchStringQuery() } },
                { runProjects: { some: RunProjectModel.search!.searchStringQuery() } },
                { runRoutines: { some: RunRoutineModel.search!.searchStringQuery() } },
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.scheduleSelect>(data, [
            ["focusModes", "FocusMode"],
            ["meetings", "Meeting"],
            ["runProjects", "RunProject"],
            ["runRoutines", "RunRoutine"],
        ], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => {
            // Find owner from the object that has the pull request
            const [onField, onData] = findFirstRel(data, [
                "focusModes",
                "meetings",
                "runProjects",
                "runRoutines",
            ]);
            const { validate } = getLogic(["validate"], onField as any, ["en"], "ScheduleModel.validate.owner");
            return Array.isArray(onData) && onData.length > 0 ?
                validate.owner(onData[0], userId)
                : {};
        },
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            focusModes: "FocusMode",
            meetings: "Meeting",
            runProjects: "RunProject",
            runRoutines: "RunRoutine",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { focusModes: { some: FocusModeModel.validate!.visibility.owner(userId) } },
                    { meetings: { some: MeetingModel.validate!.visibility.owner(userId) } },
                    { runProjects: { some: RunProjectModel.validate!.visibility.owner(userId) } },
                    { runRoutines: { some: RunRoutineModel.validate!.visibility.owner(userId) } },
                ],
            }),
        },
    },
});
