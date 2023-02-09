import { Prisma } from "@prisma/client";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { QuizModel } from "./quiz";
import { ModelLogic } from "./types";

const __typename = 'StatsQuiz' as const;
const suppFields = [] as const;
export const StatsQuizModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_quiz,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            quiz: 'Quiz',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => QuizModel.validate!.owner(data.quiz as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_quizSelect>(data, [
            ['quiz', 'Quiz'],
        ], languages),
        visibility: {
            private: { quiz: QuizModel.validate!.visibility.private },
            public: { quiz: QuizModel.validate!.visibility.public },
            owner: (userId) => ({ quiz: QuizModel.validate!.visibility.owner(userId) }),
        }
    },
})