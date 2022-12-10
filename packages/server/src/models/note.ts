import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Note } from "../endpoints/types";
import { PrismaType } from "../types";
import { getSingleTypePermissions } from "../validators";
import { NoteVersionModel } from "./noteVersion";
import { StarModel } from "./star";
import { Displayer, Formatter } from "./types";
import { ViewModel } from "./view";
import { VoteModel } from "./vote";

const __typename = 'Note' as const;

const suppFields = ['isStarred', 'isViewed', 'isUpvoted', 'permissionsRoot'] as const;
const formatter = (): Formatter<Note, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        createdBy: 'User',
        issues: 'Issue',
        labels: 'Label',
        owner: ['Organization', 'User'],
        parent: 'Note',
        pullRequests: 'PullRequest',
        questions: 'Question',
        stats: 'StatsNote',
        tags: 'Tag',
        transfers: 'Transfer',
        versions: 'NoteVersion',
    },
    joinMap: { labels: 'label', starredBy: 'user', tags: 'tag' },
    countFields: ['issuesCount', 'labelsCount', 'pullRequestsCount', 'questionsCount', 'transfersCount', 'versionsCount'],
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ ids, prisma, userData }) => [
            ['isStarred', async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename)],
            ['isUpvoted', async () => await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename)],
            ['isViewed', async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename)],
            ['permissionsRoot', async () => await getSingleTypePermissions(__typename, ids, prisma, userData)],
        ],
    },
})

const displayer = (): Displayer<
    Prisma.noteSelect,
    Prisma.noteGetPayload<SelectWrap<Prisma.noteSelect>>
> => ({
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
})

export const NoteModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.note,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})