import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrependString, SmartContract, SmartContractCreateInput, SmartContractSearchInput, SmartContractSortBy, SmartContractUpdateInput, SmartContractYou } from '@shared/consts';
import { PrismaType } from "../types";
import { SmartContractVersionModel } from "./smartContractVersion";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { StarModel } from "./star";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";
import { getLabels } from "../getters";

const __typename = 'SmartContract' as const;
type Permissions = Pick<SmartContractYou, 'canDelete' | 'canEdit' | 'canStar' | 'canTransfer' | 'canView' | 'canVote'>;
const suppFields = ['you.canDelete', 'you.canEdit', 'you.canStar', 'you.canTransfer', 'you.canView', 'you.canVote', 'you.isStarred', 'you.isUpvoted', 'you.isViewed', 'translatedName'] as const;
export const SmartContractModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: SmartContractCreateInput,
    GqlUpdate: SmartContractUpdateInput,
    GqlModel: SmartContract,
    GqlSearch: SmartContractSearchInput,
    GqlSort: SmartContractSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.smart_contractUpsertArgs['create'],
    PrismaUpdate: Prisma.smart_contractUpsertArgs['update'],
    PrismaModel: Prisma.smart_contractGetPayload<SelectWrap<Prisma.smart_contractSelect>>,
    PrismaSelect: Prisma.smart_contractSelect,
    PrismaWhere: Prisma.smart_contractWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.smart_contract,
    display: {
        select: () => ({
            id: true,
            versions: {
                orderBy: { versionIndex: 'desc' },
                take: 1,
                select: SmartContractVersionModel.display.select(),
            }
        }),
        label: (select, languages) => select.versions.length > 0 ?
            SmartContractVersionModel.display.label(select.versions[0] as any, languages) : '',
    },
    format: {
        gqlRelMap: {
            __typename,
            createdBy: 'User',
            issues: 'Issue',
            labels: 'Label',
            owner: {
                ownedByUser: 'User',
                ownedByOrganization: 'Organization',
            },
            parent: 'SmartContract',
            pullRequests: 'PullRequest',
            questions: 'Question',
            starredBy: 'User',
            tags: 'Tag',
            transfers: 'Transfer',
            versions: 'NoteVersion',
        },
        prismaRelMap: {
            __typename,
            createdBy: 'User',
            issues: 'Issue',
            labels: 'Label',
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
            parent: 'NoteVersion',
            pullRequests: 'PullRequest',
            questions: 'Question',
            starredBy: 'User',
            tags: 'Tag',
            transfers: 'Transfer',
            versions: 'NoteVersion',
        },
        joinMap: { labels: 'label', starredBy: 'user', tags: 'tag' },
        countFields: {
            issuesCount: true,
            pullRequestsCount: true,
            questionsCount: true,
            transfersCount: true,
            versionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                    'you.isStarred': await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
                    'you.isViewed': await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                    'you.isUpvoted': await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    'translatedName': await getLabels(ids, __typename, prisma, userData?.languages ?? ['en'], 'smartContract.translatedName')
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})