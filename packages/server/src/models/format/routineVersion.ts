import { RoutineVersionModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "RoutineVersion" as const;
export const RoutineVersionFormat: Formatter<RoutineVersionModelLogic> = {
    gqlRelMap: {
        __typename,
        apiVersion: "ApiVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        forks: "Routine",
        inputs: "RoutineVersionInput",
        nodes: "Node",
        outputs: "RoutineVersionOutput",
        pullRequest: "PullRequest",
        resourceList: "ResourceList",
        reports: "Report",
        root: "Routine",
        smartContractVersion: "SmartContractVersion",
        suggestedNextByRoutineVersion: "RoutineVersion",
        // you.runs: 'RunRoutine', //TODO
    },
    prismaRelMap: {
        __typename,
        apiVersion: "ApiVersion",
        comments: "Comment",
        directoryListings: "ProjectVersionDirectory",
        reports: "Report",
        smartContractVersion: "SmartContractVersion",
        nodes: "Node",
        nodeLinks: "NodeLink",
        resourceList: "ResourceList",
        root: "Routine",
        forks: "Routine",
        inputs: "RoutineVersionInput",
        outputs: "RoutineVersionOutput",
        pullRequest: "PullRequest",
        runRoutines: "RunRoutine",
        runSteps: "RunRoutineStep",
        suggestedNextByRoutineVersion: "RoutineVersion",
    },
    joinMap: {
        suggestedNextByRoutineVersion: "toRoutineVersion",
    },
    countFields: {
        commentsCount: true,
        directoryListingsCount: true,
        forksCount: true,
        inputsCount: true,
        nodeLinksCount: true,
        nodesCount: true,
        outputsCount: true,
        reportsCount: true,
        suggestedNextByRoutineVersionCount: true,
        translationsCount: true,
    },
};