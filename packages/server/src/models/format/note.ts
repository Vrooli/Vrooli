import { NoteModelLogic } from "../base/types";
import { Formatter } from "../types";

export const NoteFormat: Formatter<NoteModelLogic> = {
    gqlRelMap: {
        __typename: "Note",
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "Note",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "NoteVersion",
    },
    prismaRelMap: {
        __typename: "Note",
        parent: "NoteVersion",
        createdBy: "User",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        versions: "NoteVersion",
        pullRequests: "PullRequest",
        labels: "Label",
        issues: "Issue",
        tags: "Tag",
        bookmarkedBy: "User",
        questions: "Question",
    },
    joinMap: { labels: "label", bookmarkedBy: "user", tags: "tag" },
    countFields: {
        issuesCount: true,
        labelsCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};
