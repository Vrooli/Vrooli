import { Question, QuestionTranslation, QuestionYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const questionTranslation: GqlPartial<QuestionTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
        name: true,
    },
};

export const questionYou: GqlPartial<QuestionYou> = {
    common: {
        reaction: true,
    },
};

export const question: GqlPartial<Question> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import("./user")).user, "nav"),
        hasAcceptedAnswer: true,
        isPrivate: true,
        score: true,
        bookmarks: true,
        answersCount: true,
        commentsCount: true,
        reportsCount: true,
        forObject: {
            __union: {
                Api: async () => rel((await import("./api")).api, "nav"),
                Code: async () => rel((await import("./code")).code, "nav"),
                Note: async () => rel((await import("./note")).note, "nav"),
                Project: async () => rel((await import("./project")).project, "nav"),
                Routine: async () => rel((await import("./routine")).routine, "nav"),
                Standard: async () => rel((await import("./standard")).standard, "nav"),
                Team: async () => rel((await import("./team")).team, "nav"),
            },
        },
        tags: async () => rel((await import("./tag")).tag, "list"),
        you: () => rel(questionYou, "full"),
    },
    full: {
        answers: async () => rel((await import("./questionAnswer")).questionAnswer, "full", { omit: "question" }),
        translations: () => rel(questionTranslation, "full"),
    },
    list: {
        translations: () => rel(questionTranslation, "list"),
    },
};
