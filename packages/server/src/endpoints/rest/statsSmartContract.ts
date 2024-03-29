import { statsSmartContract_findMany } from "../generated";
import { StatsSmartContractEndpoints } from "../logic/statsSmartContract";
import { setupRoutes } from "./base";

export const StatsSmartContractRest = setupRoutes({
    "/stats/smartContract": {
        get: [StatsSmartContractEndpoints.Query.statsSmartContract, statsSmartContract_findMany],
    },
});
