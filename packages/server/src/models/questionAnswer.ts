import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const QuestionAnswerModel = ({
    delegate: (prisma: PrismaType) => prisma.question_answer,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'QuestionAnswer' as GraphQLModelType,
    validate: {} as any,
})