import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.question_answerSelect,
    Prisma.question_answerGetPayload<SelectWrap<Prisma.question_answerSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const QuestionAnswerModel = ({
    delegate: (prisma: PrismaType) => prisma.question_answer,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'QuestionAnswer' as GraphQLModelType,
    validate: {} as any,
})