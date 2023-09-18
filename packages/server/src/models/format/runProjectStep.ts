import { RunProjectStepModelLogic } from "../base/types";
import { Formatter } from "../types";

export const RunProjectStepFormat: Formatter<RunProjectStepModelLogic> = {
    gqlRelMap: {
        __typename: "RunProjectStep",
        directory: "ProjectVersionDirectory",
        run: "RunProject",
    },
    prismaRelMap: {
        __typename: "RunProjectStep",
        directory: "ProjectVersionDirectory",
        runProject: "RunProject",
    },
    countFields: {},
};
