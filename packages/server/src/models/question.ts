import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Question, QuestionCreateInput, QuestionSearchInput, QuestionSortBy, QuestionUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuestionCreateInput,
    GqlUpdate: QuestionUpdateInput,
    GqlModel: Question,
    GqlSearch: QuestionSearchInput,
    GqlSort: QuestionSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.questionUpsertArgs['create'],
    PrismaUpdate: Prisma.questionUpsertArgs['update'],
    PrismaModel: Prisma.questionGetPayload<SelectWrap<Prisma.questionSelect>>,
    PrismaSelect: Prisma.questionSelect,
    PrismaWhere: Prisma.questionWhereInput,
}

const __typename = 'Question' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const QuestionModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})