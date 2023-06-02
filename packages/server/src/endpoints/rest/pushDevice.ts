import { pushDevice_create, pushDevice_findMany, pushDevice_update } from "@local/shared";
import { PushDeviceEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const PushDeviceRest = setupRoutes({
    "/pushDevice": {
        post: [PushDeviceEndpoints.Mutation.pushDeviceCreate, pushDevice_create],
    },
    "/pushDevices": {
        get: [PushDeviceEndpoints.Query.pushDevices, pushDevice_findMany],
    },
    "/pushDevice/:id": {
        put: [PushDeviceEndpoints.Mutation.pushDeviceUpdate, pushDevice_update],
    },
});
