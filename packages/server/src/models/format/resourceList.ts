import { ResourceListModelLogic } from "../base/types";
import { Formatter } from "../types";

export const ResourceListFormat: Formatter<ResourceListModelLogic> = {
    gqlRelMap: {
        __typename: "ResourceList",
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
        __typename: "ResourceList",
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
