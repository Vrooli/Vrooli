import { Prisma } from "@prisma/client";
import { Schedule, ScheduleCreateInput, ScheduleSearchInput, ScheduleSortBy, ScheduleUpdateInput } from '@shared/consts';
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { FocusModeModel } from "./focusMode";
import { MeetingModel } from "./meeting";
import { RunProjectModel } from "./runProject";
import { RunRoutineModel } from "./runRoutine";
import { ModelLogic } from "./types";

const __typename = 'Schedule' as const;
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
    PrismaCreate: Prisma.scheduleUpsertArgs['create'],
    PrismaUpdate: Prisma.scheduleUpsertArgs['update'],
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
            return '';
        }
    },
    format: {
        gqlRelMap: {
            __typename,
            exceptions: 'ScheduleException',
            labels: 'Label',
            recurrences: 'ScheduleRecurrence',
            runProjects: 'RunProject',
            runRoutines: 'RunRoutine',
            focusModes: 'FocusMode',
            meetings: 'Meeting',
        },
        prismaRelMap: {
            __typename,
            exceptions: 'ScheduleException',
            labels: 'Label',
            recurrences: 'ScheduleRecurrence',
            runProjects: 'RunProject',
            runRoutines: 'RunRoutine',
            focusModes: 'FocusMode',
            meetings: 'Meeting',
        },
        joinMap: { labels: 'label' },
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
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                { focusModes: { some: FocusModeModel.search!.searchStringQuery() } },
                { meetings: { some: MeetingModel.search!.searchStringQuery() } },
                { runProjects: { some: RunProjectModel.search!.searchStringQuery() } },
                { runRoutines: { some: RunRoutineModel.search!.searchStringQuery() } },
            ]
        }),
    },
    validate: {} as any,
})