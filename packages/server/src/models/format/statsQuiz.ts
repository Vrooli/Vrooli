import { StatsQuiz, StatsQuizSearchInput, StatsQuizSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { QuizModel } from "./quiz";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "StatsQuiz" as const;
export const StatsQuizFormat: Formatter<ModelStatsQuizLogic> = {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            quiz: "Quiz",
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsQuizSortBy.PeriodStartAsc,
        sortBy: StatsQuizSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ quiz: QuizModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            quiz: "Quiz",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => QuizModel.validate!.owner(data.quiz as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_quizSelect>(data, [
            ["quiz", "Quiz"],
        ], languages),
        visibility: {
            private: { quiz: QuizModel.validate!.visibility.private },
            public: { quiz: QuizModel.validate!.visibility.public },
            owner: (userId) => ({ quiz: QuizModel.validate!.visibility.owner(userId) }),
        },
    },
};
