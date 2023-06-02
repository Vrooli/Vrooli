import { QuestionAnswerModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "QuestionAnswer" as const;
export const QuestionAnswerFormat: Formatter<QuestionAnswerModelLogic> = {
    gqlRelMap: {
        __typename,
        bookmarkedBy: "User",
        createdBy: "User",
        comments: "Comment",
        question: "Question",
    },
    prismaRelMap: {
        __typename,
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
