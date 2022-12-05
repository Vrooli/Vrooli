import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const QuestionModel = ({
    delegate: (prisma: PrismaType) => prisma.question,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Question' as GraphQLModelType,
    validate: {} as any,
})