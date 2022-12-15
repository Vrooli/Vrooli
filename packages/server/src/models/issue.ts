import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Issue, IssueCreateInput, IssuePermission, IssueSearchInput, IssueSortBy } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { StarModel } from "./star";
import { Displayer, Formatter, Searcher } from "./types";
import { VoteModel } from "./vote";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: IssueCreateInput,
    GqlModel: Issue,
    GqlSearch: IssueSearchInput,
    GqlSort: IssueSortBy,
    GqlPermission: IssuePermission,
    PrismaCreate: Prisma.issueUpsertArgs['create'],
    PrismaUpdate: Prisma.issueUpsertArgs['update'],
    PrismaModel: Prisma.issueGetPayload<SelectWrap<Prisma.issueSelect>>,
    PrismaSelect: Prisma.issueSelect,
    PrismaWhere: Prisma.issueWhereInput,
}

const __typename = 'Issue' as const;

const suppFields = ['isStarred', 'isUpvoted', 'permissionsIssue'] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    gqlRelMap: {
        __typename,
        closedBy: 'User',
        comments: 'Comment',
        createdBy: 'User',
        labels: 'Label',
        reports: 'Report',
        starredBy: 'User',
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
        starredBy: 'User',
        viewedBy: 'View',
    },
    joinMap: { labels: 'label', starredBy: 'user' },
    countFields: ['commentsCount', 'labelsCount', 'reportsCount', 'translationsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => ({
            isStarred: async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
            isUpvoted: async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
            permissionsIssue: async () => await getSingleTypePermissions(__typename, ids, prisma, userData),
        }),
    },
})

const searcher = (): Searcher<Model> => ({
    defaultSort: IssueSortBy.ScoreDesc,
    sortBy: IssueSortBy,
    searchFields: [
        'apiId',
        'closedById',
        'createdById',
        'createdTimeFrame',
        'minScore',
        'minStars',
        'minViews',
        'noteId',
        'organizationId',
        'projectId',
        'referencedVersionId',
        'routineId',
        'smartContractId',
        'standardId',
        'status',
        'translationLanguages',
        'updatedTimeFrame',
        'visibility',
    ],
    searchStringQuery: () => ({ OR: ['transDescriptionWrapped', 'transNameWrapped'] }),
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const IssueModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.issue,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: searcher(),
    validate: {} as any,
})