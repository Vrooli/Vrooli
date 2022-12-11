import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

const __typename = 'QuizQuestion' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.quiz_questionSelect,
    Prisma.quiz_questionGetPayload<SelectWrap<Prisma.quiz_questionSelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, questionText: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'questionText', languages),
})

export const QuizQuestionModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_question,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})