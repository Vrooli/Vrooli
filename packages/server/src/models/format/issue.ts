import { IssueModelLogic } from "../base/types";
import { Formatter } from "../types";

export const IssueFormat: Formatter<IssueModelLogic> = {
    gqlRelMap: {
        __typename: "Issue",
        closedBy: "User",
        comments: "Comment",
        createdBy: "User",
        labels: "Label",
        reports: "Report",
        bookmarkedBy: "User",
        to: {
            api: "Api",
            organization: "Organization",
            note: "Note",
            project: "Project",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
    },
    prismaRelMap: {
        __typename: "Issue",
        api: "Api",
        organization: "Organization",
        note: "Note",
        project: "Project",
        routine: "Routine",
        smartContract: "SmartContract",
        standard: "Standard",
        closedBy: "User",
        comments: "Comment",
        labels: "Label",
        reports: "Report",
        reactions: "Reaction",
        bookmarkedBy: "User",
        viewedBy: "View",
    },
    joinMap: { labels: "label", bookmarkedBy: "user" },
    countFields: {
        commentsCount: true,
        labelsCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};
