import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Reminder, ReminderCreateInput, ReminderSearchInput, ReminderSortBy, ReminderUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { reminderValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

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
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                name: data.name,
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: data.index,
                ...(await shapeHelper({ relation: 'reminderList', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'ReminderList', parentRelationshipName: 'reminders', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'reminderItems', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'ReminderItem', parentRelationshipName: 'reminder', data, prisma, userData })),
            }),
            update: async ({ data, prisma, userData }) => ({
                name: noNull(data.name),
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: noNull(data.index),
                isComplete: noNull(data.isComplete),
                ...(await shapeHelper({ relation: 'reminderItems', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'ReminderItem', parentRelationshipName: 'reminder', data, prisma, userData })),
            })
        },
        yup: reminderValidation,
    },
    search: {
        defaultSort: ReminderSortBy.DueDateAsc,
        sortBy: ReminderSortBy,
        searchFields: {
            createdTimeFrame: true,
            reminderListId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                'descriptionWrapped',
                'nameWrapped',
            ]
        }),
    },
    validate: {} as any,
})