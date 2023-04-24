import { QuestionAnswer, QuestionAnswerTranslation } from ":/consts";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const questionAnswerTranslation: GqlPartial<QuestionAnswerTranslation> = {
    __typename: "QuestionAnswerTranslation",
    common: {
        id: true,
        language: true,
        text: true,
    },
    full: {},
    list: {},
};

export const questionAnswer: GqlPartial<QuestionAnswer> = {
    __typename: "QuestionAnswer",
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
        question: async () => rel((await import("./question")).question, "nav", { omit: "answers" }),
        translations: () => rel(questionAnswerTranslation, "full"),
    },
    list: {
        translations: () => rel(questionAnswerTranslation, "list"),
    },
};
