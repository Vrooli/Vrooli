import { StatsSmartContractModelLogic } from "../base/types";
import { Formatter } from "../types";

export const StatsSmartContractFormat: Formatter<StatsSmartContractModelLogic> = {
    gqlRelMap: {
        __typename: "StatsSmartContract",
    },
    prismaRelMap: {
        __typename: "StatsSmartContract",
        smartContract: "SmartContract",
    },
    countFields: {},
};
