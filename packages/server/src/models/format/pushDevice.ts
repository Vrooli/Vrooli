import { PushDeviceModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "PushDevice" as const;
export const PushDeviceFormat: Formatter<PushDeviceModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        user: "User",
    },
    countFields: {},
};
