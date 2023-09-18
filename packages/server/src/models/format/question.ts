import { QuestionModelLogic } from "../base/types";
import { Formatter } from "../types";

export const QuestionFormat: Formatter<QuestionModelLogic> = {
    gqlRelMap: {
        __typename: "Question",
        createdBy: "User",
        answers: "QuestionAnswer",
        comments: "Comment",
        forObject: {
            api: "Api",
            note: "Note",
            organization: "Organization",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
        reports: "Report",
        bookmarkedBy: "User",
        tags: "Tag",
    },
    prismaRelMap: {
        __typename: "Question",
        createdBy: "User",
        api: "Api",
        note: "Note",
        organization: "Organization",
        project: "Project",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
        comments: "Comment",
        answers: "QuestionAnswer",
        reports: "Report",
        tags: "Tag",
        bookmarkedBy: "User",
        reactions: "Reaction",
        viewedBy: "User",
    },
    joinMap: { bookmarkedBy: "user", tags: "tag" },
    countFields: {
        answersCount: true,
        commentsCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};
