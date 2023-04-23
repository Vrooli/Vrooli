import { StatsQuizSortBy } from "@local/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { QuizModel } from "./quiz";
const __typename = "StatsQuiz";
const suppFields = [];
export const StatsQuizModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_quiz,
    display: {
        select: () => ({ id: true, quiz: selPad(QuizModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: QuizModel.display.label(select.quiz, languages),
        }),
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
        searchStringQuery: () => ({ quiz: QuizModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            quiz: "Quiz",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => QuizModel.validate.owner(data.quiz, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["quiz", "Quiz"],
        ], languages),
        visibility: {
            private: { quiz: QuizModel.validate.visibility.private },
            public: { quiz: QuizModel.validate.visibility.public },
            owner: (userId) => ({ quiz: QuizModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=statsQuiz.js.map