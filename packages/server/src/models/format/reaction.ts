import { ReactionModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Reaction" as const;
export const ReactionFormat: Formatter<ReactionModelLogic> = {
    gqlRelMap: {
        __typename,
        by: "User",
        to: {
            api: "Api",
            chatMessage: "ChatMessage",
            comment: "Comment",
            issue: "Issue",
            note: "Note",
            post: "Post",
            project: "Project",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            quiz: "Quiz",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
    },
    prismaRelMap: {
        __typename,
        by: "User",
        api: "Api",
        chatMessage: "ChatMessage",
        comment: "Comment",
        issue: "Issue",
        note: "Note",
        post: "Post",
        project: "Project",
        question: "Question",
        questionAnswer: "QuestionAnswer",
        quiz: "Quiz",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
    },
    countFields: {},
};
