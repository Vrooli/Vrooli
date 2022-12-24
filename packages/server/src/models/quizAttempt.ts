import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { QuizAttempt, QuizAttemptCreateInput, QuizAttemptPermission, QuizAttemptSearchInput, QuizAttemptSortBy, QuizAttemptUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { QuizModel } from "./quiz";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: QuizAttemptCreateInput,
    GqlUpdate: QuizAttemptUpdateInput,
    GqlModel: QuizAttempt,
    GqlSearch: QuizAttemptSearchInput,
    GqlSort: QuizAttemptSortBy,
    GqlPermission: QuizAttemptPermission,
    PrismaCreate: Prisma.quiz_attemptUpsertArgs['create'],
    PrismaUpdate: Prisma.quiz_attemptUpsertArgs['update'],
    PrismaModel: Prisma.quiz_attemptGetPayload<SelectWrap<Prisma.quiz_attemptSelect>>,
    PrismaSelect: Prisma.quiz_attemptSelect,
    PrismaWhere: Prisma.quiz_attemptWhereInput,
}

const __typename = 'QuizAttempt' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
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
})

export const QuizAttemptModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_attempt,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})