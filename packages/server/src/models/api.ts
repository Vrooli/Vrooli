import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Api } from "../endpoints/types";
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { ApiVersionModel } from "./apiVersion";
import { StarModel } from "./star";
import { Displayer, Formatter } from "./types";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";

const __typename = 'Api' as const;

const suppFields = ['isStarred', 'isViewed', 'isUpvoted', 'permissionsRoot'] as const;
const formatter = (): Formatter<Api, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        createdBy: 'User',
        owner: ['Organization', 'User'],
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
    joinMap: { labels: 'label', starredBy: 'user', tags: 'tag' },
    countFields: ['versionsCount', 'pullRequestsCount', 'questionsCount', 'transfersCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename)],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename)],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename)],
            ['permissionsRoot', async () => await getSingleTypePermissions(__typename, ids, prisma, userData)],
        ],
    },
})

const displayer = (): Displayer<
    Prisma.apiSelect,
    Prisma.apiGetPayload<SelectWrap<Prisma.apiSelect>>
> => ({
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
    search: {} as any,
    validate: {} as any,
})