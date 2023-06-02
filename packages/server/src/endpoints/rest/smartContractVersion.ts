import { smartContractVersion_create, smartContractVersion_findMany, smartContractVersion_findOne, smartContractVersion_update } from "@local/shared";
import { SmartContractVersionEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const SmartContractVersionRest = setupRoutes({
    "/smartContractVersion/:id": {
        get: [SmartContractVersionEndpoints.Query.smartContractVersion, smartContractVersion_findOne],
        put: [SmartContractVersionEndpoints.Mutation.smartContractVersionUpdate, smartContractVersion_update],
    },
    "/smartContractVersions": {
        get: [SmartContractVersionEndpoints.Query.smartContractVersions, smartContractVersion_findMany],
    },
    "/smartContractVersion": {
        post: [SmartContractVersionEndpoints.Mutation.smartContractVersionCreate, smartContractVersion_create],
    },
});
