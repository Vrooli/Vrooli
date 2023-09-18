import { StandardVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StandardVersionFormat: Formatter<StandardVersionModelLogic> = {
    gqlRelMap: {
        __typename: "StandardVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "StandardVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "Standard",
    },
    prismaRelMap: {
        __typename: "StandardVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "StandardVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "Standard",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};
