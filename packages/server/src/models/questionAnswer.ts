import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { QuestionAnswer, QuestionAnswerCreateInput, QuestionAnswerSearchInput, QuestionAnswerSortBy, QuestionAnswerUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'QuestionAnswer' as const;
const suppFields = [] as const;
export const QuestionAnswerModel: ModelLogic<{
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
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question_answer,
    display: {
        select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations as any, 'name', languages)
    },
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})