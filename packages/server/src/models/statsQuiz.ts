import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'StatsQuiz' as const;
const suppFields = [] as const;
export const StatsQuizModel: ModelLogic<any, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.stats_quiz,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})