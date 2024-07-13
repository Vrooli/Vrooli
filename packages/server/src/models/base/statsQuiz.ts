import { StatsQuizSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsQuizFormat } from "../formats";
import { QuizModelInfo, QuizModelLogic, StatsQuizModelInfo, StatsQuizModelLogic } from "./types";

const __typename = "StatsQuiz" as const;
export const StatsQuizModel: StatsQuizModelLogic = ({
    __typename,
    dbTable: "stats_quiz",
    display: () => ({
        label: {
            select: () => ({ id: true, quiz: { select: ModelMap.get<QuizModelLogic>("Quiz").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<QuizModelLogic>("Quiz").display().label.get(select.quiz as QuizModelInfo["PrismaModel"], languages),
            }),
        },
    }),
    format: StatsQuizFormat,
    search: {
        defaultSort: StatsQuizSortBy.PeriodStartAsc,
        sortBy: StatsQuizSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ quiz: ModelMap.get<QuizModelLogic>("Quiz").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            quiz: "Quiz",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<QuizModelLogic>("Quiz").validate().owner(data?.quiz as QuizModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsQuizModelInfo["PrismaSelect"]>([["quiz", "Quiz"]], ...rest),
        visibility: {
            private: function getVisibilityPrivate(...params) {
                return {
                    quiz: ModelMap.get<QuizModelLogic>("Quiz").validate().visibility.private(...params),
                };
            },
            public: function getVisibilityPublic(...params) {
                return {
                    quiz: ModelMap.get<QuizModelLogic>("Quiz").validate().visibility.public(...params),
                };
            },
            owner: (userId) => ({ quiz: ModelMap.get<QuizModelLogic>("Quiz").validate().visibility.owner(userId) }),
        },
    }),
});
