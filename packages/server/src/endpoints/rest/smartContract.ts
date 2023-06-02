import { smartContract_create, smartContract_findMany, smartContract_findOne, smartContract_update } from "@local/shared";
import { SmartContractEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const SmartContractRest = setupRoutes({
    "/smartContract/:id": {
        get: [SmartContractEndpoints.Query.smartContract, smartContract_findOne],
        put: [SmartContractEndpoints.Mutation.smartContractUpdate, smartContract_update],
    },
    "/smartContracts": {
        get: [SmartContractEndpoints.Query.smartContracts, smartContract_findMany],
    },
    "/smartContract": {
        post: [SmartContractEndpoints.Mutation.smartContractCreate, smartContract_create],
    },
});
