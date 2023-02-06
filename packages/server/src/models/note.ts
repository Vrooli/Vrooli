import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Note, NoteCreateInput, NoteSearchInput, NoteSortBy, NoteUpdateInput, NoteYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { NoteVersionModel } from "./noteVersion";
import { StarModel } from "./star";
import { ModelLogic } from "./types";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";

const __typename = 'Note' as const;
type Permissions = Pick<NoteYou, 'canDelete' | 'canUpdate' | 'canStar' | 'canTransfer' | 'canRead' | 'canVote'>;
const suppFields = ['you.canDelete', 'you.canUpdate', 'you.canStar', 'you.canTransfer', 'you.canRead', 'you.canVote', 'you.isStarred', 'you.isUpvoted', 'you.isViewed'] as const;
export const NoteModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: NoteCreateInput,
    GqlUpdate: NoteUpdateInput,
    GqlModel: Note,
    GqlSearch: NoteSearchInput,
    GqlSort: NoteSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.noteUpsertArgs['create'],
    PrismaUpdate: Prisma.noteUpsertArgs['update'],
    PrismaModel: Prisma.noteGetPayload<SelectWrap<Prisma.noteSelect>>,
    PrismaSelect: Prisma.noteSelect,
    PrismaWhere: Prisma.noteWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.note,
    display: {
        select: () => ({
            id: true,
            versions: {
                orderBy: { versionIndex: 'desc' },
                take: 1,
                select: NoteVersionModel.display.select(),
            }
        }),
        label: (select, languages) => select.versions.length > 0 ?
            NoteVersionModel.display.label(select.versions[0] as any, languages) : '',
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
            parent: 'Note',
            pullRequests: 'PullRequest',
            questions: 'Question',
            starredBy: 'User',
            tags: 'Tag',
            transfers: 'Transfer',
            versions: 'NoteVersion',
        },
        prismaRelMap: {
            __typename,
            parent: 'NoteVersion',
            createdBy: 'User',
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
            versions: 'NoteVersion',
            pullRequests: 'PullRequest',
            labels: 'Label',
            issues: 'Issue',
            tags: 'Tag',
            starredBy: 'User',
            questions: 'Question',
        },
        joinMap: { labels: 'label', starredBy: 'user', tags: 'tag' },
        countFields: {
            issuesCount: true,
            labelsCount: true,
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
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})