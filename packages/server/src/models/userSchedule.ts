import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

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

const displayer = (): Displayer<
    Prisma.user_scheduleSelect,
    Prisma.user_scheduleGetPayload<SelectWrap<Prisma.user_scheduleSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const UserScheduleModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user_schedule,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})