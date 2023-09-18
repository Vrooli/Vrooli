import { RunProjectModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RunProjectFormat: Formatter<RunProjectModelLogic> = {
    gqlRelMap: {
        __typename: "RunProject",
        projectVersion: "ProjectVersion",
        schedule: "Schedule",
        steps: "RunProjectStep",
        user: "User",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename: "RunProject",
        projectVersion: "ProjectVersion",
        schedule: "Schedule",
        steps: "RunProjectStep",
        user: "User",
        organization: "Organization",
    },
    countFields: {
        stepsCount: true,
    },
};
