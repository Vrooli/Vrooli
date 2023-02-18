import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrependString, QuizQuestionResponse, QuizQuestionResponseCreateInput, QuizQuestionResponseSearchInput, QuizQuestionResponseSortBy, QuizQuestionResponseUpdateInput, QuizQuestionResponseYou } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { QuizQuestionModel } from "./quizQuestion";
import { selPad } from "../builders";

const __typename = 'QuizQuestionResponse' as const;
type Permissions = Pick<QuizQuestionResponseYou, 'canDelete' | 'canUpdate'>;
const suppFields = ['you'] as const;
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
    display: {
        select: () => ({ id: true, quizQuestion: selPad(QuizQuestionModel.display.select) }),
        label: (select, languages) => i18next.t(`common:QuizQuestionResponseLabel`, {
            lng: languages.length > 0 ? languages[0] : 'en',
            questionLabel: QuizQuestionModel.display.label(select.quizQuestion as any, languages),
        }),
    },
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
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    }
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})