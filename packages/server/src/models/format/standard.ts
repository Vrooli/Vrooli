import { Formatter } from "../types";

const __typename = "Standard" as const;
export const StandardFormat: Formatter<ModelStandardLogic> = {
    gqlRelMap: {
        __typename,
        createdBy: "User",
        issues: "Issue",
        labels: "Label",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "Project",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        tags: "Tag",
        transfers: "Transfer",
        versions: "StandardVersion",
    },
    prismaRelMap: {
        __typename,
        createdBy: "User",
        ownedByOrganization: "Organization",
        ownedByUser: "User",
        issues: "Issue",
        labels: "Label",
        parent: "StandardVersion",
        tags: "Tag",
        bookmarkedBy: "User",
        versions: "StandardVersion",
        pullRequests: "PullRequest",
        stats: "StatsStandard",
        questions: "Question",
        transfers: "Transfer",
    },
    joinMap: { labels: "label", tags: "tag", bookmarkedBy: "user" },
    countFields: {
        forksCount: true,
        issuesCount: true,
        pullRequestsCount: true,
        questionsCount: true,
        transfersCount: true,
        versionsCount: true,
    },
};
