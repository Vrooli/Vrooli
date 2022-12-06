import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const QuizModel = ({
    delegate: (prisma: PrismaType) => prisma.quiz,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Quiz' as GraphQLModelType,
    validate: {} as any,
})