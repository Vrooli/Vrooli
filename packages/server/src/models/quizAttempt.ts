import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { QuizModel } from "./quiz";
import { Displayer } from "./types";

const __typename = 'QuizAttempt' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.quiz_attemptSelect,
    Prisma.quiz_attemptGetPayload<SelectWrap<Prisma.quiz_attemptSelect>>
> => ({
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

export const QuizAttemptModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_attempt,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})