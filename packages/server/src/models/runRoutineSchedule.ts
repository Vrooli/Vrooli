import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RunRoutineSchedule, RunRoutineScheduleCreateInput, RunRoutineScheduleSearchInput, RunRoutineScheduleSortBy, RunRoutineScheduleUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: RunRoutineScheduleCreateInput,
    GqlUpdate: RunRoutineScheduleUpdateInput,
    GqlModel: RunRoutineSchedule,
    GqlSearch: RunRoutineScheduleSearchInput,
    GqlSort: RunRoutineScheduleSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.run_routine_scheduleUpsertArgs['create'],
    PrismaUpdate: Prisma.run_routine_scheduleUpsertArgs['update'],
    PrismaModel: Prisma.run_routine_scheduleGetPayload<SelectWrap<Prisma.run_routine_scheduleSelect>>,
    PrismaSelect: Prisma.run_routine_scheduleSelect,
    PrismaWhere: Prisma.run_routine_scheduleWhereInput,
}

const __typename = 'RunRoutineSchedule' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const RunRoutineScheduleModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_schedule,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})