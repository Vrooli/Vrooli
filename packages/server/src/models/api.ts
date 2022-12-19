import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Api, ApiCreateInput, ApiSearchInput, ApiSortBy, ApiUpdateInput, RootPermission } from "../endpoints/types";
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { ApiVersionModel } from "./apiVersion";
import { StarModel } from "./star";
import { ModelLogic } from "./types";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";

const __typename = 'Api' as const;
const suppFields = ['isStarred', 'isViewed', 'isUpvoted', 'permissionsRoot'] as const;
export const ApiModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: ApiCreateInput,
    GqlUpdate: ApiUpdateInput,
    GqlModel: Api,
    GqlPermission: RootPermission,
    GqlSearch: ApiSearchInput,
    GqlSort: ApiSortBy,
    PrismaCreate: Prisma.apiUpsertArgs['create'],
    PrismaUpdate: Prisma.apiUpsertArgs['update'],
    PrismaModel: Prisma.apiGetPayload<SelectWrap<Prisma.apiSelect>>,
    PrismaSelect: Prisma.apiSelect,
    PrismaWhere: Prisma.apiWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.api,
    display: {
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
    },
    format: {
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
    },
    mutate: {} as any,
    search: {
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
    },
    validate: {} as any,
})