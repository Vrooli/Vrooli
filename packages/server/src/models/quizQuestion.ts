import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { QuizQuestion, QuizQuestionCreateInput, QuizQuestionYou, QuizQuestionSearchInput, QuizQuestionSortBy, QuizQuestionUpdateInput, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { quizQuestionValidation } from "@shared/validation";

const __typename = 'QuizQuestion' as const;
type Permissions = Pick<QuizQuestionYou, 'canDelete' | 'canUpdate'>;
const suppFields = ['you'] as const;
export const QuizQuestionModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizQuestionCreateInput,
    GqlUpdate: QuizQuestionUpdateInput,
    GqlModel: QuizQuestion,
    GqlSearch: QuizQuestionSearchInput,
    GqlSort: QuizQuestionSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.quiz_questionUpsertArgs['create'],
    PrismaUpdate: Prisma.quiz_questionUpsertArgs['update'],
    PrismaModel: Prisma.quiz_questionGetPayload<SelectWrap<Prisma.quiz_questionSelect>>,
    PrismaSelect: Prisma.quiz_questionSelect,
    PrismaWhere: Prisma.quiz_questionWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_question,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, questionText: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'questionText', languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            quiz: 'Quiz',
            responses: 'QuizQuestionResponse',
            standardVersion: 'StandardVersion',
        },
        prismaRelMap: {
            __typename,
            quiz: 'Quiz',
            responses: 'QuizQuestionResponse',
            standardVersion: 'StandardVersion',
        },
        countFields: {
            responsesCount: true,
        },
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
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any),
            update: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any)
        },
        yup: quizQuestionValidation,
    },
    search: {
        defaultSort: QuizQuestionSortBy.OrderAsc,
        sortBy: QuizQuestionSortBy,
        searchFields: {
            createdTimeFrame: true,
            translationLanguages: true,
            quizId: true,
            standardId: true,
            userId: true,
            responseId: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transQuestionTextWrapped',
            ]
        }),
    },
    validate: {} as any,
})