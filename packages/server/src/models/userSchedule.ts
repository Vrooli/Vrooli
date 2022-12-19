import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { UserSchedule, UserScheduleCreateInput, UserScheduleSearchInput, UserScheduleSortBy, UserScheduleUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: UserScheduleCreateInput,
    GqlUpdate: UserScheduleUpdateInput,
    GqlModel: UserSchedule,
    GqlSearch: UserScheduleSearchInput,
    GqlSort: UserScheduleSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.user_scheduleUpsertArgs['create'],
    PrismaUpdate: Prisma.user_scheduleUpsertArgs['update'],
    PrismaModel: Prisma.user_scheduleGetPayload<SelectWrap<Prisma.user_scheduleSelect>>,
    PrismaSelect: Prisma.user_scheduleSelect,
    PrismaWhere: Prisma.user_scheduleWhereInput,
}

const __typename = 'UserSchedule' as const;

const suppFields = [] as const;

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
//         'minStars',
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

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name,
})

export const UserScheduleModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user_schedule,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})