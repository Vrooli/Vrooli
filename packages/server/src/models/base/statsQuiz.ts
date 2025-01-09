import { DEFAULT_LANGUAGE, MaxObjects, StatsQuizSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { useVisibility } from "../../builders/visibilityBuilder";
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
                lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE,
                objectName: ModelMap.get<QuizModelLogic>("Quiz").display().label.get(select.quiz as QuizModelInfo["DbModel"], languages),
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
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            quiz: "Quiz",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<QuizModelLogic>("Quiz").validate().owner(data?.quiz as QuizModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsQuizModelInfo["DbSelect"]>([["quiz", "Quiz"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    quiz: useVisibility("Quiz", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    quiz: useVisibility("Quiz", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    quiz: useVisibility("Quiz", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    quiz: useVisibility("Quiz", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    quiz: useVisibility("Quiz", "Public", data),
                };
            },
        },
    }),
});
