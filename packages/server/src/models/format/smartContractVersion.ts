import { SmartContractVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

export const SmartContractVersionFormat: Formatter<SmartContractVersionModelLogic> = {
    gqlRelMap: {
        __typename: "SmartContractVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "SmartContractVersion",
        pullRequest: "PullRequest",
        reports: "Report",
        root: "SmartContract",
    },
    prismaRelMap: {
        __typename: "SmartContractVersion",
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
