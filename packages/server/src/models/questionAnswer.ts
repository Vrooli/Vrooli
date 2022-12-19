import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerSortBy, QuestionAnswerUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuestionAnswerCreateInput,
    GqlUpdate: QuestionAnswerUpdateInput,
    GqlModel: QuestionAnswer,
    GqlSearch: QuestionAnswerSearchInput,
    GqlSort: QuestionAnswerSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.question_answerUpsertArgs['create'],
    PrismaUpdate: Prisma.question_answerUpsertArgs['update'],
    PrismaModel: Prisma.question_answerGetPayload<SelectWrap<Prisma.question_answerSelect>>,
    PrismaSelect: Prisma.question_answerSelect,
    PrismaWhere: Prisma.question_answerWhereInput,
}

const __typename = 'QuestionAnswer' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations as any, 'name', languages)
})

export const QuestionAnswerModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question_answer,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})