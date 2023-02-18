import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Reminder, ReminderCreateInput, ReminderSearchInput, ReminderSortBy, ReminderUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'Reminder' as const;
const suppFields = [] as const;
export const ReminderModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReminderCreateInput,
    GqlUpdate: ReminderUpdateInput,
    GqlModel: Reminder,
    GqlSearch: ReminderSearchInput,
    GqlSort: ReminderSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.reminderUpsertArgs['create'],
    PrismaUpdate: Prisma.reminderUpsertArgs['update'],
    PrismaModel: Prisma.reminderGetPayload<SelectWrap<Prisma.reminderSelect>>,
    PrismaSelect: Prisma.reminderSelect,
    PrismaWhere: Prisma.reminderWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name
    },
    format: {
        gqlRelMap: {
            __typename,
            reminderItems: 'ReminderItem',
            reminderList: 'ReminderList',
        },
        prismaRelMap: {
            __typename,
            reminderItems: 'ReminderItem',
            reminderList: 'ReminderList',
        },
        countFields: {},
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})