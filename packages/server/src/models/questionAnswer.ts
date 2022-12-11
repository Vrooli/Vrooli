import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

const __typename = 'QuestionAnswer' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.question_answerSelect,
    Prisma.question_answerGetPayload<SelectWrap<Prisma.question_answerSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations as any, 'name', languages)
})

export const QuestionAnswerModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.question_answer,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})