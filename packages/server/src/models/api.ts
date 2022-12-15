import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Api, ApiSearchInput, ApiSortBy } from "../endpoints/types";
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { ApiVersionModel } from "./apiVersion";
import { StarModel } from "./star";
import { Displayer, Formatter, Searcher } from "./types";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";

type Model = {
    GqlModel: Api,
    GqlSearch: ApiSearchInput,
    GqlSort: ApiSortBy,
    PrismaModel: Prisma.apiGetPayload<SelectWrap<Prisma.apiSelect>>,
    PrismaSelect: Prisma.apiSelect,
    PrismaWhere: Prisma.apiWhereInput,
}

const __typename = 'Api' as const;

const suppFields = ['isStarred', 'isViewed', 'isUpvoted', 'permissionsRoot'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        createdBy: 'User',
        owner: {
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
        },
        parent: 'Api',
        tags: 'Tag',
        versions: 'ApiVersion',
        labels: 'Label',
        issues: 'Issue',
        pullRequests: 'PullRequest',
        questions: 'Question',
        starredBy: 'User',
        stats: 'StatsApi',
        transfers: 'Transfer',
    },
    prismaRelMap: {
        __typename,
        createdBy: 'User',
        ownedByUser: 'User',
        ownedByOrganization: 'Organization',
        parent: 'ApiVersion',
        tags: 'Tag',
        issues: 'Issue',
        starredBy: 'User',
        votedBy: 'Vote',
        viewedBy: 'View',
        pullRequests: 'PullRequest',
        versions: 'ApiVersion',
        labels: 'Label',
        stats: 'StatsApi',
        questions: 'Question',
        transfers: 'Transfer',
    },
    joinMap: { labels: 'label', starredBy: 'user', tags: 'tag' },
    countFields: ['versionsCount', 'pullRequestsCount', 'questionsCount', 'transfersCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => ({
            isStarred: async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
            isViewed: async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
            isUpvoted: async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
            permissionsRoot: async () => await getSingleTypePermissions(__typename, ids, prisma, userData),
        }),
    },
})

const searcher = (): Searcher<Model> => ({
    defaultSort: ApiSortBy.ScoreDesc,
    sortBy: ApiSortBy,
    searchFields: [
        'createdById',
        'createdTimeFrame',
        'maxScore',
        'maxStars',
        'minScore',
        'minStars',
        'ownedByOrganizationId',
        'ownedByUserId',
        'parentId',
        'tags',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: () => ({
        OR: [
            'tagsWrapped',
            'labelsWrapped',
            { versions: { some: 'transNameWrapped' } },
            { versions: { some: 'transSummaryWrapped' } }
        ]
    })
})

const displayer = (): Displayer<Model> => ({
    select: () => ({
        id: true,
        versions: {
            orderBy: { versionIndex: 'desc' },
            take: 1,
            select: ApiVersionModel.display.select(),
        }
    }),
    label: (select, languages) => select.versions.length > 0 ?
        ApiVersionModel.display.label(select.versions[0] as any, languages) : '',
})

export const ApiModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: searcher(),
    validate: {} as any,
})