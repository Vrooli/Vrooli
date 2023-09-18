import { QuestionAnswerModelLogic } from "../base/types";
import { Formatter } from "../types";

export const QuestionAnswerFormat: Formatter<QuestionAnswerModelLogic> = {
    gqlRelMap: {
        __typename: "QuestionAnswer",
        bookmarkedBy: "User",
        createdBy: "User",
        comments: "Comment",
        question: "Question",
    },
    prismaRelMap: {
        __typename: "QuestionAnswer",
        bookmarkedBy: "User",
        createdBy: "User",
        comments: "Comment",
        question: "Question",
        reactions: "Reaction",
    },
    countFields: {
        commentsCount: true,
    },
    joinMap: { bookmarkedBy: "user" },
};
