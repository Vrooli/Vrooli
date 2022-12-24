import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RunProjectSchedule, RunProjectScheduleCreateInput, RunProjectScheduleSearchInput, RunProjectScheduleSortBy, RunProjectScheduleUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'RunProjectSchedule' as const;
const suppFields = [] as const;
export const RunProjectScheduleModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: RunProjectScheduleCreateInput,
    GqlUpdate: RunProjectScheduleUpdateInput,
    GqlModel: RunProjectSchedule,
    GqlSearch: RunProjectScheduleSearchInput,
    GqlSort: RunProjectScheduleSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.run_project_scheduleUpsertArgs['create'],
    PrismaUpdate: Prisma.run_project_scheduleUpsertArgs['update'],
    PrismaModel: Prisma.run_project_scheduleGetPayload<SelectWrap<Prisma.run_project_scheduleSelect>>,
    PrismaSelect: Prisma.run_project_scheduleSelect,
    PrismaWhere: Prisma.run_project_scheduleWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project_schedule,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})