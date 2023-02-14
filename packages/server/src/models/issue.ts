import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Issue, IssueCreateInput, IssueSearchInput, IssueSortBy, IssueUpdateInput, IssueYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel, defaultPermissions, oneIsPublic } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { StarModel } from "./bookmark";
import { ModelLogic } from "./types";
import { VoteModel } from "./vote";

const __typename = 'Issue' as const;
type Permissions = Pick<IssueYou, 'canComment' | 'canDelete' | 'canUpdate' | 'canStar' | 'canReport' | 'canRead' | 'canVote'>;
const suppFields = ['you'] as const;
export const IssueModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: IssueCreateInput,
    GqlUpdate: IssueUpdateInput,
    GqlModel: Issue,
    GqlSearch: IssueSearchInput,
    GqlSort: IssueSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.issueUpsertArgs['create'],
    PrismaUpdate: Prisma.issueUpsertArgs['update'],
    PrismaModel: Prisma.issueGetPayload<SelectWrap<Prisma.issueSelect>>,
    PrismaSelect: Prisma.issueSelect,
    PrismaWhere: Prisma.issueWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.issue,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages)
    },
    format: {
        gqlRelMap: {
            __typename,
            closedBy: 'User',
            comments: 'Comment',
            createdBy: 'User',
            labels: 'Label',
            reports: 'Report',
            bookmarkedBy: 'User',
            to: {
                api: 'Api',
                organization: 'Organization',
                note: 'Note',
                project: 'Project',
                routine: 'Routine',
                smartContract: 'SmartContract',
                standard: 'Standard',
            }
        },
        prismaRelMap: {
            __typename,
            api: 'Api',
            organization: 'Organization',
            note: 'Note',
            project: 'Project',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
            closedBy: 'User',
            comments: 'Comment',
            labels: 'Label',
            reports: 'Report',
            votedBy: 'Vote',
            bookmarkedBy: 'User',
            viewedBy: 'View',
        },
        joinMap: { labels: 'label', bookmarkedBy: 'user' },
        countFields: {
            commentsCount: true,
            labelsCount: true,
            reportsCount: true,
            translationsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await StarModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isUpvoted: await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                    }
                }
            },
        },
    },
    mutate: {} as any,
    search: {
        defaultSort: IssueSortBy.ScoreDesc,
        sortBy: IssueSortBy,
        searchFields: {
            apiId: true,
            closedById: true,
            createdById: true,
            createdTimeFrame: true,
            minScore: true,
            minBookmarks: true,
            minViews: true,
            noteId: true,
            organizationId: true,
            projectId: true,
            referencedVersionId: true,
            routineId: true,
            smartContractId: true,
            standardId: true,
            status: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({ OR: ['transDescriptionWrapped', 'transNameWrapped'] }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.issueSelect>(data, [
            ['api', 'Api'],
            ['organization', 'Organization'],
            ['note', 'Note'],
            ['project', 'Project'],
            ['routine', 'Routine'],
            ['smartContract', 'SmartContract'],
            ['standard', 'Standard'],
        ], languages),
        isTransferable: false,
        maxObjects: {
            User: {
                private: 0,
                public: 10000,
            },
            Organization: 0,
        },
        owner: (data) => ({
            User: data.createdBy,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            api: 'Api',
            createdBy: 'User',
            organization: 'Organization',
            note: 'Note',
            project: 'Project',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ createdBy: { id: userId } }),
        }
    },
})