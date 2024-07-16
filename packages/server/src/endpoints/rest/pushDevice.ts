import { pushDevice_create, pushDevice_findMany, pushDevice_test, pushDevice_update } from "../generated";
import { PushDeviceEndpoints } from "../logic/pushDevice";
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
    "/pushDeviceTest/:id": {
        put: [PushDeviceEndpoints.Mutation.pushDeviceTest, pushDevice_test],
    },
});
