import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrependString, Quiz, QuizCreateInput, QuizSearchInput, QuizSortBy, QuizUpdateInput, QuizYou } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { StarModel } from "./star";
import { VoteModel } from "./vote";

const type = 'Quiz' as const;
type Permissions = Pick<QuizYou, 'canDelete' | 'canEdit' | 'canStar' | 'canView' | 'canVote'>;
const suppFields = ['you.canDelete', 'you.canEdit', 'you.canStar', 'you.canView', 'you.canVote', 'you.isStarred', 'you.isUpvoted'] as const;
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
    type,
    delegate: (prisma: PrismaType) => prisma.quiz,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages)
    },
    format: {
        gqlRelMap: {
            type,
            attempts: 'QuizAttempt',
            createdBy: 'User',
            project: 'Project',
            quizQuestions: 'QuizQuestion',
            routine: 'Routine',
            starredBy: 'User',
        },
        prismaRelMap: {
            type,
            attempts: 'QuizAttempt',
            createdBy: 'User',
            project: 'Project',
            quizQuestions: 'QuizQuestion',
            routine: 'Routine',
            starredBy: 'User',
        },
        joinMap: { starredBy: 'user' },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(type, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                    'you.isStarred': await StarModel.query.getIsStarreds(prisma, userData?.id, ids, type),
                    'you.isUpvoted': await VoteModel.query.getIsUpvoteds(prisma, userData?.id, ids, type),
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})