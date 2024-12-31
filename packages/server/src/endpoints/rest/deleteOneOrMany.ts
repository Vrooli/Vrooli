import { endpointsActions } from "@local/shared";
import { deleteOneOrMany_deleteMany, deleteOneOrMany_deleteOne } from "../generated";
import { DeleteOneOrManyEndpoints } from "../logic/deleteOneOrMany";
import { setupRoutes } from "./base";

export const DeleteOneOrManyRest = setupRoutes([
    [endpointsActions.deleteOne, DeleteOneOrManyEndpoints.Mutation.deleteOne, deleteOneOrMany_deleteOne],
    [endpointsActions.deleteMany, DeleteOneOrManyEndpoints.Mutation.deleteMany, deleteOneOrMany_deleteMany],
]);
