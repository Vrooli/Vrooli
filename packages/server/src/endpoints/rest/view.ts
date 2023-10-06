import { view_findMany } from "../generated";
import { ViewEndpoints } from "../logic/view";
import { setupRoutes } from "./base";

export const ViewRest = setupRoutes({
    "/views": {
        get: [ViewEndpoints.Query.views, view_findMany],
    },
});
