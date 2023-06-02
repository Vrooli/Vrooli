import { statsSmartContract_findMany } from "@local/shared";
import { StatsSmartContractEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const StatsSmartContractRest = setupRoutes({
    "/stats/smartContract": {
        get: [StatsSmartContractEndpoints.Query.statsSmartContract, statsSmartContract_findMany],
    },
});
