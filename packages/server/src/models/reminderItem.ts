import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ReminderItem, ReminderItemCreateInput, ReminderItemUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { reminderItemValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

const __typename = 'ReminderItem' as const;
const suppFields = [] as const;
export const ReminderItemModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReminderItemCreateInput,
    GqlUpdate: ReminderItemUpdateInput,
    GqlModel: ReminderItem,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.reminder_itemUpsertArgs['create'],
    PrismaUpdate: Prisma.reminder_itemUpsertArgs['update'],
    PrismaModel: Prisma.reminder_itemGetPayload<SelectWrap<Prisma.reminder_itemSelect>>,
    PrismaSelect: Prisma.reminder_itemSelect,
    PrismaWhere: Prisma.reminder_itemWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder_item,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name
    },
    format: {
        gqlRelMap: {
            __typename,
            reminder: 'Reminder',
        },
        prismaRelMap: {
            __typename,
            reminder: 'Reminder',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: data.index,
                name: data.name,
                ...(await shapeHelper({ relation: 'reminder', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Reminder', parentRelationshipName: 'reminderItems', data, ...rest })),
            }),
            update: async ({ data }) => ({
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: noNull(data.index),
                isComplete: noNull(data.isComplete),
                name: noNull(data.name),
            }),
        },
        yup: reminderItemValidation,
    },
    validate: {} as any,
})