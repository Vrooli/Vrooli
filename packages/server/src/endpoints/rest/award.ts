import { endpointsAward } from "@local/shared";
import { award_findMany } from "../generated";
import { AwardEndpoints } from "../logic/award";
import { setupRoutes } from "./base";

export const AwardRest = setupRoutes([
    [endpointsAward.findMany, AwardEndpoints.Query.awards, award_findMany],
]);
