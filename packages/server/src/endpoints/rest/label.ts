import { endpointsLabel } from "@local/shared";
import { label_create, label_findMany, label_findOne, label_update } from "../generated";
import { LabelEndpoints } from "../logic/label";
import { setupRoutes } from "./base";

export const LabelRest = setupRoutes([
    [endpointsLabel.findOne, LabelEndpoints.Query.label, label_findOne],
    [endpointsLabel.findMany, LabelEndpoints.Query.labels, label_findMany],
    [endpointsLabel.createOne, LabelEndpoints.Mutation.labelCreate, label_create],
    [endpointsLabel.updateOne, LabelEndpoints.Mutation.labelUpdate, label_update],
]);
