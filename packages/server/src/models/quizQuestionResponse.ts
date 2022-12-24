import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { QuizQuestionResponse, QuizQuestionResponseCreateInput, QuizQuestionResponsePermission, QuizQuestionResponseSearchInput, QuizQuestionResponseSortBy, QuizQuestionResponseUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizQuestionResponseCreateInput,
    GqlUpdate: QuizQuestionResponseUpdateInput,
    GqlModel: QuizQuestionResponse,
    GqlSearch: QuizQuestionResponseSearchInput,
    GqlSort: QuizQuestionResponseSortBy,
    GqlPermission: QuizQuestionResponsePermission,
    PrismaCreate: Prisma.quiz_question_responseUpsertArgs['create'],
    PrismaUpdate: Prisma.quiz_question_responseUpsertArgs['update'],
    PrismaModel: Prisma.quiz_question_responseGetPayload<SelectWrap<Prisma.quiz_question_responseSelect>>,
    PrismaSelect: Prisma.quiz_question_responseSelect,
    PrismaWhere: Prisma.quiz_question_responseWhereInput,
}

const __typename = 'QuizQuestionResponse' as const;

const suppFields = [] as const;

// TODO
const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true }),
    label: () => '',
})

export const QuizQuestionResponseModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_question_response,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})