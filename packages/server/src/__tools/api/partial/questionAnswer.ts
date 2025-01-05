import { QuestionAnswer, QuestionAnswerTranslation } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const questionAnswerTranslation: GqlPartial<QuestionAnswerTranslation> = {
    common: {
        id: true,
        language: true,
        text: true,
    },
};

export const questionAnswer: GqlPartial<QuestionAnswer> = {
    common: {
        id: true,
        created_at: true,
        updated_at: true,
        createdBy: async () => rel((await import("./user")).user, "nav"),
        score: true,
        bookmarks: true,
        isAccepted: true,
        commentsCount: true,
    },
    full: {
        comments: async () => rel((await import("./comment")).comment, "full", { omit: "commentedOn" }),
        question: async () => rel((await import("./question")).question, "common", { omit: "answers" }),
        translations: () => rel(questionAnswerTranslation, "full"),
    },
    list: {
        translations: () => rel(questionAnswerTranslation, "list"),
    },
};
