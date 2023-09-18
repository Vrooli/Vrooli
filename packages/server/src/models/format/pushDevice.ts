import { PushDeviceModelLogic } from "../base/types";
import { Formatter } from "../types";

export const PushDeviceFormat: Formatter<PushDeviceModelLogic> = {
    gqlRelMap: {
        __typename: "PushDevice",
    },
    prismaRelMap: {
        __typename: "PushDevice",
        user: "User",
    },
    countFields: {},
};
