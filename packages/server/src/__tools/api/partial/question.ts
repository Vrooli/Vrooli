import { Question, QuestionTranslation, QuestionYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const questionTranslation: ApiPartial<QuestionTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const questionYou: ApiPartial<QuestionYou> = {
    common: {
        reaction: true,
    },
};

export const question: ApiPartial<Question> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import("./user.js")).user, "nav"),
        hasAcceptedAnswer: true,
        isPrivate: true,
        score: true,
        bookmarks: true,
        answersCount: true,
        commentsCount: true,
        reportsCount: true,
        forObject: {
            __union: {
                Api: async () => rel((await import("./api.js")).api, "nav"),
                Code: async () => rel((await import("./code.js")).code, "nav"),
                Note: async () => rel((await import("./note.js")).note, "nav"),
                Project: async () => rel((await import("./project.js")).project, "nav"),
                Routine: async () => rel((await import("./routine.js")).routine, "nav"),
                Standard: async () => rel((await import("./standard.js")).standard, "nav"),
                Team: async () => rel((await import("./team.js")).team, "nav"),
            },
        },
        tags: async () => rel((await import("./tag.js")).tag, "list"),
        you: () => rel(questionYou, "full"),
    },
    full: {
        answers: async () => rel((await import("./questionAnswer.js")).questionAnswer, "full", { omit: "question" }),
        translations: () => rel(questionTranslation, "full"),
    },
    list: {
        translations: () => rel(questionTranslation, "list"),
    },
};
