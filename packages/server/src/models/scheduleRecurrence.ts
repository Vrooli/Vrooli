import { Prisma } from "@prisma/client";
import { RunRoutineSearchInput, RunRoutineSortBy, ScheduleRecurrence, ScheduleRecurrenceCreateInput, ScheduleRecurrenceUpdateInput } from '@shared/consts';
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'ScheduleRecurrence' as const;
const suppFields = [] as const;
export const ScheduleRecurrenceModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ScheduleRecurrenceCreateInput,
    GqlUpdate: ScheduleRecurrenceUpdateInput,
    GqlModel: ScheduleRecurrence,
    GqlPermission: {},
    GqlSearch: RunRoutineSearchInput,
    GqlSort: RunRoutineSortBy,
    PrismaCreate: Prisma.schedule_recurrenceUpsertArgs['create'],
    PrismaUpdate: Prisma.schedule_recurrenceUpsertArgs['update'],
    PrismaModel: Prisma.schedule_recurrenceGetPayload<SelectWrap<Prisma.schedule_recurrenceSelect>>,
    PrismaSelect: Prisma.schedule_recurrenceSelect,
    PrismaWhere: Prisma.schedule_recurrenceWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.schedule_recurrence,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    validate: {} as any,
})