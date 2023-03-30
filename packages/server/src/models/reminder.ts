import { Prisma } from "@prisma/client";
import { MaxObjects, Reminder, ReminderCreateInput, ReminderSearchInput, ReminderSortBy, ReminderUpdateInput } from '@shared/consts';
import { reminderValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { ReminderListModel } from "./reminderList";
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
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                name: data.name,
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: data.index,
                ...(await shapeHelper({ relation: 'reminderList', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'ReminderList', parentRelationshipName: 'reminders', data, ...rest })),
                ...(await shapeHelper({ relation: 'reminderItems', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'ReminderItem', parentRelationshipName: 'reminder', data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                name: noNull(data.name),
                description: noNull(data.description),
                dueDate: noNull(data.dueDate),
                index: noNull(data.index),
                isComplete: noNull(data.isComplete),
                ...(await shapeHelper({ relation: 'reminderItems', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'ReminderItem', parentRelationshipName: 'reminder', data, ...rest })),
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
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => ReminderListModel.validate!.isPublic(data.reminderList as any, languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ReminderListModel.validate!.owner(data.reminderList as any, userId),
        permissionResolvers: (params) => defaultPermissions(params),
        permissionsSelect: () => ({
            id: true,
            reminderList: 'ReminderList',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                reminderList: ReminderListModel.validate!.visibility.owner(userId),
            })
        }
    },
})