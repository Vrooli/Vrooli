import { BookmarkModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Bookmark" as const;
export const BookmarkFormat: Formatter<BookmarkModelLogic> = {
    gqlRelMap: {
        __typename,
        by: "User",
        list: "BookmarkList",
        to: {
            api: "Api",
            comment: "Comment",
            issue: "Issue",
            note: "Note",
            organization: "Organization",
            post: "Post",
            project: "Project",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            quiz: "Quiz",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
            tag: "Tag",
            user: "User",
        },
    },
    prismaRelMap: {
        __typename,
        api: "Api",
        comment: "Comment",
        issue: "Issue",
        list: "BookmarkList",
        note: "Note",
        organization: "Organization",
        post: "Post",
        project: "Project",
        question: "Question",
        questionAnswer: "QuestionAnswer",
        quiz: "Quiz",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
        tag: "Tag",
        user: "User",
    },
    countFields: {},
};