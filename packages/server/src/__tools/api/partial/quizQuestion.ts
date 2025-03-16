import { QuizQuestion, QuizQuestionTranslation, QuizQuestionYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const quizQuestionTranslation: ApiPartial<QuizQuestionTranslation> = {
    common: {
        id: true,
        language: true,
        helpText: true,
        questionText: true,
    },
};

export const quizQuestionYou: ApiPartial<QuizQuestionYou> = {
    common: {
        canDelete: true,
        canUpdate: true,
    },
};

export const quizQuestion: ApiPartial<QuizQuestion> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        order: true,
        points: true,
        responsesCount: true,
        quiz: async () => rel((await import("./quiz.js")).quiz, "nav", { omit: "quizQuestions" }),
        standardVersion: async () => rel((await import("./standardVersion.js")).standardVersion, "nav"),
        you: () => rel(quizQuestionYou, "full"),
    },
    full: {
        responses: async () => rel((await import("./quizQuestionResponse.js")).quizQuestionResponse, "full", { omit: "quizQuestion" }),
        translations: () => rel(quizQuestionTranslation, "full"),
    },
    list: {
        translations: () => rel(quizQuestionTranslation, "list"),
    },
};
