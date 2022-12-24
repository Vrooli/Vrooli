import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Reminder, ReminderCreateInput, ReminderSearchInput, ReminderSortBy, ReminderUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReminderCreateInput,
    GqlUpdate: ReminderUpdateInput,
    GqlModel: Reminder,
    GqlSearch: ReminderSearchInput,
    GqlSort: ReminderSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.reminderUpsertArgs['create'],
    PrismaUpdate: Prisma.reminderUpsertArgs['update'],
    PrismaModel: Prisma.reminderGetPayload<SelectWrap<Prisma.reminderSelect>>,
    PrismaSelect: Prisma.reminderSelect,
    PrismaWhere: Prisma.reminderWhereInput,
}


const __typename = 'Reminder' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name
})

export const ReminderModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})