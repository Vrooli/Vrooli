import { RunProjectStepModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "RunProjectStep" as const;
export const RunProjectStepFormat: Formatter<RunProjectStepModelLogic> = {
    gqlRelMap: {
        __typename,
        directory: "ProjectVersionDirectory",
        run: "RunProject",
    },
    prismaRelMap: {
        __typename,
        directory: "ProjectVersionDirectory",
        runProject: "RunProject",
    },
    countFields: {},
};
