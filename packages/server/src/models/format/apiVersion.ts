import { ApiVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ApiVersion" as const;
export const ApiVersionFormat: Formatter<ApiVersionModelLogic> = {
    gqlRelMap: {
        __typename,
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "ApiVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        resourceList: "ResourceList",
        root: "Api",
    },
    prismaRelMap: {
        __typename,
        calledByRoutineVersions: "RoutineVersion",
        comments: "Comment",
        reports: "Report",
        root: "Api",
        forks: "Api",
        resourceList: "ResourceList",
        pullRequest: "PullRequest",
        directoryListings: "ProjectVersionDirectory",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
    },
};
