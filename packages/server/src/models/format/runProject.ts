import { Formatter } from "../types";

const __typename = "RunProject" as const;
export const RunProjectFormat: Formatter<ModelRunProjectLogic> = {
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
