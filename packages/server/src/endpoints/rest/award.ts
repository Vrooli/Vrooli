import { award_findMany } from "@local/shared";
import { AwardEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const AwardRest = setupRoutes({
    "/awards": {
        post: [AwardEndpoints.Query.awards, award_findMany],
    },
});
