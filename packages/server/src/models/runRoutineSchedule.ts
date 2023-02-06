import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RunRoutineSchedule, RunRoutineScheduleCreateInput, RunRoutineScheduleSearchInput, RunRoutineScheduleSortBy, RunRoutineScheduleUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'RunRoutineSchedule' as const;
const suppFields = [] as const;
export const RunRoutineScheduleModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: RunRoutineScheduleCreateInput,
    GqlUpdate: RunRoutineScheduleUpdateInput,
    GqlModel: RunRoutineSchedule,
    GqlSearch: RunRoutineScheduleSearchInput,
    GqlSort: RunRoutineScheduleSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.run_routine_scheduleUpsertArgs['create'],
    PrismaUpdate: Prisma.run_routine_scheduleUpsertArgs['update'],
    PrismaModel: Prisma.run_routine_scheduleGetPayload<SelectWrap<Prisma.run_routine_scheduleSelect>>,
    PrismaSelect: Prisma.run_routine_scheduleSelect,
    PrismaWhere: Prisma.run_routine_scheduleWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_schedule,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})