import { Prisma } from "@prisma/client";
import { RunRoutineSearchInput, RunRoutineSortBy, Schedule, ScheduleCreateInput, ScheduleUpdateInput } from '@shared/consts';
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
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
    GqlSearch: RunRoutineSearchInput,
    GqlSort: RunRoutineSortBy,
    PrismaCreate: Prisma.scheduleUpsertArgs['create'],
    PrismaUpdate: Prisma.scheduleUpsertArgs['update'],
    PrismaModel: Prisma.scheduleGetPayload<SelectWrap<Prisma.scheduleSelect>>,
    PrismaSelect: Prisma.scheduleSelect,
    PrismaWhere: Prisma.scheduleWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.schedule,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    validate: {} as any,
})