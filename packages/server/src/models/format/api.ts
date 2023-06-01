import { ModelApiLogic } from "../api";
import { Formatter } from "../types";

// TODO for morning: figure out how to split formatters from modellogic correctly. Then split all, and create a FormatMap to use in gqlSelect.ts script instead of ObjectMap. Hopefully this won't load the full server like it's doing now. If this works, then we might be able to generate PartialGraphQLInfo objects instead of GraphQLResolveInfo objects, without having to worry aobut circular dependencies (since REST endpoints use these generated types, so if the whole server is loaded during the script, we have a problem)
const __typename = "Api" as const;
export const ApiFormat: Formatter<ModelApiLogic> = {
    gqlRelMap: {
        __typename,
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
    prismaRelMap: {
        __typename,
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
