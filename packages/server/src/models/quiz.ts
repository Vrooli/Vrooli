import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrependString, Quiz, QuizCreateInput, QuizSearchInput, QuizSortBy, QuizUpdateInput, QuizYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { StarModel } from "./star";
import { VoteModel } from "./vote";

const __typename = 'Quiz' as const;
type Permissions = Pick<QuizYou, 'canDelete' | 'canUpdate' | 'canStar' | 'canRead' | 'canVote'>;
const suppFields = ['you.canDelete', 'you.canUpdate', 'you.canStar', 'you.canRead', 'you.canVote', 'you.isStarred', 'you.isUpvoted'] as const;
export const QuizModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizCreateInput,
    GqlUpdate: QuizUpdateInput,
    GqlModel: Quiz,
    GqlSearch: QuizSearchInput,
    GqlSort: QuizSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.quizUpsertArgs['create'],
    PrismaUpdate: Prisma.quizUpsertArgs['update'],
    PrismaModel: Prisma.quizGetPayload<SelectWrap<Prisma.quizSelect>>,
    PrismaSelect: Prisma.quizSelect,
    PrismaWhere: Prisma.quizWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages)
    },
    format: {
        gqlRelMap: {
            __typename,
            attempts: 'QuizAttempt',
            createdBy: 'User',
            project: 'Project',
            quizQuestions: 'QuizQuestion',
            routine: 'Routine',
            starredBy: 'User',
        },
        prismaRelMap: {
            __typename,
            attempts: 'QuizAttempt',
            createdBy: 'User',
            project: 'Project',
            quizQuestions: 'QuizQuestion',
            routine: 'Routine',
            starredBy: 'User',
        },
        joinMap: { starredBy: 'user' },
        countFields: {
            attemptsCount: true,
            quizQuestionsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                    'you.isStarred': await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
                    'you.isUpvoted': await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, __typename),
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})