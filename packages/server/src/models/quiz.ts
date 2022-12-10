import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

const __typename = 'Quiz' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.quizSelect,
    Prisma.quizGetPayload<SelectWrap<Prisma.quizSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const QuizModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})