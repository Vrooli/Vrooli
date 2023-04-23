import { rel } from "../utils";
export const quizQuestionTranslation = {
    __typename: "QuizQuestionTranslation",
    common: {
        id: true,
        language: true,
        helpText: true,
        questionText: true,
    },
    full: {},
    list: {},
};
export const quizQuestionYou = {
    __typename: "QuizQuestionYou",
    common: {
        canDelete: true,
        canUpdate: true,
    },
    full: {},
    list: {},
};
export const quizQuestion = {
    __typename: "QuizQuestion",
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        order: true,
        points: true,
        responsesCount: true,
        quiz: async () => rel((await import("./quiz")).quiz, "nav", { omit: "quizQuestions" }),
        standardVersion: async () => rel((await import("./standardVersion")).standardVersion, "nav"),
        you: () => rel(quizQuestionYou, "full"),
    },
    full: {
        responses: async () => rel((await import("./quizQuestionResponse")).quizQuestionResponse, "full", { omit: "quizQuestion" }),
        translations: () => rel(quizQuestionTranslation, "full"),
    },
    list: {
        translations: () => rel(quizQuestionTranslation, "list"),
    },
};
//# sourceMappingURL=quizQuestion.js.map