import { ResourceListModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "ResourceList" as const;
export const ResourceListFormat: Formatter<ResourceListModelLogic> = {
    gqlRelMap: {
        __typename,
        resources: "Resource",
        apiVersion: "ApiVersion",
        organization: "Organization",
        post: "Post",
        projectVersion: "ProjectVersion",
        routineVersion: "RoutineVersion",
        smartContractVersion: "SmartContractVersion",
        standardVersion: "StandardVersion",
        focusMode: "FocusMode",
    },
    prismaRelMap: {
        __typename,
        resources: "Resource",
        apiVersion: "ApiVersion",
        organization: "Organization",
        post: "Post",
        projectVersion: "ProjectVersion",
        routineVersion: "RoutineVersion",
        smartContractVersion: "SmartContractVersion",
        standardVersion: "StandardVersion",
        focusMode: "FocusMode",
    },
    countFields: {},
};
