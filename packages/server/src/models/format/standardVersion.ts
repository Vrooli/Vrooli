import { StandardVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StandardVersion" as const;
export const StandardVersionFormat: Formatter<StandardVersionModelLogic> = {
    gqlRelMap: {
        __typename,
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "StandardVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "Standard",
    },
    prismaRelMap: {
        __typename,
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
