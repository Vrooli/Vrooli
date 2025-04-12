import { QuestionAnswer, QuestionAnswerTranslation } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const questionAnswerTranslation: ApiPartial<QuestionAnswerTranslation> = {
    common: {
        id: true,
        language: true,
        text: true,
    },
};

export const questionAnswer: ApiPartial<QuestionAnswer> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import("./user.js")).user, "nav"),
        score: true,
        bookmarks: true,
        isAccepted: true,
        commentsCount: true,
    },
    full: {
        comments: async () => rel((await import("./comment.js")).comment, "full", { omit: "commentedOn" }),
        question: async () => rel((await import("./question.js")).question, "common", { omit: "answers" }),
        translations: () => rel(questionAnswerTranslation, "full"),
    },
    list: {
        translations: () => rel(questionAnswerTranslation, "list"),
    },
};
