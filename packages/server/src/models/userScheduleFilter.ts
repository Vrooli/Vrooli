import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { UserScheduleFilter, UserScheduleFilterCreateInput, UserScheduleSearchInput, UserScheduleSortBy } from "../endpoints/types";
import { PrismaType } from "../types";
import { TagModel } from "./tag";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: UserScheduleFilterCreateInput,
    GqlUpdate: undefined,
    GqlModel: UserScheduleFilter,
    GqlPermission: any,
    GqlSearch: UserScheduleSearchInput,
    GqlSort: UserScheduleSortBy,
    PrismaCreate: Prisma.user_schedule_filterUpsertArgs['create'],
    PrismaUpdate: Prisma.user_schedule_filterUpsertArgs['update'],
    PrismaModel: Prisma.user_schedule_filterGetPayload<SelectWrap<Prisma.user_schedule_filterSelect>>,
    PrismaSelect: Prisma.user_schedule_filterSelect,
    PrismaWhere: Prisma.user_schedule_filterWhereInput,
}

const __typename = 'UserScheduleFilter' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, tag: { select: TagModel.display.select() } }),
    label: (select, languages) => select.tag ? TagModel.display.label(select.tag as any, languages) : '',
})

export const UserScheduleFilterModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user_schedule_filter,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})