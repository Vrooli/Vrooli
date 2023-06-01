import { StatsQuizSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../builders";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { QuizModel } from "./quiz";
import { ModelLogic, StatsQuizModelLogic } from "./types";

const __typename = "StatsQuiz" as const;
const suppFields = [] as const;
export const StatsQuizModel: ModelLogic<StatsQuizModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_quiz,
    display: {
        label: {
            select: () => ({ id: true, quiz: selPad(QuizModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: QuizModel.display.label.get(select.quiz as any, languages),
            }),
        },
    },
    format: {
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
});
