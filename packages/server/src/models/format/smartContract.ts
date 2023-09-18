import { SmartContractModelLogic } from "../base/types";
import { Formatter } from "../types";

export const SmartContractFormat: Formatter<SmartContractModelLogic> = {
    gqlRelMap: {
        __typename: "SmartContract",
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "SmartContract",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "NoteVersion",
    },
    prismaRelMap: {
        __typename: "SmartContract",
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        parent: "NoteVersion",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "NoteVersion",
    },
    joinMap: { labels: "label", bookmarkedBy: "user", tags: "tag" },
    countFields: {
        issuesCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};
