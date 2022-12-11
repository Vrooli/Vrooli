import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { Displayer } from "./types";

const __typename = 'QuizQuestionResponse' as const;

const suppFields = [] as const;

// Doesn't make sense to have a displayer for this model
const displayer = (): Displayer<
    Prisma.quiz_question_responseSelect,
    Prisma.quiz_question_responseGetPayload<SelectWrap<Prisma.quiz_question_responseSelect>>
> => ({
    select: () => ({ id: true }),
    label: () => '',
})

export const QuizQuestionResponseModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.quiz_question_response,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})