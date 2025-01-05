import { Quiz, QuizTranslation, QuizYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const quizTranslation: GqlPartial<QuizTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const quizYou: GqlPartial<QuizYou> = {
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

export const quiz: GqlPartial<Quiz> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import("./user")).user, "nav"),
        score: true,
        bookmarks: true,
        attemptsCount: true,
        quizQuestionsCount: true,
        project: async () => rel((await import("./project")).project, "nav"),
        routine: async () => rel((await import("./routine")).routine, "nav"),
        you: async () => rel((await import("./quiz")).quizYou, "full"),
    },
    full: {
        quizQuestions: async () => rel((await import("./quizQuestion")).quizQuestion, "full", { omit: "quiz" }),
        translations: () => rel(quizTranslation, "full"),
    },
    list: {
        translations: () => rel(quizTranslation, "list"),
    },
};
