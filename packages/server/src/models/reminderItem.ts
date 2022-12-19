import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReminderItemCreateInput,
    GqlUpdate: ReminderItemUpdateInput,
    GqlModel: ReminderItem,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.reminder_itemUpsertArgs['create'],
    PrismaUpdate: Prisma.reminder_itemUpsertArgs['update'],
    PrismaModel: Prisma.reminder_itemGetPayload<SelectWrap<Prisma.reminder_itemSelect>>,
    PrismaSelect: Prisma.reminder_itemSelect,
    PrismaWhere: Prisma.reminder_itemWhereInput,
}


const __typename = 'ReminderItem' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name
})

export const ReminderItemModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder_item,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})