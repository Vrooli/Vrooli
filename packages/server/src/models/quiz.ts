import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Quiz, QuizCreateInput, QuizPermission, QuizSearchInput, QuizSortBy, QuizUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizCreateInput,
    GqlUpdate: QuizUpdateInput,
    GqlModel: Quiz,
    GqlSearch: QuizSearchInput,
    GqlSort: QuizSortBy,
    GqlPermission: QuizPermission,
    PrismaCreate: Prisma.quizUpsertArgs['create'],
    PrismaUpdate: Prisma.quizUpsertArgs['update'],
    PrismaModel: Prisma.quizGetPayload<SelectWrap<Prisma.quizSelect>>,
    PrismaSelect: Prisma.quizSelect,
    PrismaWhere: Prisma.quizWhereInput,
}

const __typename = 'Quiz' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const QuizModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})