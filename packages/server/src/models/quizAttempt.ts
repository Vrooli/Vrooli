import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrependString, QuizAttempt, QuizAttemptCreateInput, QuizAttemptSearchInput, QuizAttemptSortBy, QuizAttemptUpdateInput, QuizAttemptYou } from '@shared/consts';
import { PrismaType } from "../types";
import { QuizModel } from "./quiz";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";

const type = 'QuizAttempt' as const;
type Permissions = Pick<QuizAttemptYou, 'canDelete' | 'canEdit'>;
const suppFields = ['you.canDelete', 'you.canEdit'] as const;
export const QuizAttemptModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizAttemptCreateInput,
    GqlUpdate: QuizAttemptUpdateInput,
    GqlModel: QuizAttempt,
    GqlSearch: QuizAttemptSearchInput,
    GqlSort: QuizAttemptSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.quiz_attemptUpsertArgs['create'],
    PrismaUpdate: Prisma.quiz_attemptUpsertArgs['update'],
    PrismaModel: Prisma.quiz_attemptGetPayload<SelectWrap<Prisma.quiz_attemptSelect>>,
    PrismaSelect: Prisma.quiz_attemptSelect,
    PrismaWhere: Prisma.quiz_attemptWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.quiz_attempt,
    display: {
        select: () => ({ 
            id: true,
            created_at: true,
            quiz: { select: QuizModel.display.select() },
        }),
        // Label is quiz name + created_at date
        label: (select, languages) => {
            const quizName = QuizModel.display.label(select.quiz as any, languages);
            const date = new Date(select.created_at).toLocaleDateString();
            return `${quizName} - ${date}`;
        }
    },
    format: {
        gqlRelMap: {
            type,
            quiz: 'Quiz',
            responses: 'QuizQuestionResponse',
            user: 'User',
        },
        prismaRelMap: {
            type,
            quiz: 'Quiz',
            responses: 'QuizQuestionResponse',
            user: 'User',
        },
        countFields: {
            responsesCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(type, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})