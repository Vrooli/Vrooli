import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { QuizQuestion, QuizQuestionCreateInput, QuizQuestionPermission, QuizQuestionSearchInput, QuizQuestionSortBy, QuizQuestionUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizQuestionCreateInput,
    GqlUpdate: QuizQuestionUpdateInput,
    GqlModel: QuizQuestion,
    GqlSearch: QuizQuestionSearchInput,
    GqlSort: QuizQuestionSortBy,
    GqlPermission: QuizQuestionPermission,
    PrismaCreate: Prisma.quiz_questionUpsertArgs['create'],
    PrismaUpdate: Prisma.quiz_questionUpsertArgs['update'],
    PrismaModel: Prisma.quiz_questionGetPayload<SelectWrap<Prisma.quiz_questionSelect>>,
    PrismaSelect: Prisma.quiz_questionSelect,
    PrismaWhere: Prisma.quiz_questionWhereInput,
}

const __typename = 'QuizQuestion' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, questionText: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'questionText', languages),
})

export const QuizQuestionModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_question,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})