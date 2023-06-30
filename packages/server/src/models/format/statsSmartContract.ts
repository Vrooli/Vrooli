import { StatsSmartContractModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "StatsSmartContract" as const;
export const StatsSmartContractFormat: Formatter<StatsSmartContractModelLogic> = {
    gqlRelMap: {
        __typename,
    },
    prismaRelMap: {
        __typename,
        smartContract: "SmartContract",
    },
    countFields: {},
};
