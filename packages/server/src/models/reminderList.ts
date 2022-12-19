import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ReminderList, ReminderListCreateInput, ReminderListSearchInput, ReminderListSortBy, ReminderListUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";
import { UserScheduleModel } from "./userSchedule";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReminderListCreateInput,
    GqlUpdate: ReminderListUpdateInput,
    GqlModel: ReminderList,
    GqlSearch: ReminderListSearchInput,
    GqlSort: ReminderListSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.reminder_listUpsertArgs['create'],
    PrismaUpdate: Prisma.reminder_listUpsertArgs['update'],
    PrismaModel: Prisma.reminder_listGetPayload<SelectWrap<Prisma.reminder_listSelect>>,
    PrismaSelect: Prisma.reminder_listSelect,
    PrismaWhere: Prisma.reminder_listWhereInput,
}


const __typename = 'ReminderList' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, userSchedule: { select: UserScheduleModel.display.select() } }),
    // Label is schedule's label
    label: (select, languages) =>  UserScheduleModel.display.label(select.userSchedule as any, languages),
})

export const ReminderListModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder_list,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})