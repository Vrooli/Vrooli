import { SmartContractVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "SmartContractVersion" as const;
export const SmartContractVersionFormat: Formatter<SmartContractVersionModelLogic> = {
    gqlRelMap: {
        __typename,
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "SmartContractVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "SmartContract",
    },
    prismaRelMap: {
        __typename,
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "SmartContractVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "SmartContract",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        reportsCount: true,
        translationsCount: true,
    },
};
