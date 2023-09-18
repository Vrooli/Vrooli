
import { ApiModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ApiFormat: Formatter<ApiModelLogic> = {
    gqlRelMap: {
        __typename: "Api",
        createdBy: "User",
        owner: {
            ownedByUser: "User",
            ownedByOrganization: "Organization",
        },
        parent: "Api",
        tags: "Tag",
        versions: "ApiVersion",
        labels: "Label",
        issues: "Issue",
        pullRequests: "PullRequest",
        questions: "Question",
        bookmarkedBy: "User",
        stats: "StatsApi",
        transfers: "Transfer",
    },
    unionFields: {
        owner: {},
    },
    prismaRelMap: {
        __typename: "Api",
        createdBy: "User",
        ownedByUser: "User",
        ownedByOrganization: "Organization",
        parent: "ApiVersion",
        tags: "Tag",
        issues: "Issue",
        bookmarkedBy: "User",
        reactions: "Reaction",
        viewedBy: "View",
        pullRequests: "PullRequest",
        versions: "ApiVersion",
        labels: "Label",
        stats: "StatsApi",
        questions: "Question",
        transfers: "Transfer",
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
