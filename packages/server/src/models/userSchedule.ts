import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, UserSchedule, UserScheduleCreateInput, UserScheduleSearchInput, UserScheduleSortBy, UserScheduleUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { defaultPermissions } from "../utils";
import { userScheduleValidation } from "@shared/validation";

// const searcher = (): Searcher<
//     StandardSearchInput,
//     StandardSortBy,
//     Prisma.standardWhereInput
// > => ({
//     defaultSort: StandardSortBy.ScoreDesc,
//     sortBy: StandardSortBy,
//     searchFields: [
//         'createdById',
//         'createdTimeFrame',
//         'issuesId',
//         'labelsId',
//         'minScore',
//         'minBookmarks',
//         'minViews',
//         'ownedByOrganizationId',
//         'ownedByUserId',
//         'parentId',
//         'pullRequestsId',
//         'questionsId',
//         'standardTypeLatestVersion',
//         'tags',
//         'transfersId',
//         'translationLanguagesLatestVersion',
//         'updatedTimeFrame',
//         'visibility',
//     ],
//     searchStringQuery: ({ insensitive, languages }) => ({
//         OR: [
//             { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
//             { translations: { some: { language: languages ? { in: languages } : undefined, name: { ...insensitive } } } },
//         ]
//     }),
//     /**
//      * Can only query for your own schedules
//      */
//     customQueryData: (_, userData) => ({ user: { id: userData.id } }),
// })

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
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any),
            update: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any)
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