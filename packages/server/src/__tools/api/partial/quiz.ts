import { Quiz, QuizTranslation, QuizYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const quizTranslation: ApiPartial<QuizTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const quizYou: ApiPartial<QuizYou> = {
    common: {
        canDelete: true,
        canBookmark: true,
        canUpdate: true,
        canRead: true,
        canReact: true,
        hasCompleted: true,
        isBookmarked: true,
        reaction: true,
    },
};

export const quiz: ApiPartial<Quiz> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import("./user.js")).user, "nav"),
        score: true,
        bookmarks: true,
        attemptsCount: true,
        quizQuestionsCount: true,
        project: async () => rel((await import("./project.js")).project, "nav"),
        routine: async () => rel((await import("./routine.js")).routine, "nav"),
        you: async () => rel((await import("./quiz.js")).quizYou, "full"),
    },
    full: {
        quizQuestions: async () => rel((await import("./quizQuestion.js")).quizQuestion, "full", { omit: "quiz" }),
        translations: () => rel(quizTranslation, "full"),
    },
    list: {
        translations: () => rel(quizTranslation, "list"),
    },
};
