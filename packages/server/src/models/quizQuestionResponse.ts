import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrependString, QuizQuestionResponse, QuizQuestionResponseCreateInput, QuizQuestionResponseSearchInput, QuizQuestionResponseSortBy, QuizQuestionResponseUpdateInput, QuizQuestionResponseYou } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";

const __typename = 'QuizQuestionResponse' as const;
type Permissions = Pick<QuizQuestionResponseYou, 'canDelete' | 'canUpdate'>;
const suppFields = ['you.canDelete', 'you.canUpdate'] as const;
export const QuizQuestionResponseModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizQuestionResponseCreateInput,
    GqlUpdate: QuizQuestionResponseUpdateInput,
    GqlModel: QuizQuestionResponse,
    GqlSearch: QuizQuestionResponseSearchInput,
    GqlSort: QuizQuestionResponseSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.quiz_question_responseUpsertArgs['create'],
    PrismaUpdate: Prisma.quiz_question_responseUpsertArgs['update'],
    PrismaModel: Prisma.quiz_question_responseGetPayload<SelectWrap<Prisma.quiz_question_responseSelect>>,
    PrismaSelect: Prisma.quiz_question_responseSelect,
    PrismaWhere: Prisma.quiz_question_responseWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_question_response,
    display: {} as any,
    format: {
        gqlRelMap: {
            __typename,
            quizAttempt: 'QuizAttempt',
            quizQuestion: 'QuizQuestion',
        },
        prismaRelMap: {
            __typename,
            quizAttempt: 'QuizAttempt',
            quizQuestion: 'QuizQuestion',
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
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