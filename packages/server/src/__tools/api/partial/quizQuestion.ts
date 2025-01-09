import { QuizQuestion, QuizQuestionTranslation, QuizQuestionYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
