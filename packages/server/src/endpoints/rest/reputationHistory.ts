import { reputationHistory_findMany, reputationHistory_findOne } from "@local/shared";
import { ReputationHistoryEndpoints } from "../logic";
import { setupRoutes } from "./base";

export const ReputationHistoryRest = setupRoutes({
    "/reputationHistory/:id": {
        get: [ReputationHistoryEndpoints.Query.reputationHistory, reputationHistory_findOne],
    },
    "/reputationHistories": {
        get: [ReputationHistoryEndpoints.Query.reputationHistories, reputationHistory_findMany],
    },
});
