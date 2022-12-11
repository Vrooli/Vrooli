import { Prisma } from "@prisma/client";
import { searchStringBuilder } from "../builders";
import { SelectWrap } from "../builders/types";
import { Issue, IssueSearchInput, IssueSortBy } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { StarModel } from "./star";
import { Displayer, Formatter, Searcher } from "./types";
import { VoteModel } from "./vote";

const __typename = 'Issue' as const;

const suppFields = ['isStarred', 'isUpvoted', 'permissionsIssue'] as const;
const formatter = (): Formatter<Issue, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        closedBy: 'User',
        comments: 'Comment',
        createdBy: 'User',
        labels: 'Label',
        reports: 'Report',
        starredBy: 'User',
        to: ['Api', 'Organization', 'Note', 'Project', 'Routine', 'SmartContract', 'Standard'],
    },
    joinMap: { labels: 'label', starredBy: 'user' },
    countFields: ['commentsCount', 'labelsCount', 'reportsCount', 'translationsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename)],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename)],
            ['permissionsIssue', async () => await getSingleTypePermissions(__typename, ids, prisma, userData)],
        ],
    },
})

const searcher = (): Searcher<
    IssueSearchInput,
    IssueSortBy,
    Prisma.issueWhereInput
> => ({
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
    searchStringQuery: (params) => ({
        OR: searchStringBuilder(['translationsDescription', 'translationsName'], params),
    }),
})

const displayer = (): Displayer<
    Prisma.issueSelect,
    Prisma.issueGetPayload<SelectWrap<Prisma.issueSelect>>
> => ({
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