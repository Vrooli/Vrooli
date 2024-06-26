import { label_create, label_findMany, label_findOne, label_update } from "../generated";
import { LabelEndpoints } from "../logic/label";
import { setupRoutes } from "./base";

export const LabelRest = setupRoutes({
    "/label/:id": {
        get: [LabelEndpoints.Query.label, label_findOne],
        put: [LabelEndpoints.Mutation.labelUpdate, label_update],
    },
    "/labels": {
        get: [LabelEndpoints.Query.labels, label_findMany],
    },
    "/label": {
        post: [LabelEndpoints.Mutation.labelCreate, label_create],
    },
});
