import { endpointsActions } from "@local/shared";
import { copy_copy } from "../generated";
import { CopyEndpoints } from "../logic/copy";
import { setupRoutes } from "./base";

export const CopyRest = setupRoutes([
    [endpointsActions.copy, CopyEndpoints.Mutation.copy, copy_copy],
]);
