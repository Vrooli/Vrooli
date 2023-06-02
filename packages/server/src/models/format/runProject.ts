import { RunProjectModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "RunProject" as const;
export const RunProjectFormat: Formatter<RunProjectModelLogic> = {
    gqlRelMap: {
        __typename,
        projectVersion: "ProjectVersion",
        schedule: "Schedule",
        steps: "RunProjectStep",
        user: "User",
        organization: "Organization",
    },
    prismaRelMap: {
        __typename,
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
