import { view_findMany } from "@local/shared";
import { ViewEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ViewRest = setupRoutes({
    "/views": {
        get: [ViewEndpoints.Query.views, view_findMany],
    },
});
