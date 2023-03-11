import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, UserSchedule, UserScheduleCreateInput, UserScheduleSearchInput, UserScheduleSortBy, UserScheduleUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { defaultPermissions, labelShapeHelper } from "../utils";
import { userScheduleValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

const __typename = 'UserSchedule' as const;
const suppFields = [] as const;
export const UserScheduleModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: UserScheduleCreateInput,
    GqlUpdate: UserScheduleUpdateInput,
    GqlModel: UserSchedule,
    GqlSearch: UserScheduleSearchInput,
    GqlSort: UserScheduleSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.user_scheduleUpsertArgs['create'],
    PrismaUpdate: Prisma.user_scheduleUpsertArgs['update'],
    PrismaModel: Prisma.user_scheduleGetPayload<SelectWrap<Prisma.user_scheduleSelect>>,
    PrismaSelect: Prisma.user_scheduleSelect,
    PrismaWhere: Prisma.user_scheduleWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user_schedule,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            filters: 'UserScheduleFilter',
            labels: 'Label',
            reminderList: 'ReminderList',
        },
        prismaRelMap: {
            __typename,
            reminderList: 'ReminderList',
            resourceList: 'ResourceList',
            user: 'User',
            labels: 'Label',
            filters: 'UserScheduleFilter',
        },
        countFields: {},
        joinMap: { labels: 'label' },
    },
    mutate: {
        shape: {
            create: async ({ prisma, userData, data }) => ({
                id: data.id,
                name: data.name,
                description: noNull(data.description),
                timeZone: noNull(data.timeZone),
                eventStart: noNull(data.eventStart),
                eventEnd: noNull(data.eventEnd),
                recurring: noNull(data.recurring),
                recurrStart: noNull(data.recurrStart),
                recurrEnd: noNull(data.recurrEnd),
                user: { connect: { id: userData.id } },
                ...(await shapeHelper({ relation: 'filters', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'UserScheduleFilter', parentRelationshipName: 'userSchedule', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'reminderList', relTypes: ['Connect', 'Create'], isOneToOne: true, isRequired: false, objectType: 'ReminderList', parentRelationshipName: 'userSchedule', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'userSchedule', data, prisma, userData })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Create'], parentType: 'UserSchedule', relation: 'labels', data, prisma, userData })),

            }),
            update: async ({ prisma, userData, data }) => ({
                name: noNull(data.name),
                description: noNull(data.description),
                timeZone: noNull(data.timeZone),
                eventStart: noNull(data.eventStart),
                eventEnd: noNull(data.eventEnd),
                recurring: noNull(data.recurring),
                recurrStart: noNull(data.recurrStart),
                recurrEnd: noNull(data.recurrEnd),
                user: { connect: { id: userData.id } },
                ...(await shapeHelper({ relation: 'filters', relTypes: ['Create', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'UserScheduleFilter', parentRelationshipName: 'userSchedule', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'reminderList', relTypes: ['Connect', 'Disconnect', 'Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'ReminderList', parentRelationshipName: 'userSchedule', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'resourceList', relTypes: ['Create', 'Update'], isOneToOne: true, isRequired: false, objectType: 'ResourceList', parentRelationshipName: 'userSchedule', data, prisma, userData })),
                ...(await labelShapeHelper({ relTypes: ['Connect', 'Disconnect', 'Create'], parentType: 'UserSchedule', relation: 'labels', data, prisma, userData })),
            }),
        },
        yup: userScheduleValidation,
    },
    search: {
        defaultSort: UserScheduleSortBy.TitleAsc,
        sortBy: UserScheduleSortBy,
        searchFields: {
            createdTimeFrame: true,
            eventStartTimeFrame: true,
            eventEndTimeFrame: true,
            recurrStartTimeFrame: true,
            recurrEndTimeFrame: true,
            labelsIds: true,
            timeZone: true,
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
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.user,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            user: 'User',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                user: { id: userId },
            }),
        },
    },
})